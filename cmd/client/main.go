package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
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

	// recebe mensagens em paralelo enquanto você digita
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Fatal("conexão encerrada")
			}
			fmt.Println(string(msg))
		}
	}()

	// cada Enter envia a linha
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		conn.WriteMessage(websocket.TextMessage, sc.Bytes())
	}
	if err := sc.Err(); err != nil {
		log.Fatal("erro lendo o teclado: ", err)
	}
}
