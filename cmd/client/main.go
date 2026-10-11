package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("uso: client ws://HOST:23456/ws")
		os.Exit(1)
	}

	url := os.Args[1]

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
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
			fmt.Println(">", string(msg))
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
