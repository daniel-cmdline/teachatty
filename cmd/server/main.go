package main

import (
	"cmp"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Upgrader instance that upgrades the HTTP connection to WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // accepts any origin
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
	flag.Usage = usage
	flag.Parse()
	checkArgs()

	// open the port before announcing: with a busy port or a bad address,
	// the error shows up on its own, with no misleading "listening" first
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/ws", handleWS)
	fmt.Printf("teachat-server listening on %s · default room: %s\n", *addr, defaultRoom)
	log.Fatal(http.Serve(ln, nil))
}

// handleWS serves each connection on /ws, in the goroutine net/http creates for it.
// It's a named function (not an anonymous one inside main) so the tests can use it.
func handleWS(w http.ResponseWriter, r *http.Request) {
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

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
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
