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
	mu      sync.Mutex
	clients = map[*websocket.Conn]bool{}
)

func main() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		mu.Lock()
		clients[conn] = true
		mu.Unlock()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			// repassa a mensagem para todos os outros conectados
			mu.Lock()
			for c := range clients {
				if c != conn {
					c.WriteMessage(websocket.TextMessage, msg)
				}
			}
			mu.Unlock()
		}

		mu.Lock()
		clients[conn] = false
		mu.Unlock()
	})

	fmt.Println("Servidor iniciado na porta 23456")
	log.Fatal(http.ListenAndServe(":23456", nil))
}
