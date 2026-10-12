# Features

Log of shipped features, newest first. Add an entry with every commit.

## 2026-10-12 13:05 — Everything in English

- Code comments, the site and documentation (`docs/index.html`, `docs/docs.html`), `roadmap.md` and this log are now in English, like the terminal output already was. Older entries below were translated; quoted strings that used to be in Portuguese (`conexão encerrada`, `teachat-server escutando em ...`) are kept as they were, since that's what the program printed back then.
- Test data changed from `"oi"` to `"hi"`.
- `gofmt` is now clean across the repo: the two older comments missing a space after `//` were fixed while translating them.
- Code shown in `docs.html` was regenerated from the translated source files.

## 2026-10-12 12:07 — `--addr` on the server and a 1 MB message limit

- **`--addr`:** the server listens wherever you tell it, as `host:port`. `--addr :9000` = port 9000 on all interfaces; `--addr 10.0.0.5:7347` = only on the internal network. Default `:7347`. The server's `-h` now lists its flags (read from the flags themselves, as in the client) and has examples.
- The server opens the port **before** announcing `listening on ...`. Previously, with a busy port or a bad address, it announced first and failed afterwards. Now it exits right away with the error (`address already in use`, `missing port in address`) and code 1.
- **1 MB limit per message** (`conn.SetReadLimit`, constant `maxMessage`). Without a limit, someone could send 1 GB and the server would try to hold it all in memory. Over the limit, gorilla answers close 1009 (`message too big`) and only that connection leaves; the room carries on. 1 MB rather than 64 KB so large log lines in pipe mode don't break.
- **Pipe mode now notices when the connection drops or the server refuses a line.** Previously a 2 MB line was refused and the pipe exited with code 0, as if it had worked: writes go to a buffer and "succeed", and only the reader goroutine saw the close. Now it records the error (`readErr`), the pipe checks it before each line and at the end, and exits with `connection closed: websocket: close 1009 (message too big)` and code 1.
- Full screen also shows why the connection dropped: `connection closed: ` + the error.
- **New tests:** a message exactly at the limit gets through; 1 byte over closes only that connection with 1009, and the room keeps working. Checked that the test fails without `SetReadLimit`. 6 tests now, all with `-race`.
- **Tested end to end:** `--addr :9000` and `--addr 127.0.0.1:9001` (loopback only, checked with `ss`), busy port, bad address, a 2 MB pipe line (refused, code 1), 900 KB (gets through), server dying in the middle of a slow pipe (code 1 with the reason).
- **Docs:** README (server flags section), roadmap (`--addr` leaves the next steps), and the HTML: flags and limit in the server reference, `--addr` in "Getting started" and on the site, new exit codes, the "no size limit" limitation removed, code excerpts regenerated.

## 2026-10-12 11:45 — Help when a command does nothing

- A stray argument (`teachat dev`, `teachat-server start`) was silently ignored: the server started and the client tried to connect. Now both print `unknown argument: ...` and the help, and exit with code 2 without doing anything.
- `teachat help` and `teachat-server help` show the help with code 0.
- An unknown flag (`--romm`) already showed the help, through Go's `flag` package.
- Code: `checkArgs()` in each binary's `usage.go`, called right after `flag.Parse()`.
- Docs: exit code table and server reference in `docs.html`, README and roadmap.

## 2026-10-12 11:43 — Terminal output in English

- Everything the program prints to the terminal moved to English: `connection closed` (previously `conexão encerrada`), `error reading input`, the server's startup line (`teachat-server listening on :7347 · default room: local`), and test names and messages.
- The alias for users without `$USER` changed from `anonimo` to `anonymous`, since it shows up in the chat.
- Code comments, the site and documentation (`docs/`), and these notes stayed in Portuguese at this point.
- The site and documentation were adjusted to quote the new messages, and the client code excerpts were regenerated.

## 2026-10-12 11:38 — Port 7347, -h help and first tests

- **Port 7347** on the server and as the client default (previously 23456). The port became the `addr` constant on the server.
- **`-h` on both binaries**, in English: what it is, how to use it, flags with defaults, keys and examples. Flag descriptions went to English too, since they show up in the help. In the client, defaults come from the flags themselves (`flag.Lookup`/`VisitAll`), so the help follows any change. The server didn't read flags at all: `server -h` started the server. Now it shows the help and exits.
- The server's startup message states the port and default room: `teachat-server escutando em :7347 · sala padrão: local` (translated to English in the next entry).
- The `/ws` handler moved out of `main` into the `handleWS` function, so the tests can use it. No behavior change.
- **First automated tests** (`cmd/server/main_test.go`, `go test ./...`): the `local` room exists before any connection; connecting without a room lands in it and it survives empty; other rooms disappear when they empty; a message in `local` arrives as `ana: oi`. They run with `-race`. Checked that they catch bugs: removing the `local` protection in `leave` makes a test fail.
- **Tested end to end** with the binaries: a client with no flags opens on `local · del`; another client in `local` receives the message; `echo ... | teachat` with no flags reaches `local`; everyone leaves, someone joins again, and `local` is still there.
- **Docs:** README, roadmap and the HTML (port, default room `local`, `-h`, tests in the repository map, and the server code regenerated from `handleWS`).

## 2026-10-12 11:29 — Default room `local`

- The server creates the `local` room when it starts, and it's never deleted, even when empty. Other rooms keep appearing and disappearing as before.
- Without `--room`, the client joins `local` (previously `geral`). On the server, a connection with no room in the URL also lands in `local`, instead of a room with an empty name.
- **Why:** just run `teachat` and everyone on the same server is in the same room, without agreeing on a name.
- **Why permanent:** creating the room at server start would be pointless if it got deleted on the first departure.
- In the code: the `defaultRoom` constant, the `rooms` map starts with it, `cmp.Or` for the empty name, and `leave` doesn't delete the default room.
- **Tested:** a client without `--room` opens with the title `local · ana`; a raw connection with no room receives its messages; leaving and coming back keeps the room working; room `dev` is unchanged. No races on the server.
- **Docs (HTML):** updated in the next entry.

## 2026-10-12 11:14 — Site and documentation up to date

- Clears the pending HTML updates for alias colors, the default alias and pipe mode.
- **Site (`docs/index.html`):** the screen mockup shows aliases in their real colors (computed with the same hash as the client) and the screen's owner in bold. New highlight "Each person gets a color". The pipes card lost its "coming soon" tag and gained an example; "Start in a minute" shows `uptime | teachat`; pipe mode left "What's next".
- **Documentation (`docs/docs.html`):**
  - Client reference: `--alias` defaulting to `$USER`, colors in the message area, pipe mode exit codes and a new **Pipe mode** subsection (usage, echo, one message per line, no size limit, safe in scripts). Linked from the sidebar.
  - Client internals: new subsections **Colors per alias** (hash, bold, `tview.Escape`, the palette) and **Pipe mode internals** (the two goroutines, why it reads the room, the close at the end, the flooded-room test). Code shown was regenerated from the current files, with the whole `pipe.go`.
  - Architecture: `pipe.go` in the repository map, `golang.org/x/term` in the dependencies, updated line count (~380).
  - Limitations: "pipe without a terminal fails" removed, "large logs flood the room" added; "slow client" now with the measured number (2,082 of 5,000).

## 2026-10-12 10:58 — Pipe mode

- When stdin is not a terminal (`command | teachat`), the client doesn't open the screen: it sends each line to the room, echoes it to the terminal (like `tee`) and exits when the input ends. Code in `cmd/client/pipe.go`; `main.go` only gained `term.IsTerminal`.
- `tail -f app.log | teachat --room ops` works as a continuous stream, until Ctrl+C.
- An empty line is echoed but not sent to the room. A last line without `\n` is still sent.
- No line length limit: `bufio.Reader.ReadString` instead of `bufio.Scanner`, which gives up at 64 KB.
- **Clean exit:** at the end it sends the WebSocket close frame and waits for the server's answer (up to 2 s), so the last lines aren't lost. Exit code 0; failing to connect or the connection dropping midway: 1, with the error on stderr.
- **Why it reads the room without showing anything:** other people's messages keep arriving. Without reading, the network buffers fill up and the server's broadcast stalls the whole room waiting on the pipe. One goroutine reads and discards everything.
- **Tested:** 2,000 lines (`seq`) all arrive, including the last; a 200 KB line arrives whole; with the server down it prints `connection refused` and exits 1. With a room flooded by 5,000 messages of 2 KB and a slow pipe connected, all 5,000 reached the others. With a client that never reads in the pipe's place, the room stalled at 2,082 of 5,000: the "slow client stalls the room" limitation, now measured.
- `golang.org/x/term` went from an indirect to a direct dependency (it was already in the project).
- **Docs (HTML):** updated on 2026-10-12.

## 2026-10-12 10:06 — Default alias is your user

- Without `--alias`, the client uses your system user (`$USER`) instead of `anonimo`. Without `$USER` set (cron, some containers), it stays `anonimo` (later renamed `anonymous`).
- **Why:** almost nobody needs to type `--alias` anymore. One line (`cmp.Or`), no dependency.
- **Docs (HTML):** updated on 2026-10-12.

## 2026-10-12 09:50 — Colors per alias

- In the message area, each person's alias shows in its own color, picked by hashing the alias (FNV-32a) into a 12-color palette. The same alias always gets the same color, on any client, without changing the server or the protocol.
- Your own messages: alias in bold, in the normal text color. This applies only to lines you sent, not to the alias: if someone else joins with your alias, they show up colored, not bold.
- Only the alias is colored; the message text stays in the normal color.
- **Security:** to color text, tview now interprets tags like `[red]`. Everything that comes from outside (alias, text, room name in the title) goes through `tview.Escape`, so whoever sends `[red]oi` still sees and shows a literal `[red]oi`.
- **Limits:** with 12 colors, collisions happen (5 people in a room: ~60% chance two share a color). The client finds the alias by splitting the line at the first `": "`, because the server sends `alias: text` as a single string; an alias containing `": "` is split wrongly. Both go away once messages are structured (JSON) in the protocol.
- Tested live with 5 clients in the same room: different colors per alias, bold only on your own messages, `[red]oi` shown literally.
- **Docs (HTML):** updated on 2026-10-12.

## 2026-10-12 09:40 — Site and documentation split

- The single page `teachat-por-dentro.html` became two, in `docs/`, with shared CSS in `docs/style.css`:
  - `index.html`, the site: what teachat is, the screen, the principles, the measured numbers, the animation of how a message travels, how to get started and what's next. A **docs** button in the top bar and the hero.
  - `docs.html`, the documentation, with a sidebar: overview, getting started, client reference (flags, keys, exit codes) and server reference (endpoint, message format, room lifecycle), architecture, protocol, server and client internals, capacity, limitations and roadmap.
- **Why:** the single page mixed presentation with reference. People getting to know the project and people looking up how something works want different things.
- The docs are now in git. The `.gitignore`, which only ignored the old page, was deleted.
- New limitations documented: the server accepts WebSocket from any origin and doesn't limit message size.

## 2026-10-12 09:26 — Full-screen client (tview)

- The client became a full screen built with `tview`, in the new `internal/ui` package: messages on top, in an area titled `room · alias`, and an input box at the bottom.
- **No more overwriting:** an incoming message no longer writes over what you're typing. The terminal is in raw mode and only the screen's goroutine draws. The receiving goroutine hands each message over with `QueueUpdateDraw`.
- Enter sends. Your own message shows up as `alias: text`, like everyone else's. The server still doesn't echo a message back to its sender; the client shows yours locally.
- The message area follows the newest messages. PgUp and PgDn scroll. PgUp stops following, so you can read older messages without being pulled back. Typing anything jumps back to the end.
- The input box grows with the text, from 1 to 5 lines, and goes back to 1 after Enter. It breaks lines at any character, not just between words, because that way the height comes from one calculation: text width on screen ÷ box width.
- Ctrl+C quits (previously Ctrl+D). If the connection drops, the screen closes, the terminal goes back to normal and only then does `conexão encerrada` appear. The previous `log.Fatal` would have exited without restoring the terminal.
- The client's `main.go` only knows `ui.New`, `Print`, `Run` and `Stop`. Everything tview lives in `internal/ui`.
- **Bug found in the live test:** tview's `TextView` doesn't follow new text by default. Once the screen filled up, new messages stayed hidden until you pressed PgDn. Fixed with `ScrollToEnd()`.
- **New dependencies:** `rivo/tview` v0.42.0, `gdamore/tcell/v2` v2.8.1, `rivo/uniseg` (text width on screen) and their indirect dependencies. All pure Go: the client is still a single binary.
- **Broke:** `echo oi | teachat` fails, because tview requires a terminal. It comes back with the line-by-line fallback, the first step of pipe mode.
- No mouse, on purpose: with the mouse on, selecting and copying chat text would require Shift.

## 2026-10-12 08:43 — Module renamed to `github.com/daniel-cmdline/teachat`

- `go.mod` changed from `module httpEndpoint` to `module github.com/daniel-cmdline/teachat`, the same path as the repository.
- **Why:** the module name is the start of every internal import. With our own packages arriving (`internal/ui`), imports become `github.com/daniel-cmdline/teachat/internal/...`. And `go install github.com/daniel-cmdline/teachat/...@latest` only works when the module name matches the repository path.
- **Why now:** today it's one line. Once packages exist, renaming means touching every import.
- No behavior change.

## 2026-10-11 10:28 — Per-room lock

- `Room` moved to `cmd/server/room.go`, with its own mutex and the `join`, `leave` and `broadcast` methods.
- The global mutex now guards only the `rooms` map. A broadcast locks only its own room.
- **Why:** with a single global lock, a broadcast in a large room blocked all the others.
- **Result:** with 50,000 clients in one room receiving messages, latency in a small room dropped from ~265 ms to ~0.1 ms. The race detector finds no problems.

## 2026-10-11 09:45 — Rooms and aliases

- The client takes the `--server`, `--room` and `--alias` flags (defaults: `localhost:23456`, `geral`, `anonimo`) instead of a raw `ws://` URL. The URL is built internally.
- `--server` lets a company point the client at its own teachat instance.
- Messages only go to people in the sender's room, with the alias in front (`daniel: oi`).
- Joining a room that doesn't exist creates it. A room is deleted when its last member leaves.

## 2026-10-11 09:34 — First commit: WebSocket chat

- Server (`cmd/server`) and client (`cmd/client`) as separate binaries in one module.
- WebSocket with `gorilla/websocket`. The server relays each message to every other connected client.
- The client sends a message on every Enter and shows incoming messages in parallel.
