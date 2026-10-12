package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
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
