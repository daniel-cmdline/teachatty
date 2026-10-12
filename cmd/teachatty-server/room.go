package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// writeWait is how long one member gets to take a message. A member that doesn't read
// (a laptop that went to sleep, a network that dropped without closing the connection, a
// teachatty pipe suspended with Ctrl+Z, a badly written client) fills its network buffers and then
// blocks every write to it; without a limit, broadcast would wait forever while holding
// the room's lock, and the whole room would freeze. A var, not a const, so the tests can
// shorten it.
var writeWait = 5 * time.Second

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
		if c == myOwnConn {
			continue
		}
		c.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			// too slow or already gone: drop it so the room keeps going. Closing the
			// connection also wakes its own goroutine (blocked in ReadMessage), which then
			// leaves the room the normal way; deleting it here (allowed while ranging over
			// a map) stops the next broadcasts from waiting on it again.
			delete(r.members, c)
			c.Close()
		}
	}
}
