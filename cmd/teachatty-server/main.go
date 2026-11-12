package main

import (
	"cmp"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"

	"github.com/daniel-cmdline/teachatty/internal/tlstrust"
	"github.com/daniel-cmdline/teachatty/internal/version"
)

// Upgrader instance that upgrades the HTTP connection to WebSocket
var upgrader = websocket.Upgrader{
	// browsers always send an Origin header on WebSocket connections, and the teachatty
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

	shutdownGracePeriod = 10 * time.Second
	closeWriteWait      = time.Second
)

var (
	mu    sync.Mutex // guards only the rooms map
	rooms = map[string]*Room{defaultRoom: {members: map[*websocket.Conn]string{}}}
)

func main() {
	// --addr: where to listen. ":9000" = port 9000 on all interfaces;
	// "10.0.0.5:7347" = only that interface (for example, only the internal network)
	addr := flag.String("addr", defaultAddr, "address to listen on, host:port (empty host = all interfaces)")
	flag.BoolVar(&noPipes, "no-pipes", false, "refuse clients in pipe mode (command | teachatty), e.g. to keep build logs out")
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
		fmt.Println("teachatty-server", version.String())
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

	fmt.Printf("teachatty-server listening on %s · default room: %s · %s\n", *addr, defaultRoom, security)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := newChatServer().serve(ctx, ln, shutdownGracePeriod); err != nil {
		log.Fatal(err)
	}
}

type chatServer struct {
	connections *connectionManager
	http        *http.Server
}

func newChatServer() *chatServer {
	return &chatServer{connections: newConnectionManager()}
}

func (s *chatServer) serve(ctx context.Context, ln net.Listener, grace time.Duration) error {
	s.http = &http.Server{Handler: http.HandlerFunc(s.handleWS)}
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.http.Serve(ln) }()

	var err error
	select {
	case err = <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	shutdownErr := s.shutdown(shutdownCtx)
	if err == nil {
		err = <-serveErr
	}
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	return err
}

func (s *chatServer) shutdown(ctx context.Context) error {
	s.connections.stopAccepting()
	err := s.http.Shutdown(ctx)
	if err != nil {
		_ = s.http.Close()
	}
	if s.connections.waitPending(ctx) != nil {
		s.connections.rejectPending()
	}

	connections := s.connections.snapshot()
	message := websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down")
	writesDone := make(chan struct{}, len(connections))
	for conn := range connections {
		go func(conn *websocket.Conn) {
			_ = conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(closeWriteWait))
			writesDone <- struct{}{}
		}(conn)
	}
	writeTimer := time.NewTimer(closeWriteWait)
	for i := 0; i < len(connections); i++ {
		select {
		case <-writesDone:
		case <-writeTimer.C:
			i = len(connections)
		}
	}
	writeTimer.Stop()
	if s.connections.waitEmpty(ctx) != nil {
		s.connections.forceClose()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), closeWriteWait)
		_ = s.connections.waitEmpty(cleanupCtx)
		cancel()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func (s *chatServer) handleWS(w http.ResponseWriter, r *http.Request) {
	// --no-pipes: the teachatty client declares pipe mode in the URL. A courtesy rule, not
	// a security boundary: a client that doesn't declare it isn't stopped (the rate limit
	// is what holds back abuse); it keeps teammates' build logs out of a chat room.
	if noPipes && r.URL.Query().Get("mode") == "pipe" {
		http.Error(w, "this server doesn't accept pipe mode (it was started with --no-pipes)", http.StatusForbidden)
		return
	}
	reservation, ok := s.connections.reserve()
	if !ok {
		http.Error(w, "server shutting down", http.StatusServiceUnavailable)
		return
	}
	defer reservation.release()

	// one IP can't hold more than --max-per-ip connections (see limits.go)
	release, ok := acquireIP(r)
	if !ok {
		http.Error(w, "too many connections from your address", http.StatusTooManyRequests)
		return
	}
	keepIP := false
	defer func() {
		if !keepIP {
			release()
		}
	}()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	if !reservation.register(conn) {
		_ = conn.Close()
		return
	}
	keepIP = true
	name := cmp.Or(r.URL.Query().Get("room"), defaultRoom)
	alias := sanitizeAlias(r.URL.Query().Get("alias"))
	var room *Room
	defer func() {
		if room != nil {
			leaveRoom(name, room, conn)
		}
		_ = conn.Close()
		release()
		s.connections.remove(conn)
	}()

	// a message over the limit: ReadMessage returns an error, gorilla answers with
	// close 1009 ("message too big") and this connection leaves the loop and the room like any other
	conn.SetReadLimit(maxMessage)

	// no room in the URL (old client, another program): goes to the default room
	// joining a room that doesn't exist creates it
	mu.Lock()
	room = rooms[name]
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
}

// leaveRoom removes a connection from its room and, when the room is now empty and
// still the one registered under that name, unregisters it (the default room is the
// exception: it lives as long as the server). The identity check is not paranoia:
// a handler goroutine can wake up long after its connection was dropped (by
// broadcast's 5-second write deadline, say) and the room replaced — unregistering
// by name alone would then erase the *new* room and silently split the new members
// from each other. Lock order: room.mu (inside leave) under mu, as everywhere.
func leaveRoom(name string, room *Room, conn *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()
	if room.leave(conn) && name != defaultRoom && rooms[name] == room {
		delete(rooms, name)
	}
}
