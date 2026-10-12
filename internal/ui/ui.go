// Package ui é a tela do cliente: mensagens em cima, caixa de digitação embaixo.
// Tudo que é tview fica aqui dentro; o main só conhece New, Print, Run e Stop.
package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
)

// UI guarda a aplicação tview e a área de mensagens, que o Print precisa alcançar
type UI struct {
	app      *tview.Application
	messages *tview.TextView
}

// New monta a tela. send é chamada a cada Enter com o texto digitado.
func New(room, alias string, send func(text string)) *UI {
	u := &UI{app: tview.NewApplication()}

	// área de mensagens. Por padrão o TextView do tview NÃO acompanha texto novo:
	// quando a tela enche, as mensagens novas ficam escondidas embaixo.
	// ScrollToEnd liga o "seguir o fim". PgUp desliga, para dar para ler o passado.
	u.messages = tview.NewTextView()
	u.messages.SetBorder(true).SetTitle(" " + room + " · " + alias + " ")
	u.messages.ScrollToEnd()

	// caixa de digitação: TextArea quebra linhas longas em vez de rolar para o lado.
	// Sem quebra por palavra: quebrando em qualquer caractere, dá para calcular
	// quantas linhas o texto ocupa só pela largura dele.
	input := tview.NewTextArea()
	input.SetBorder(true)
	input.SetWordWrap(false)

	// mensagens ocupam o que sobrar; a caixa começa com 3 linhas (1 de texto + 2 de borda)
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(u.messages, 0, 1, false).
		AddItem(input, 3, 0, true)
	u.app.SetRoot(layout, true)

	input.SetChangedFunc(func() {
		// digitou qualquer coisa: volta para as mensagens mais novas e segue o fim de novo
		u.messages.ScrollToEnd()

		// a caixa cresce com o texto, de 1 até 5 linhas.
		// StringWidth conta colunas na tela, não bytes: "ã" ocupa 1, um emoji ocupa 2.
		// O +1 é a linha atual; quando o texto enche a linha, o cursor já pula para a próxima.
		_, _, width, _ := input.GetInnerRect()
		if width <= 0 {
			return // a caixa ainda não foi desenhada
		}
		lines := uniseg.StringWidth(input.GetText())/width + 1
		layout.ResizeItem(input, min(lines, 5)+2, 0)
		if lines <= 5 {
			// cabe tudo: volta a rolagem interna da caixa para o topo. Sem isso, texto
			// colado (que chega antes da caixa crescer) deixa a primeira linha escondida.
			input.SetOffset(0, 0)
		}
	})

	// intercepta teclas antes do TextArea; devolver nil = o TextArea não vê a tecla
	input.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyEnter:
			// Enter envia em vez de pular linha
			text := input.GetText()
			if strings.TrimSpace(text) != "" {
				send(text)
				// sua própria mensagem aparece igual às dos outros: "alias: texto"
				fmt.Fprintf(u.messages, "%s: %s\n", alias, text)
			}
			input.SetText("", false)
			return nil
		case tcell.KeyPgUp, tcell.KeyPgDn:
			// o foco fica sempre na caixa; a rolagem é repassada para as mensagens
			u.messages.InputHandler()(ev, nil)
			return nil
		}
		return ev
	})

	return u
}

// Print mostra uma linha recebida. Pode ser chamada de outra goroutine:
// QueueUpdateDraw entrega a mudança para a goroutine da tela, que redesenha.
func (u *UI) Print(line string) {
	u.app.QueueUpdateDraw(func() {
		fmt.Fprintln(u.messages, line)
	})
}

// Run toma conta do terminal e só retorna quando a tela fecha (Ctrl+C ou Stop).
// Quando retorna, o terminal já voltou ao normal.
func (u *UI) Run() error {
	return u.app.Run()
}

// Stop fecha a tela. Pode ser chamada de outra goroutine.
func (u *UI) Stop() {
	u.app.Stop()
}
