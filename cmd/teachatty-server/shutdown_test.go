package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func startChatServer(t *testing.T, grace time.Duration) (*chatServer, net.Listener, context.CancelFunc, <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	app := newChatServer()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		defer close(done)
		done <- app.serve(ctx, ln, grace)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("server did not stop")
		}
	})
	return app, ln, cancel, done
}

func connectAddr(t *testing.T, ln net.Listener, query string) *websocket.Conn {
	t.Helper()
	url := "ws://" + ln.Addr().String() + "/ws?" + query
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestGracefulShutdownClosesAndCleansUp(t *testing.T) {
	room := fmt.Sprintf("shutdown-%d", time.Now().UnixNano())
	app, ln, stop, done := startChatServer(t, time.Second)
	conn := connectAddr(t, ln, "room="+room+"&alias=ana")
	defer conn.Close()

	clientRead := make(chan error, 1)
	go func() {
		_, _, err := conn.ReadMessage()
		clientRead <- err
	}()
	stop()

	select {
	case err := <-clientRead:
		closeErr, ok := err.(*websocket.CloseError)
		if !ok || closeErr.Code != websocket.CloseGoingAway || closeErr.Text != "server shutting down" {
			t.Fatalf("got close error %v, want close 1001 server shutting down", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("client did not receive shutdown close frame")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := members(room); got != -1 {
		t.Fatalf("room still has %d members after shutdown", got)
	}
	app.connections.mu.Lock()
	active := len(app.connections.active)
	app.connections.mu.Unlock()
	if active != 0 {
		t.Fatalf("manager still tracks %d connections", active)
	}
}

func TestReservationRejectedAfterShutdownBegins(t *testing.T) {
	manager := newConnectionManager()
	manager.stopAccepting()
	if _, ok := manager.reserve(); ok {
		t.Fatal("reservation succeeded after shutdown began")
	}
}

func TestPendingUpgradeIsIncludedInShutdownTracking(t *testing.T) {
	manager := newConnectionManager()
	reservation, ok := manager.reserve()
	if !ok {
		t.Fatal("initial reservation failed")
	}
	manager.stopAccepting()

	type result struct {
		connections map[*websocket.Conn]struct{}
		err         error
	}
	started := make(chan struct{})
	finished := make(chan result, 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() {
		close(started)
		err := manager.waitPending(ctx)
		finished <- result{connections: manager.snapshot(), err: err}
	}()
	<-started

	conn := &websocket.Conn{}
	if !reservation.register(conn) {
		t.Fatal("pending upgrade was rejected before forced shutdown")
	}
	got := <-finished
	if got.err != nil {
		t.Fatal(got.err)
	}
	if _, ok := got.connections[conn]; !ok {
		t.Fatal("upgraded connection escaped shutdown snapshot")
	}
}

func TestShutdownForceClosesClientThatDoesNotReply(t *testing.T) {
	const grace = 80 * time.Millisecond
	app, ln, stop, done := startChatServer(t, grace)
	conn := connectAddr(t, ln, "room=unresponsive")
	defer conn.Close()
	stop() // no client read: its close frame is not acknowledged
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not force-close the unresponsive client")
	}
	app.connections.mu.Lock()
	active := len(app.connections.active)
	app.connections.mu.Unlock()
	if active != 0 {
		t.Fatalf("force-close left %d connections tracked", active)
	}
	app.connections.mu.Lock()
	forced := app.connections.forced
	app.connections.mu.Unlock()
	if !forced {
		t.Fatal("shutdown did not enter forced-close mode after the grace period")
	}
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, _, err := conn.ReadMessage()
	if err == nil {
		t.Fatal("client connection remained open after forced shutdown")
	}
	if strings.Contains(err.Error(), "i/o timeout") {
		t.Fatalf("client connection was not force-closed: %v", err)
	}
}

func TestShutdownWithNoConnectionsAndRepeatedCleanup(t *testing.T) {
	app, _, stop, done := startChatServer(t, time.Second)
	stop()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := app.shutdown(ctx); err != nil {
		t.Fatalf("repeated shutdown failed: %v", err)
	}
	app.connections.remove(nil)
	app.connections.forceClose()
}

func TestShutdownReleasesPerIPSlot(t *testing.T) {
	room := fmt.Sprintf("shutdown-ip-%d", time.Now().UnixNano())
	app, ln, stop, done := startChatServer(t, time.Second)
	ipMu.Lock()
	before := perIP["127.0.0.1"]
	ipMu.Unlock()
	conn := connectAddr(t, ln, "room="+room)
	clientRead := make(chan error, 1)
	go func() {
		_, _, err := conn.ReadMessage()
		clientRead <- err
	}()
	stop()
	<-clientRead
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := members(room); got != -1 {
		t.Fatalf("room still has %d members after shutdown", got)
	}
	app.connections.mu.Lock()
	active := len(app.connections.active)
	app.connections.mu.Unlock()
	if active != 0 {
		t.Fatalf("manager still tracks %d connections", active)
	}

	ipMu.Lock()
	got := perIP["127.0.0.1"]
	ipMu.Unlock()
	if got != before {
		conn.Close()
		t.Fatalf("per-IP count after shutdown is %d, want prior count %d", got, before)
	}
	conn.Close()
}
