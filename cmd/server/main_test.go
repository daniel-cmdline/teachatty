package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// connect sobe o handleWS num servidor de teste e conecta nele, como o cliente faria.
// query é o que vai depois do "?": "room=dev&alias=ana", ou "" para não mandar nada.
func connect(t *testing.T, srv *httptest.Server, query string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?" + query
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

// members conta quem está na sala agora; -1 se a sala não existe
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

// eventually espera a condição virar verdade: entrar e sair acontecem na goroutine
// do servidor, um pouco depois do Dial ou do Close aqui no teste.
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

	conn := connect(t, srv, "alias=ana") // sem room
	eventually(t, "ana joins local", func() bool { return members(defaultRoom) == 1 })

	conn.Close()
	eventually(t, "ana leaves local", func() bool { return members(defaultRoom) == 0 })
	// vazia, mas continua existindo (0, não -1)
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
	bia := connect(t, srv, "room=local&alias=bia") // pedir "local" explicitamente dá no mesmo
	defer bia.Close()
	eventually(t, "both join", func() bool { return members(defaultRoom) == 2 })

	if err := ana.WriteMessage(websocket.TextMessage, []byte("oi")); err != nil {
		t.Fatal(err)
	}
	bia.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := bia.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "ana: oi" {
		t.Fatalf("bia got %q, want %q", msg, "ana: oi")
	}
}
