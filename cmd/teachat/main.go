package main

import (
	"bytes"
	"cmp"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"golang.org/x/term"

	"github.com/daniel-cmdline/teachat/internal/tlstrust"
	"github.com/daniel-cmdline/teachat/internal/ui"
	"github.com/daniel-cmdline/teachat/internal/version"
)

func main() {
	// flag.String returns pointers (&value)
	server := flag.String("server", "localhost:7347", "teachat server address, host:port")
	room := flag.String("room", "local", "room to join; created if it doesn't exist")
	// default alias: your system user ($USER). cmp.Or returns the first non-empty
	// value, so without $USER (cron, some containers) it is "anonymous".
	alias := flag.String("alias", cmp.Or(os.Getenv("USER"), "anonymous"), "your name in the chat")
	noTLS := flag.Bool("no-tls", false, "connect over plain ws:// without encryption")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = usage
	flag.Parse()
	checkArgs()
	if *showVersion {
		fmt.Println("teachat", version.String())
		return
	}

	// stdin is not a terminal (command | teachat): pipe mode, no screen
	pipeMode := !term.IsTerminal(int(os.Stdin.Fd()))

	// build wss://SERVER/ws?room=...&alias=... from the flags (ws:// with --no-tls)
	// dereference the pointer to get the string
	scheme := "wss"
	if *noTLS {
		scheme = "ws"
	}
	query := url.Values{"room": {*room}, "alias": {*alias}}
	// pipe mode says so, so a server started with --no-pipes can refuse it
	if pipeMode {
		query.Set("mode", "pipe")
	}
	u := url.URL{
		Scheme:   scheme,
		Host:     *server,
		Path:     "/ws",
		RawQuery: query.Encode(),
	}

	// TLS trust (trusted on first use, then pinned) lives in internal/tlstrust.
	// A notice like "trusting new server ..." goes into a buffer and is shown below,
	// once we know whether it goes to the screen or to stderr.
	var notice bytes.Buffer
	dialer := *websocket.DefaultDialer
	if !*noTLS {
		cfg, err := tlstrust.Client(*server, &notice)
		if err != nil {
			log.Fatal(err)
		}
		dialer.TLSClientConfig = cfg
	}
	conn, resp, err := dialer.Dial(u.String(), nil)
	if err != nil {
		// the server refuses some connections on purpose, before the WebSocket handshake
		// (403: pipes not allowed; 429: too many connections from your IP), and says why
		// in the response body: show that reason instead of a generic "bad handshake"
		if resp != nil && (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests) {
			reason, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			log.Fatal("the server refused the connection: ", strings.TrimSpace(string(reason)))
		}
		log.Fatal(tlstrust.DialHint(err, !*noTLS))
	}
	defer conn.Close()

	if pipeMode {
		os.Stderr.Write(notice.Bytes())
		pipe(conn)
		return
	}

	// the screen calls this on every Enter; only the screen's goroutine writes to the
	// connection, so gorilla's one-writer-per-connection rule still holds
	screen := ui.New(*room, *alias, func(text string) {
		conn.WriteMessage(websocket.TextMessage, []byte(text))
	})
	if notice.Len() > 0 {
		screen.Notice(strings.TrimSpace(notice.String()))
	}

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
