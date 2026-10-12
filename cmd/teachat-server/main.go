package main

import (
	"cmp"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/daniel-cmdline/teachat/internal/tlstrust"
	"github.com/daniel-cmdline/teachat/internal/version"
)

// Upgrader instance that upgrades the HTTP connection to WebSocket
var upgrader = websocket.Upgrader{
	// browsers always send an Origin header on WebSocket connections, and the teachat
	// client never does. Refusing any request that has one keeps web pages (opened in a
	// browser on the same network) from connecting to the server behind the user's back.
	CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == "" },
}

const (
	// default address: port 7347 on all interfaces. Change it with --addr.
	defaultAddr = ":7347"

	// largest accepted message, in bytes. Without a limit, someone sends 1 GB and the
	// server tries to hold it all in memory before relaying. 1 MB easily fits any typed
	// message and almost any log line in pipe mode (stack traces, JSON).
	maxMessage = 1 << 20

	// default room: created with the server and never deleted, even when empty.
	// Anyone who connects without naming a room lands in it, so people meet without arranging anything.
	defaultRoom = "local"
)

var (
	mu    sync.Mutex // guards only the rooms map
	rooms = map[string]*Room{defaultRoom: {members: map[*websocket.Conn]string{}}}
)

func main() {
	// --addr: where to listen. ":9000" = port 9000 on all interfaces;
	// "10.0.0.5:7347" = only that interface (for example, only the internal network)
	addr := flag.String("addr", defaultAddr, "address to listen on, host:port (empty host = all interfaces)")
	flag.BoolVar(&noPipes, "no-pipes", false, "refuse clients in pipe mode (command | teachat), e.g. to keep build logs out")
	flag.IntVar(&maxPerIP, "max-per-ip", maxPerIP, "max simultaneous connections from one IP address (0 = no limit, e.g. behind a company NAT)")
	noTLS := flag.Bool("no-tls", false, "serve plain ws:// without encryption (for testing)")
	// --tls-cert/--tls-key: the administrator's own certificate (Let's Encrypt, company CA)
	// instead of the self-signed one; clients then verify it like a browser does
	certFile := flag.String("tls-cert", "", "certificate file (PEM), e.g. Let's Encrypt's fullchain.pem; default: self-signed")
	keyFile := flag.String("tls-key", "", "private key file (PEM) for --tls-cert, e.g. privkey.pem")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = usage
	flag.Parse()
	checkArgs()
	if *showVersion {
		fmt.Println("teachat-server", version.String())
		return
	}
	if *noTLS && (*certFile != "" || *keyFile != "") {
		log.Fatal("--no-tls can't be combined with --tls-cert/--tls-key")
	}

	// open the port before announcing: with a busy port or a bad address,
	// the error shows up on its own, with no misleading "listening" first
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	// TLS is on by default; everything about certificates lives in internal/tlstrust
	security := "no TLS (plain ws://)"
	if !*noTLS {
		cfg, desc, err := tlstrust.Server(*certFile, *keyFile, os.Stderr)
		if err != nil {
			log.Fatal(err)
		}
		ln = tls.NewListener(ln, cfg)
		security = "TLS · " + desc
	}

	http.HandleFunc("/ws", handleWS)
	fmt.Printf("teachat-server listening on %s · default room: %s · %s\n", *addr, defaultRoom, security)
	log.Fatal(http.Serve(ln, nil))
}

// handleWS serves each connection on /ws, in the goroutine net/http creates for it.
// It's a named function (not an anonymous one inside main) so the tests can use it.
func handleWS(w http.ResponseWriter, r *http.Request) {
	// --no-pipes: the teachat client declares pipe mode in the URL. A courtesy rule, not
	// a security boundary: a client that doesn't declare it isn't stopped (the rate limit
	// is what holds back abuse); it keeps teammates' build logs out of a chat room.
	if noPipes && r.URL.Query().Get("mode") == "pipe" {
		http.Error(w, "this server doesn't accept pipe mode (it was started with --no-pipes)", http.StatusForbidden)
		return
	}

	// one IP can't hold more than --max-per-ip connections (see limits.go)
	release, ok := acquireIP(r)
	if !ok {
		http.Error(w, "too many connections from your address", http.StatusTooManyRequests)
		return
	}
	defer release()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	// a message over the limit: ReadMessage returns an error, gorilla answers with
	// close 1009 ("message too big") and this connection leaves the loop and the room like any other
	conn.SetReadLimit(maxMessage)

	// no room in the URL (old client, another program): goes to the default room
	name := cmp.Or(r.URL.Query().Get("room"), defaultRoom)
	alias := r.URL.Query().Get("alias")

	// joining a room that doesn't exist creates it
	mu.Lock()
	room := rooms[name]
	if room == nil {
		room = &Room{members: map[*websocket.Conn]string{}}
		rooms[name] = room
	}
	room.join(conn, alias)
	mu.Unlock()

	msgs, bytes := limiters()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// over its budget, this connection waits here before the next read: nothing is
		// dropped, the sender is just slowed down (see limits.go)
		if msgs.Wait(r.Context()) != nil || bytes.WaitN(r.Context(), len(msg)) != nil {
			break
		}
		// relay the message to the others in the same room
		room.broadcast(conn, []byte(alias+": "+string(msg)))
	}

	// leave the room; an empty room stops existing, except the default one
	mu.Lock()
	if room.leave(conn) && name != defaultRoom {
		delete(rooms, name)
	}
	mu.Unlock()
}
