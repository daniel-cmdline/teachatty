package main

import (
	"cmp"
	"flag"
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

const (
	// porta em que o servidor escuta, em todas as interfaces
	addr = ":7347"

	// sala padrão: nasce junto com o servidor e nunca é apagada, mesmo vazia.
	// Quem conecta sem dizer a sala cai nela, então todo mundo se encontra sem combinar nada.
	defaultRoom = "local"
)

var (
	mu    sync.Mutex // protege só o mapa de salas
	rooms = map[string]*Room{defaultRoom: {members: map[*websocket.Conn]string{}}}
)

func main() {
	// ainda não há flags, mas o Parse faz o -h mostrar a ajuda em vez de subir o servidor
	flag.Usage = usage
	flag.Parse()
	checkArgs()

	http.HandleFunc("/ws", handleWS)
	fmt.Printf("teachat-server listening on %s · default room: %s\n", addr, defaultRoom)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// handleWS atende cada conexão em /ws, na goroutine que o net/http cria para ela.
// Tem nome (em vez de ser uma função anônima no main) para os testes conseguirem usá-lo.
func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	// sem sala na URL (cliente antigo, outro programa): vai para a sala padrão
	name := cmp.Or(r.URL.Query().Get("room"), defaultRoom)
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

	// sai da sala; sala vazia deixa de existir, menos a padrão
	mu.Lock()
	if room.leave(conn) && name != defaultRoom {
		delete(rooms, name)
	}
	mu.Unlock()
}
