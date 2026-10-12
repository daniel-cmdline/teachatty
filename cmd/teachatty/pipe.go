package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// batching in pipe mode: how long to wait for more lines after the first one, and the
// largest batch. 100 ms is unnoticeable for a human reading the room; 64 KB keeps each
// message far below the server's 1 MB limit.
const (
	batchWait = 100 * time.Millisecond
	batchMax  = 64 << 10
)

// pipe is the screenless mode (command | teachatty): it sends each stdin line to the room,
// echoes the line to the terminal (like tee) and exits when the input ends.
// It never opens the screen on purpose: in a script or cron there's nobody to press Ctrl+C.
func pipe(conn *websocket.Conn) {
	// the room's messages keep arriving and must be read, even if not shown.
	// Without this, the network buffers fill up and the server's broadcast stalls the
	// whole room waiting on us. This goroutine is the only reader; pipe is the only writer.
	// readErr records why reading stopped. It is written before close(done), so whoever
	// sees done closed also sees the right value (same guarantee as "closed" in main.go).
	done := make(chan struct{})
	var readErr error
	// closing is set right before we send our own close frame at the end, so the reader
	// can tell "we're finishing" from "the connection dropped while we wait for input"
	var closing atomic.Bool
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				readErr = err
				close(done)
				// dropped while main may be blocked waiting for the next input line (a quiet
				// tail -f, or after being suspended and dropped by the server): exit now
				// instead of hanging until a line arrives. No screen here, nothing to restore.
				if !closing.Load() {
					log.Fatal("connection closed: ", err)
				}
				return
			}
		}
	}()
	// the connection dropped or the server refused something (e.g. a line over the limit,
	// close 1009): writes still "work" because they go to a buffer, so the reader is what tells us
	dropped := func() {
		log.Fatal("connection closed: ", readErr)
	}

	// send writes one message, reporting the real reason if the connection is gone
	send := func(text string) {
		select {
		case <-done:
			dropped()
		default:
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
			// the real reason usually arrives through the reader (e.g. close 1009, line
			// too big); wait for it briefly before showing the write error
			select {
			case <-done:
				dropped()
			case <-time.After(time.Second):
				log.Fatal("connection closed: ", err)
			}
		}
	}

	// stdin is read in its own goroutine, so the loop below can wait for "the next line
	// or a short pause, whichever comes first". ReadString has no line length limit
	// (bufio.Scanner gives up at 64 KB).
	lines := make(chan string)
	go func() {
		r := bufio.NewReader(os.Stdin)
		for {
			line, err := r.ReadString('\n')
			if line != "" {
				lines <- strings.TrimRight(line, "\r\n")
			}
			if err != nil {
				if err != io.EOF {
					log.Fatal("error reading input: ", err)
				}
				close(lines)
				return
			}
		}
	}()

	// batching: lines that arrive within batchWait of the first one go out together, as
	// one multi-line message of up to batchMax bytes. A 5,000-line build log becomes a
	// handful of messages instead of flooding the room (and the server's rate limit).
	var batch []string
	size := 0
	flush := func() {
		if len(batch) > 0 {
			send(strings.Join(batch, "\n"))
			batch, size = nil, 0
		}
	}
	timer := time.NewTimer(batchWait)
	timer.Stop()
	for open := true; open; {
		select {
		case line, ok := <-lines:
			if !ok {
				open = false
				break
			}
			fmt.Println(line) // echo: you still see the command's output
			if strings.TrimSpace(line) == "" {
				continue // an empty line is echoed, but not sent to the room
			}
			if size > 0 && size+len(line) > batchMax {
				flush()
			}
			if len(batch) == 0 {
				timer.Reset(batchWait)
			}
			batch = append(batch, line)
			size += len(line) + 1
		case <-timer.C:
			flush()
		}
	}
	flush()

	// close properly: send the close frame and wait for the server to answer (then the
	// reader goroutine ends and closes done). Exiting without this can lose the last lines.
	closing.Store(true)
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	select {
	case <-done:
		// a normal closure (1000) is the answer to our close; anything else is a problem
		if !websocket.IsCloseError(readErr, websocket.CloseNormalClosure) {
			dropped()
		}
	case <-time.After(2 * time.Second):
	}
}
