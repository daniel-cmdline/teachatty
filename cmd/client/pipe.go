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
	// readErr guarda por que a leitura parou. É escrito antes do close(done), então quem
	// vê done fechado já enxerga o valor certo (mesma garantia do "closed" no main.go).
	done := make(chan struct{})
	var readErr error
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				readErr = err
				close(done)
				return
			}
		}
	}()
	// a conexão caiu ou o servidor recusou algo (ex.: linha acima do limite, close 1009):
	// a escrita ainda "funciona" porque vai para o buffer, então quem avisa é a leitura
	dropped := func() {
		log.Fatal("connection closed: ", readErr)
	}

	// ReadString não tem limite de tamanho de linha (o bufio.Scanner para em 64 KB)
	r := bufio.NewReader(os.Stdin)
	for {
		line, err := r.ReadString('\n')
		if line != "" {
			line = strings.TrimRight(line, "\r\n")
			fmt.Println(line) // eco: você continua vendo a saída do comando
			// linha vazia aparece no eco, mas não vai para a sala
			if strings.TrimSpace(line) != "" {
				select {
				case <-done:
					dropped()
				default:
				}
				if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
					// o motivo de verdade costuma chegar pela leitura (ex.: close 1009, linha
					// grande demais); espera ela um instante antes de mostrar o erro da escrita
					select {
					case <-done:
						dropped()
					case <-time.After(time.Second):
						log.Fatal("connection closed: ", err)
					}
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("error reading input: ", err)
		}
	}

	// fecha direito: manda o frame de close e espera o servidor responder (aí a goroutine
	// de leitura termina e fecha done). Sair sem isso pode perder as últimas linhas.
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	select {
	case <-done:
		// fechamento normal (1000) é a resposta ao nosso close; qualquer outro é problema
		if !websocket.IsCloseError(readErr, websocket.CloseNormalClosure) {
			dropped()
		}
	case <-time.After(2 * time.Second):
	}
}
