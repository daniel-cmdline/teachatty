package main

import (
	"cmp"
	"flag"
	"log"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
	"golang.org/x/term"

	"github.com/daniel-cmdline/teachat/internal/ui"
)

func main() {
	// flag.String returns pointers (&value)
	server := flag.String("server", "localhost:7347", "teachat server address, host:port")
	room := flag.String("room", "local", "room to join; created if it doesn't exist")
	// default alias: your system user ($USER). cmp.Or returns the first non-empty
	// value, so without $USER (cron, some containers) it is "anonymous".
	alias := flag.String("alias", cmp.Or(os.Getenv("USER"), "anonymous"), "your name in the chat")
	flag.Usage = usage
	flag.Parse()
	checkArgs()

	// monta ws://SERVER/ws?room=...&alias=... a partir das flags
	// dereference the pointer to get the string
	u := url.URL{
		Scheme:   "ws",
		Host:     *server,
		Path:     "/ws",
		RawQuery: url.Values{"room": {*room}, "alias": {*alias}}.Encode(),
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// stdin is not a terminal (command | teachat): pipe mode, no screen
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		pipe(conn)
		return
	}

	// the screen calls this on every Enter; only the screen's goroutine writes to the
	// connection, so gorilla's one-writer-per-connection rule still holds
	screen := ui.New(*room, *alias, func(text string) {
		conn.WriteMessage(websocket.TextMessage, []byte(text))
	})

	// empty channel to signal that the connection dropped; the receiving goroutine closes it
	closed := make(chan struct{})
	var readErr error // why it dropped; written before close(closed), read after

	// receive messages in parallel while you type
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				// log.Fatal can't be used here: it exits without restoring the terminal.
				// Tell main and close the screen; main is the one that exits.
				readErr = err
				close(closed)
				screen.Stop()
				return
			}
			screen.Print(string(msg))
		}
	}()

	// Run owns the terminal until Ctrl+C or until the connection drops
	if err := screen.Run(); err != nil {
		log.Fatal(err)
	}

	// the terminal is back to normal; now it's safe to report that the connection dropped
	select {
	case <-closed:
		log.Fatal("connection closed: ", readErr)
	default:
	}
}
