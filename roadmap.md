# teachat — roadmap

A terminal chat for developers and sysadmins. Minimalist: lightweight, no distractions, used the way you use any Linux tool.

## Principles

- **Flags, not URLs.** Users never type `ws://...?room=x`:
  ```bash
  teachat --room mycompany --password xxx --alias daniel
  ```
  How that reaches the server is an internal detail of the client.
- **Works with pipes.** teachat is a Unix tool like any other.
- **Self-hostable.** A company can run its own instance on an internal server, with no traffic going out to the internet (`--server`).
- **Minimalist.** A feature only goes in if it pays for itself.

## Current state (2026-10-12)

- **Server** (`cmd/server`): rooms in memory, one mutex per room, port 7347 by default (`--addr` to change it), messages up to 1 MB. The `local` room starts with the server and is never deleted.
- **Client** (`cmd/client`): flags `--server`, `--room` and `--alias`. Full screen with tview (`internal/ui`): messages on top, input box at the bottom.
- Pipe mode: `command | teachat` sends each line to the room, echoes it to the terminal and exits when the input ends.
- Encrypted by default: TLS 1.3 with a self-signed certificate and trust on first use, or the administrator's own certificate (`--tls-cert`/`--tls-key`). All in `internal/tlstrust`.
- Shipped features are logged in [features.md](features.md).

## Next steps

In this order. Each one is small on purpose.

1. **Password-protected rooms.** TLS is in place, so a password can travel safely. See the open questions below.
2. **Message history.** Needed for pipe mode to be useful when nobody is in the room, and for persistent rooms. Plan: SQLite with `modernc.org/sqlite` (pure Go, so it stays a single binary).

## Future

- **Discord bridge.** Subscribe to Discord channels (that have the teachat bot) and read messages in the terminal instead of the Discord app, which is heavy and noisy. Read-only at first.
- **Room listing** (`teachat --list`) for public rooms.
- **Structured messages** (`{"alias": ..., "text": ...}` instead of `alias: text`). Removes the fragile parsing in the client and makes room for join/leave notices, history and chosen colors.
- **Config file**, so you don't depend on flags (and for apt/dnf packaging). Already decided: `/etc/teachat/teachat.conf` (machine defaults, installed by the package) and `~/.config/teachat/config` (each user's preferences). Precedence: code defaults → `/etc` → user file → flags. Format `key = value` with `#` comments, no dependency. Colors accept names and hex (tcell already understands both); an invalid value is an error with file and line. Passwords never in `/etc`, only in the user's file with 0600 permissions. Worth it once packaging, passwords or more server options arrive; until then, a shell `alias` does the job.
- **UI:** group consecutive messages from the same person (alias only on the first), highlight mentions (`@daniel`), timestamps on messages.

## Decisions made

- Joining a room that doesn't exist creates it.
- A room is deleted when its last member leaves, except the default room.
- The default room is `local`: it starts with the server, is never deleted, and is where you land if you don't pass `--room` (client) or don't send a room in the URL (server). That way everyone meets without arranging anything. It used to be `geral`, created and deleted like the others.
- Connection state lives in RAM, not in a database: it only exists while the process is alive. A database comes with history.
- `Room` stays in `package main` (`cmd/server/room.go`) until something other than the server needs it.
- The client is full screen (tview), not inline. Inline, the chat would go into the terminal scrollback and get lost among command output.
- On screen: Enter sends, PgUp/PgDn scroll the messages, Ctrl+C quits. Your own message is shown by the client, not echoed back by the server.
- No mouse on screen, so selecting and copying chat text keeps working normally.
- The screen lives in `internal/ui`. The server's `Room` moves out of `package main` later, as a separate change.
- Pipe mode exits when the input ends and never opens the screen. Joining the chat afterwards would hang scripts and cron (nobody to press Ctrl+C), and stdin is already the pipe, not the keyboard.
- Pipe mode echoes each line to the terminal (like `tee`), so whoever runs the command still sees the output. One message per line, no grouping: the simplest path. Large logs flood the room; grouping can come later if it becomes a problem.
- Default port 7347 (previously 23456).
- `-h` on both binaries, with defaults read from the flags themselves so the help never drifts out of date.
- Everything is in English: what the program shows in the terminal (help, logs, errors, startup messages, test output), code comments, the site and documentation (`docs/`), and these project notes. Until 2026-10-12, comments, docs and notes were in Portuguese.
- A command that does nothing (stray argument, unknown flag) shows the help on both binaries, with exit code 2. `help` shows the help with exit code 0.
- 1 MB limit per message on the server (`SetReadLimit`). 64 KB would be enough for typing, but would break large log lines in pipe mode (stack traces, JSON). Over the limit, the server closes only that connection (close 1009).
- `--addr` on the server in Go's `host:port` format: no host listens on all interfaces, a host listens only on that one.
- TLS before passwords, and on its own: TLS is useful without passwords, passwords are useless without TLS.
- TLS on by default (`--no-tls` to turn it off, for testing): optional security tends to stay off.
- Trust model: self-signed certificate + trust on first use, like SSH. The server generates an ECDSA P-256 certificate (10 years) in `~/.config/teachat/` and prints its fingerprint (`SHA256:` + hash of the public key; pinning the key, not the certificate, survives a renewal with the same key). The client saves the fingerprint in `~/.config/teachat/known_servers` on first connection and refuses a different one later. It accepts new servers without asking (like SSH's `accept-new`), so pipes and cron never block. TLS 1.3 minimum, standard library only, all in `internal/tlstrust`.
- An administrator can bring their own certificate (`--tls-cert`/`--tls-key`: Let's Encrypt, a company CA). The client checks every certificate against the machine's trusted authorities first, like a browser, with no pinning, so renewals with a new key keep working; only certificates that don't verify go through trust on first use. The server reloads the files when they change, so certbot renewals need no restart.
- Misconfigurations that would only break at the first renewal are warned about right away: the server warns at startup if the administrator's chain doesn't verify (typically `cert.pem` instead of `fullchain.pem`), and the client warns when a certificate that isn't self-signed falls back to first-use trust (typically connecting by IP instead of the domain name). Warnings, not errors, because a private company CA is legitimate.
- The module is `github.com/daniel-cmdline/teachat`, matching the repository path, so internal imports and `go install` work.

## Open questions

- Password rooms vs. deleting empty rooms: if a room disappears when it empties, anyone can recreate it with a different password. Should password rooms be persistent?
- Can the password go in the connection URL, or should it go in a header or a first message so it doesn't show up in logs?
- Aliases aren't verified: anyone can join as `--alias daniel`. A password protects the room, but doesn't prove who's who inside it. Do we need identity?

## Capacity (load test, 2026-10-11)

Server and clients on the same machine, short messages. Shows the order of magnitude, not precise numbers.

| Connections | Server RAM | 1 message to the whole room |
|---|---|---|
| 1,000 | 28 MB | 5.7 ms |
| 10,000 | 183 MB | 59 ms |
| 50,000 | 875 MB | 309 ms |

About 17 KB per connection, almost all of it goroutine stack and WebSocket buffers. Broadcast time grows linearly with room size. A slow client still delays its own room (measured: a client that never reads stalled a room at 2,082 of 5,000 messages). Fixing that requires a send queue per connection.
