package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

//Instancia do Upgrader para atualizar a conexão HTTP para WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // aceita qualquer origem
}

var (
	mu    sync.Mutex // protege só o mapa de salas
	rooms = map[string]*Room{}
)

func main() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		name := r.URL.Query().Get("room")
		alias := r.URL.Query().Get("alias")

		// entrar numa sala que não existe cria a sala
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
			// repassa a mensagem para os outros da mesma sala
			room.broadcast(conn, []byte(alias+": "+string(msg)))
		}

		// sai da sala; sala vazia deixa de existir
		mu.Lock()
		if room.leave(conn) {
			delete(rooms, name)
		}
		mu.Unlock()
	})

	fmt.Println("Servidor iniciado na porta 23456")
	log.Fatal(http.ListenAndServe(":23456", nil))
}
