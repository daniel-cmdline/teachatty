package main

import (
	"cmp"
	"flag"
	"log"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
	"golang.org/x/term"

	"github.com/daniel-cmdline/teachat/internal/ui"
)

func main() {
	//flag.String retorna ponteiros (&value) 
	server := flag.String("server", "localhost:7347", "teachat server address, host:port")
	room := flag.String("room", "local", "room to join; created if it doesn't exist")
	// alias padrão: o seu usuário do sistema ($USER). cmp.Or devolve o primeiro
	// valor não vazio, então sem $USER (cron, alguns containers) fica "anonimo".
	alias := flag.String("alias", cmp.Or(os.Getenv("USER"), "anonimo"), "your name in the chat")
	flag.Usage = usage
	flag.Parse()

	// monta ws://SERVER/ws?room=...&alias=... a partir das flags
	//deferencia o valor do ponteiro para obter a string
	u := url.URL{
		Scheme:   "ws",
		Host:     *server,
		Path:     "/ws",
		RawQuery: url.Values{"room": {*room}, "alias": {*alias}}.Encode(),
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// stdin não é um terminal (comando | teachat): modo pipe, sem tela
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		pipe(conn)
		return
	}

	// a tela chama esta função a cada Enter; só a goroutine da tela escreve na
	// conexão, então a regra da gorilla de um escritor por conexão continua valendo
	screen := ui.New(*room, *alias, func(text string) {
		conn.WriteMessage(websocket.TextMessage, []byte(text))
	})

	// canal vazio para avisar que a conexão caiu; a goroutine de recebimento fecha o canal
	closed := make(chan struct{})
	// recebe mensagens em paralelo enquanto você digita
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				// não dá para usar log.Fatal aqui: ele sai sem restaurar o terminal.
				// Avisa o main e fecha a tela; quem encerra é o main.
				close(closed)
				screen.Stop()
				return
			}
			screen.Print(string(msg))
		}
	}()

	// Run toma conta do terminal até o Ctrl+C ou até a conexão cair
	if err := screen.Run(); err != nil {
		log.Fatal(err)
	}

	// o terminal já voltou ao normal; agora dá para avisar se foi a conexão que caiu
	select {
	case <-closed:
		log.Fatal("conexão encerrada")
	default:
	}
}
