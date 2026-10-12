// Package ui is the client's screen: messages on top, input box at the bottom.
// Everything tview lives in here; main only knows New, Welcome, Notice, Print, Run and Stop.
package ui

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
)

// teachat's colors, from the logo: the neon mint cup and the amber speech bubble
const (
	mint  = "#72e3a6"
	amber = "#f2b04f"
)

// UI holds the tview application and the message area, which Print needs to reach
type UI struct {
	app      *tview.Application
	messages *tview.TextView
	// screen is the tcell screen, kept so a mention can ring the terminal bell.
	// Set in Run; only used from the event loop, which runs in Run's goroutine.
	screen tcell.Screen
	// mention matches "@alias" in other people's messages (whole word, any case)
	mention *regexp.Regexp
}

// alias colors. Mid tones, readable on both dark and light terminals.
var palette = []string{
	"#e06c75", "#e5a04b", "#d7c25a", "#98c379", "#4fd6a0", "#56b6c2",
	"#61afef", "#a3a0ff", "#c678dd", "#f08fc0", "#d19a66", "#7fb3d5",
}

// aliasColor picks the color from a hash of the alias: the same alias always gets the
// same color, on any client, without the server knowing anything about colors.
func aliasColor(alias string) string {
	h := fnv.New32a()
	h.Write([]byte(alias))
	return palette[h.Sum32()%uint32(len(palette))]
}

// format builds the line: a gray timestamp, the alias colored, then the text. With
// colors on, tview interprets tags like "[red]" in the text, so everything from outside
// goes through tview.Escape: without it, anyone could send "[red]" and change your screen.
// mine = a message you sent yourself: alias in bold, in the normal text color.
// mentioned = someone wrote your @alias: the text stands out in amber and bold.
func format(alias, text string, mine, mentioned bool) string {
	stamp := "[gray]" + time.Now().Format("15:04") + "[-] "
	style := "[" + aliasColor(alias) + "]"
	if mine {
		style = "[::b]"
	}
	body := tview.Escape(text)
	if mentioned {
		body = "[" + amber + "::b]" + body + "[-::-]"
	}
	return stamp + style + tview.Escape(alias) + "[-::-]: " + body
}

// New builds the screen. send is called on every Enter with the typed text.
func New(room, alias string, send func(text string)) *UI {
	// teachat's look, set before any widget is created (they copy these defaults):
	// the terminal's own background and text color, mint borders, amber titles
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.PrimaryTextColor = tcell.ColorDefault
	tview.Styles.BorderColor = tcell.GetColor(mint)
	tview.Styles.TitleColor = tcell.GetColor(amber)

	u := &UI{
		app: tview.NewApplication(),
		// "@alias" followed by anything that can't be part of a name (or the end)
		mention: regexp.MustCompile(`(?i)@` + regexp.QuoteMeta(alias) + `($|[^\w-])`),
	}

	// message area. By default tview's TextView does NOT follow new text:
	// once the screen fills up, new messages stay hidden below.
	// ScrollToEnd turns on "follow the end". PgUp turns it off, so you can read older messages.
	u.messages = tview.NewTextView()
	u.messages.SetBorder(true).SetTitle(" " + tview.Escape(room) + " · " + tview.Escape(alias) + " ")
	u.messages.ScrollToEnd()
	u.messages.SetDynamicColors(true) // turn on color tags, used for the alias

	// input box: TextArea wraps long lines instead of scrolling sideways.
	// No word wrap: breaking at any character means the number of lines
	// can be computed from the text width alone.
	input := tview.NewTextArea()
	input.SetBorder(true)
	input.SetWordWrap(false)
	// a gray hint while the box is empty; it goes away as soon as you type
	input.SetPlaceholder("type a message…")
	input.SetPlaceholderStyle(tcell.StyleDefault.Foreground(tcell.ColorGray))

	// messages take whatever is left; the box starts at 3 rows (1 of text + 2 of border)
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(u.messages, 0, 1, false).
		AddItem(input, 3, 0, true)
	u.app.SetRoot(layout, true)

	input.SetChangedFunc(func() {
		// typed anything: jump back to the newest messages and follow the end again
		u.messages.ScrollToEnd()

		// the box grows with the text, from 1 to 5 lines.
		// StringWidth counts screen columns, not bytes: "ã" takes 1, an emoji takes 2.
		// The +1 is the current line; when the text fills a line, the cursor moves to the next.
		_, _, width, _ := input.GetInnerRect()
		if width <= 0 {
			return // the box hasn't been drawn yet
		}
		lines := uniseg.StringWidth(input.GetText())/width + 1
		layout.ResizeItem(input, min(lines, 5)+2, 0)
		if lines <= 5 {
			// everything fits: reset the box's internal scroll to the top. Without this,
			// pasted text (which arrives before the box grows) leaves the first line hidden.
			input.SetOffset(0, 0)
		}
	})

	// intercept keys before the TextArea; returning nil = the TextArea never sees the key
	input.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyEnter:
			// Enter sends instead of inserting a newline
			text := input.GetText()
			if strings.TrimSpace(text) != "" {
				send(text)
				// your own message shows up like everyone else's: "alias: text"
				fmt.Fprintln(u.messages, format(alias, text, true, false))
			}
			input.SetText("", false)
			return nil
		case tcell.KeyPgUp, tcell.KeyPgDn:
			// focus always stays in the box; scrolling is forwarded to the messages
			u.messages.InputHandler()(ev, nil)
			return nil
		}
		return ev
	})

	return u
}

// cup is the teacup from the logo, in text: steam in amber, the cup in mint
var cup = []string{
	"[" + amber + "]   ( ([-]    ",
	"[" + mint + "]  .____.[-]  ",
	"[" + mint + "]  |    |][-] ",
	"[" + mint + "]  \\____/[-]  ",
}

// Welcome shows the teacup with up to four lines of information next to it (where
// you are, how the connection is secured, the keys). Like Notice, call it before Run.
func (u *UI) Welcome(info ...string) {
	for i, art := range cup {
		line := ""
		if i < len(info) {
			line = info[i]
		}
		fmt.Fprintln(u.messages, art+"  [gray]"+tview.Escape(line)+"[-]")
	}
	fmt.Fprintln(u.messages)
}

// Print shows a received line. Safe to call from another goroutine:
// QueueUpdateDraw hands the change to the screen's goroutine, which redraws.
func (u *UI) Print(line string) {
	// the server sends "alias: text" as a single string; the alias is whatever comes
	// before the first ": ". Without ": ", the whole line is shown uncolored.
	out, mentioned := tview.Escape(line), false
	if alias, text, ok := strings.Cut(line, ": "); ok {
		mentioned = u.mention.MatchString(text)
		out = format(alias, text, false, mentioned)
	}
	u.app.QueueUpdateDraw(func() {
		fmt.Fprintln(u.messages, out)
		// someone wrote your @alias: ring the terminal bell, so a terminal in the
		// background (another tab, another window) gets your attention
		if mentioned && u.screen != nil {
			u.screen.Beep()
		}
	})
}

// Notice shows a line from teachat itself (not from a person), in gray.
// Call it before Run: it writes directly instead of queueing, because QueueUpdate
// waits for the event loop, and before Run there's no loop yet (it would block forever).
func (u *UI) Notice(line string) {
	fmt.Fprintln(u.messages, "[gray]"+tview.Escape(line)+"[-]")
}

// Run takes over the terminal and only returns when the screen closes (Ctrl+C or Stop).
// By the time it returns, the terminal is back to normal.
func (u *UI) Run() error {
	// create the tcell screen ourselves instead of letting tview do it, only to keep a
	// reference for the bell (tview doesn't expose its screen). SetScreen initializes it.
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	u.screen = screen
	u.app.SetScreen(screen)
	return u.app.Run()
}

// Stop closes the screen. Safe to call from another goroutine.
func (u *UI) Stop() {
	u.app.Stop()
}
