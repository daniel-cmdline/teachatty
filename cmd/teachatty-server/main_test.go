package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

// connect connects to handleWS running on a test server, the way the client would.
// query is what goes after the "?": "room=dev&alias=ana", or "" to send nothing.
func connect(t *testing.T, srv *httptest.Server, query string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?" + query
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

// members counts who is in the room right now; -1 if the room doesn't exist
func members(name string) int {
	mu.Lock()
	defer mu.Unlock()
	room := rooms[name]
	if room == nil {
		return -1
	}
	room.mu.Lock()
	defer room.mu.Unlock()
	return len(room.members)
}

// eventually waits for the condition to become true: joining and leaving happen in the
// server's goroutine, a little after the Dial or Close here in the test.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("did not happen: " + what)
}

func TestLocalRoomExistsAtStart(t *testing.T) {
	if members(defaultRoom) != 0 {
		t.Fatalf("room %q should exist, empty, before any connection", defaultRoom)
	}
}

func TestNoRoomLandsInLocalAndLocalSurvivesEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()

	conn := connect(t, srv, "alias=ana") // no room
	eventually(t, "ana joins local", func() bool { return members(defaultRoom) == 1 })

	conn.Close()
	eventually(t, "ana leaves local", func() bool { return members(defaultRoom) == 0 })
	// empty, but still exists (0, not -1)
}

func TestOtherRoomsVanishWhenEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()

	conn := connect(t, srv, "room=dev&alias=ana")
	eventually(t, "room dev is created", func() bool { return members("dev") == 1 })

	conn.Close()
	eventually(t, "room dev vanishes", func() bool { return members("dev") == -1 })
}

func TestMessageInLocalRoom(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()

	ana := connect(t, srv, "alias=ana")
	defer ana.Close()
	bia := connect(t, srv, "room=local&alias=bia") // asking for "local" explicitly is the same
	defer bia.Close()
	eventually(t, "both join", func() bool { return members(defaultRoom) == 2 })

	if err := ana.WriteMessage(websocket.TextMessage, []byte("hi")); err != nil {
		t.Fatal(err)
	}
	bia.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := bia.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "ana: hi" {
		t.Fatalf("bia got %q, want %q", msg, "ana: hi")
	}
}

// size limit: up to maxMessage gets through; above it, only the sender's connection is closed
func TestMessageAtLimitPasses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()

	ana := connect(t, srv, "room=limit&alias=ana")
	defer ana.Close()
	bia := connect(t, srv, "room=limit&alias=bia")
	defer bia.Close()
	eventually(t, "both join", func() bool { return members("limit") == 2 })

	if err := ana.WriteMessage(websocket.TextMessage, make([]byte, maxMessage)); err != nil {
		t.Fatal(err)
	}
	bia.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := bia.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if want := len("ana: ") + maxMessage; len(msg) != want {
		t.Fatalf("bia got %d bytes, want %d", len(msg), want)
	}
}

func TestMessageTooBigClosesOnlyThatConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()

	ana := connect(t, srv, "room=big&alias=ana")
	defer ana.Close()
	bia := connect(t, srv, "room=big&alias=bia")
	defer bia.Close()
	eventually(t, "both join", func() bool { return members("big") == 2 })

	// 1 byte over the limit
	if err := ana.WriteMessage(websocket.TextMessage, make([]byte, maxMessage+1)); err != nil {
		t.Fatal(err)
	}
	ana.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := ana.ReadMessage(); !websocket.IsCloseError(err, websocket.CloseMessageTooBig) {
		t.Fatalf("ana got %v, want close 1009 (message too big)", err)
	}
	eventually(t, "ana leaves the room", func() bool { return members("big") == 1 })

	// the room keeps working, and bia never received the big message
	carol := connect(t, srv, "room=big&alias=carol")
	defer carol.Close()
	eventually(t, "carol joins", func() bool { return members("big") == 2 })
	if err := carol.WriteMessage(websocket.TextMessage, []byte("hi")); err != nil {
		t.Fatal(err)
	}
	bia.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := bia.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "carol: hi" {
		t.Fatalf("bia got %.40q, want %q", msg, "carol: hi")
	}
}

// a member that never reads is dropped after writeWait, and the room keeps going
func TestStuckMemberIsDroppedAndTheRoomKeepsGoing(t *testing.T) {
	old := writeWait
	writeWait = 200 * time.Millisecond
	defer func() { writeWait = old }()
	noRateLimit(t) // this test pushes 20 MB on purpose

	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()

	stuck := connect(t, srv, "room=stuck&alias=stuck") // connects and never reads
	defer stuck.Close()
	ana := connect(t, srv, "room=stuck&alias=ana")
	defer ana.Close()
	bia := connect(t, srv, "room=stuck&alias=bia")
	defer bia.Close()
	eventually(t, "three join", func() bool { return members("stuck") == 3 })

	// bia reads everything in the background, like a normal client
	got := make(chan int, 1)
	go func() {
		n := 0
		bia.SetReadDeadline(time.Now().Add(10 * time.Second))
		for {
			if _, _, err := bia.ReadMessage(); err != nil {
				got <- n
				return
			}
			if n++; n == 40 {
				got <- n
				return
			}
		}
	}()

	// 40 x 512 KB = 20 MB: far more than the stuck member's network buffers can hold.
	// Sent from a goroutine: if the room freezes, ana blocks too, and the timeout below
	// still fires instead of the test hanging.
	go func() {
		msg := make([]byte, 512<<10)
		for i := 0; i < 40; i++ {
			if ana.WriteMessage(websocket.TextMessage, msg) != nil {
				return
			}
		}
	}()
	select {
	case n := <-got:
		if n != 40 {
			t.Fatalf("bia got %d of 40 messages", n)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the room froze: bia stopped receiving")
	}
	eventually(t, "the stuck member is dropped", func() bool { return members("stuck") == 2 })
}

// browsers send an Origin header; the teachatty client doesn't
func TestBrowserOriginIsRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?alias=web"
	_, resp, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {"https://evil.example"}})
	if err == nil {
		t.Fatal("a connection with an Origin header should be refused")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %v", resp)
	}
}

// noRateLimit lifts the per-connection rate limits for the rest of a test
func noRateLimit(t *testing.T) {
	oldMsg, oldBurst, oldBytes := msgRate, msgBurst, byteRate
	msgRate, msgBurst, byteRate = rate.Inf, 1<<20, rate.Inf
	t.Cleanup(func() { msgRate, msgBurst, byteRate = oldMsg, oldBurst, oldBytes })
}

// over the rate limit, messages are slowed down, never dropped
func TestFloodIsSlowedDownNotDropped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()

	ana := connect(t, srv, "room=flood&alias=ana")
	defer ana.Close()
	bia := connect(t, srv, "room=flood&alias=bia")
	defer bia.Close()
	eventually(t, "both join", func() bool { return members("flood") == 2 })

	// burst of 20 goes through at once; the other 10 at 5 a second take ~2 s
	const total = 30
	start := time.Now()
	go func() {
		for i := 0; i < total; i++ {
			if ana.WriteMessage(websocket.TextMessage, []byte("spam")) != nil {
				return
			}
		}
	}()
	bia.SetReadDeadline(time.Now().Add(10 * time.Second))
	for i := 0; i < total; i++ {
		if _, _, err := bia.ReadMessage(); err != nil {
			t.Fatalf("bia got only %d of %d messages: %v", i, total, err)
		}
	}
	if took := time.Since(start); took < 1500*time.Millisecond {
		t.Fatalf("%d messages arrived in %v: the rate limit isn't slowing the sender down", total, took)
	}
}

func TestConnectionsPerIPAreCapped(t *testing.T) {
	old := maxPerIP
	maxPerIP = 2
	defer func() { maxPerIP = old }()

	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()

	a := connect(t, srv, "alias=a")
	b := connect(t, srv, "alias=b")
	defer b.Close()

	// the third one from the same IP is refused with 429
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?alias=c"
	if _, resp, err := websocket.DefaultDialer.Dial(url, nil); err == nil || resp == nil || resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for the third connection, got %v %v", resp, err)
	}

	// once one leaves, there's room again
	a.Close()
	eventually(t, "a slot frees up", func() bool {
		c, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			return false
		}
		c.Close()
		return true
	})
}

func TestNoPipesRefusesPipeModeOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()
	base := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?alias=ci"

	// by default, pipe mode is welcome
	c, _, err := websocket.DefaultDialer.Dial(base+"&mode=pipe", nil)
	if err != nil {
		t.Fatalf("pipe mode refused without --no-pipes: %v", err)
	}
	c.Close()

	noPipes = true
	defer func() { noPipes = false }()

	// with --no-pipes: a pipe is refused with 403 and the reason in the body
	_, resp, err := websocket.DefaultDialer.Dial(base+"&mode=pipe", nil)
	if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for pipe mode, got %v %v", resp, err)
	}
	reason, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(reason), "doesn't accept pipe mode") {
		t.Fatalf("the refusal should say why, got %q", reason)
	}

	// and someone typing still gets in
	c, _, err = websocket.DefaultDialer.Dial(base, nil)
	if err != nil {
		t.Fatalf("an interactive client was refused: %v", err)
	}
	c.Close()
}

// A connection whose handler goroutine runs late (dropped by broadcast, then the last
// other member leaves, then a new connection re-creates the room under the same name)
// must not unregister the *new* room when it finally cleans up, or the new members
// would be silently split from each other. Regression for that race.
//
// The ordering is fully deterministic: every step of the interleaving is driven by the
// test itself (mirroring what broadcast and the handlers do), while ana's and bia's
// real handler goroutines stay parked in ReadMessage — their connections are only
// closed at the end, after every assertion. No sleeps, no hoping for a race.
func TestStaleRoomCleanupKeepsReplacedRoom(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()
	// The rooms registry persists between the tests of one -count run; a name that no
	// two runs share keeps this sequence untouched by leftovers.
	name := fmt.Sprintf("reborn-%d", time.Now().UnixNano())
	// Handler goroutines of a faster previous run may still be exiting and holding
	// their per-IP slot for a moment; start every run with a clean per-IP slate.
	// (A late release decrements a missing key, which the guard deletes: harmless.)
	ipMu.Lock()
	perIP = map[string]int{}
	ipMu.Unlock()
	t.Cleanup(func() {
		ipMu.Lock()
		perIP = map[string]int{}
		ipMu.Unlock()
	})

	// ana joins and creates the room; capture its *Room while she is the only member.
	ana := connect(t, srv, "room="+name+"&alias=ana")
	eventually(t, "ana joins", func() bool { return members(name) == 1 })
	mu.Lock()
	oldRoom := rooms[name]
	mu.Unlock()

	// bia joins normally.
	bia := connect(t, srv, "room="+name+"&alias=bia")
	eventually(t, "bia joins", func() bool { return members(name) == 2 })
	mu.Lock()
	sameRoom := rooms[name] == oldRoom
	mu.Unlock()
	if !sameRoom {
		t.Fatal("the room was unexpectedly replaced while its members were still in it")
	}

	// The member map is keyed by the server side of each connection; the dialer-side
	// view is a different object. Find them by alias (unique in this test).
	var anaConn, biaConn *websocket.Conn
	oldRoom.mu.Lock()
	for c, a := range oldRoom.members {
		switch a {
		case "ana":
			anaConn = c
		case "bia":
			biaConn = c
		}
	}
	oldRoom.mu.Unlock()
	if anaConn == nil || biaConn == nil {
		t.Fatal("could not find the server-side connections for ana and bia")
	}

	// Step 1: broadcast finds ana too slow and does what room.go does: remove her from
	// members and close her connection. Her handler goroutine is "delayed" and stays
	// parked in ReadMessage meanwhile, so it does not clean up yet.
	oldRoom.mu.Lock()
	delete(oldRoom.members, anaConn)
	oldRoom.mu.Unlock()

	// Step 2: bia leaves. ana is already gone from members, so this is the final
	// removal: the room is empty and correctly unregistered under this identity.
	leaveRoom(name, oldRoom, biaConn)
	mu.Lock()
	_, still := rooms[name]
	mu.Unlock()
	if still {
		t.Fatal("the emptied room should have been unregistered")
	}

	// Step 3: cris joins. A brand-new Room is registered under the same name.
	cris := connect(t, srv, "room="+name+"&alias=cris")
	defer cris.Close()
	eventually(t, "cris joins the new room", func() bool { return members(name) == 1 })
	mu.Lock()
	newRoom := rooms[name]
	mu.Unlock()
	if newRoom == oldRoom {
		t.Fatal("cris should have re-created the room, not joined the old one")
	}

	// Step 4: ana's delayed handler finally cleans up with the stale room pointer.
	// The identity check must refuse to unregister cris's room.
	leaveRoom(name, oldRoom, anaConn)
	mu.Lock()
	stillNew := rooms[name] == newRoom
	mu.Unlock()
	if !stillNew {
		t.Fatal("the stale cleanup deleted cris's newly created room")
	}

	// Step 5: members of the surviving room exchange messages normally. Both handler
	// goroutines for ana and bia are still parked (their connections were never closed),
	// so what the members() helper sees is exactly cris and daniel.
	daniel := connect(t, srv, "room="+name+"&alias=daniel")
	eventually(t, "daniel joins cris's room", func() bool { return members(name) == 2 })

	if err := cris.WriteMessage(websocket.TextMessage, []byte("ola")); err != nil {
		t.Fatal(err)
	}
	daniel.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := daniel.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "cris: ola" {
		t.Fatalf("daniel got %q, want %q", msg, "cris: ola")
	}

	if err := daniel.WriteMessage(websocket.TextMessage, []byte("hey")); err != nil {
		t.Fatal(err)
	}
	cris.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err = cris.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "daniel: hey" {
		t.Fatalf("cris got %q, want %q", msg, "daniel: hey")
	}

	// Everything asserted; now it is safe to let the delayed handlers wake up. Their
	// late cleanups run against a replaced room and must leave it alone. Close every
	// connection and wait for the room to be fully gone, so the next -count run starts
	// after all handler goroutines of this one have released their resources.
	ana.Close()
	bia.Close()
	cris.Close()
	daniel.Close()
	eventually(t, "the room is fully gone after everyone left", func() bool {
		return members(name) == -1
	})
}
