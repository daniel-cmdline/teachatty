package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Room has its own mutex: one room broadcasting doesn't block the others
type Room struct {
	mu      sync.Mutex
	members map[*websocket.Conn]string // connection -> alias
}

func (r *Room) join(conn *websocket.Conn, alias string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.members[conn] = alias
}

// leave removes the connection and reports whether the room is now empty
func (r *Room) leave(conn *websocket.Conn) (empty bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.members, conn)
	return len(r.members) == 0
}

func (r *Room) broadcast(myOwnConn *websocket.Conn, msg []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for c := range r.members {
		if c != myOwnConn {
			c.WriteMessage(websocket.TextMessage, msg)
		}
	}
}
