package main

import (
	"cmp"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

//Instancia do Upgrader para atualizar a conexão HTTP para WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // aceita qualquer origem
}

const (
	// endereço padrão: porta 7347 em todas as interfaces. Muda com --addr.
	defaultAddr = ":7347"

	// maior mensagem aceita, em bytes. Sem limite, alguém manda 1 GB e o servidor tenta
	// guardar tudo na memória antes de repassar. 1 MB passa folgado qualquer mensagem
	// digitada e quase toda linha de log no modo pipe (stack trace, JSON).
	maxMessage = 1 << 20

	// sala padrão: nasce junto com o servidor e nunca é apagada, mesmo vazia.
	// Quem conecta sem dizer a sala cai nela, então todo mundo se encontra sem combinar nada.
	defaultRoom = "local"
)

var (
	mu    sync.Mutex // protege só o mapa de salas
	rooms = map[string]*Room{defaultRoom: {members: map[*websocket.Conn]string{}}}
)

func main() {
	// --addr: onde escutar. ":9000" = porta 9000 em todas as interfaces;
	// "10.0.0.5:7347" = só nessa interface (por exemplo, só a rede interna)
	addr := flag.String("addr", defaultAddr, "address to listen on, host:port (empty host = all interfaces)")
	flag.Usage = usage
	flag.Parse()
	checkArgs()

	// abre a porta antes de anunciar: com a porta ocupada ou o endereço errado,
	// o erro aparece sozinho, sem um "listening" enganoso antes
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/ws", handleWS)
	fmt.Printf("teachat-server listening on %s · default room: %s\n", *addr, defaultRoom)
	log.Fatal(http.Serve(ln, nil))
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

	// mensagem maior que o limite: o ReadMessage devolve erro, a gorilla responde com
	// close 1009 ("message too big") e esta conexão sai do loop e da sala como qualquer outra
	conn.SetReadLimit(maxMessage)

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
