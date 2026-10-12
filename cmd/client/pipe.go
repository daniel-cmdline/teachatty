package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// pipe é o modo sem tela (comando | teachat): envia cada linha do stdin para a sala,
// mostra a linha no terminal (como o tee) e sai quando a entrada acaba.
// Não abre a tela de propósito: num script ou no cron não tem ninguém para apertar Ctrl+C.
func pipe(conn *websocket.Conn) {
	// as mensagens da sala continuam chegando e precisam ser lidas, mesmo sem mostrar.
	// Sem isso, os buffers de rede enchem e o broadcast do servidor trava a sala inteira
	// esperando por nós. Esta goroutine é a única que lê; o pipe é o único que escreve.
	done := make(chan struct{})
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				close(done)
				return
			}
		}
	}()

	// ReadString não tem limite de tamanho de linha (o bufio.Scanner para em 64 KB)
	r := bufio.NewReader(os.Stdin)
	for {
		line, err := r.ReadString('\n')
		if line != "" {
			line = strings.TrimRight(line, "\r\n")
			fmt.Println(line) // eco: você continua vendo a saída do comando
			// linha vazia aparece no eco, mas não vai para a sala
			if strings.TrimSpace(line) != "" {
				if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
					log.Fatal("conexão encerrada: ", err)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("erro lendo a entrada: ", err)
		}
	}

	// fecha direito: manda o frame de close e espera o servidor responder (aí a goroutine
	// de leitura termina e fecha done). Sair sem isso pode perder as últimas linhas.
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}
