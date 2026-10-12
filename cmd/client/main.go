package main

import (
	"flag"
	"log"
	"net/url"

	"github.com/gorilla/websocket"

	"github.com/daniel-cmdline/teachat/internal/ui"
)

func main() {
	//flag.String retorna ponteiros (&value) 
	server := flag.String("server", "localhost:23456", "endereço do servidor teachat")
	room := flag.String("room", "geral", "sala para entrar")
	alias := flag.String("alias", "anonimo", "seu nome no chat")
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

	// a tela chama esta função a cada Enter; só a goroutine da tela escreve na
	// conexão, então a regra da gorilla de um escritor por conexão continua valendo
	screen := ui.New(*room, *alias, func(text string) {
		conn.WriteMessage(websocket.TextMessage, []byte(text))
	})

	// recebe mensagens em paralelo enquanto você digita
	closed := make(chan struct{})
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
