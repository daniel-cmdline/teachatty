# teachatty — roadmap

A terminal chat for developers and sysadmins. Minimalist: lightweight, no distractions, used the way you use any Linux tool.

## Principles

- **Flags, not URLs.** Users never type `ws://...?room=x`:
  ```bash
  teachatty --room mycompany --password xxx --alias daniel
  ```
  How that reaches the server is an internal detail of the client.
- **Works with pipes.** teachatty is a Unix tool like any other.
- **Self-hostable.** A company can run its own instance on an internal server, with no traffic going out to the internet (`--server`).
- **Minimalist.** A feature only goes in if it pays for itself.

## Current state (2026-10-12)

- **Server** (`cmd/teachatty-server`): rooms in memory, one mutex per room, port 7347 by default (`--addr` to change it), messages up to 1 MB. The `local` room starts with the server and is never deleted. Abuse limits: 5 messages/s and 256 KB/s per connection, 10 connections per IP (`--max-per-ip`), stuck members dropped after 5 s, `--no-pipes`. Runs as a hardened systemd service (`contrib/`).
- **Client** (`cmd/teachatty`): flags `--server`, `--room`, `--alias`, `--no-tls` and `--version`. Full screen with tview (`internal/ui`): messages on top, input box at the bottom.
- Pipe mode: `command | teachatty` sends the input to the room (lines arriving within 100 ms go as one message), echoes it to the terminal and exits when the input ends.
- Encrypted by default: TLS 1.3 with a self-signed certificate and trust on first use, or the administrator's own certificate (`--tls-cert`/`--tls-key`). All in `internal/tlstrust`.
- Shipped features are logged in [features.md](features.md).

## Next steps

In this order. Each one is small on purpose.

1. **Public server at teachatty.com.** A small VPS (Oracle Cloud's free tier is the first candidate) running the systemd unit from `contrib/`, with a Let's Encrypt certificate for teachatty.com, so `teachatty --server teachatty.com` works for anyone. The README and the site already point people there.
2. **Release v0.1.0.** A git tag and prebuilt binaries (goreleaser: Linux and macOS, amd64 and arm64), so people don't need Go installed.
3. **Access control: a server-wide shared token or per-room passwords.** Decide after the release, based on what people ask for. TLS is in place, so a secret can travel safely; whatever is stored on the server must be a salted hash (bcrypt or argon2), never the secret itself. See the open questions below.
4. **Message history.** Needed for pipe mode to be useful when nobody is in the room, and for persistent rooms. Plan: SQLite with `modernc.org/sqlite` (pure Go, so it stays a single binary).

## Future

- **Discord bridge.** Subscribe to Discord channels (that have the teachatty bot) and read messages in the terminal instead of the Discord app, which is heavy and noisy. Read-only at first.
- **Room listing** (`teachatty --list`) for public rooms.
- **Structured messages** (`{"alias": ..., "text": ...}` instead of `alias: text`). Removes the fragile parsing in the client and makes room for join/leave notices, history and chosen colors.
- **Config file**, so you don't depend on flags (and for apt/dnf packaging). Already decided: `/etc/teachatty/teachatty.conf` (machine defaults, installed by the package) and `~/.config/teachatty/config` (each user's preferences). Precedence: code defaults → `/etc` → user file → flags. Format `key = value` with `#` comments, no dependency. Colors accept names and hex (tcell already understands both); an invalid value is an error with file and line. Passwords never in `/etc`, only in the user's file with 0600 permissions. Worth it once packaging, passwords or more server options arrive; until then, a shell `alias` does the job.
- **UI:** group consecutive typed messages from the same person (pipe batches already show the alias once), a "↓ new messages" marker while scrolled up, `/me` actions. Join/leave notices and `/who` after structured messages.

## Decisions made

- Joining a room that doesn't exist creates it.
- A room is deleted when its last member leaves, except the default room.
- The default room is `local`: it starts with the server, is never deleted, and is where you land if you don't pass `--room` (client) or don't send a room in the URL (server). That way everyone meets without arranging anything. It used to be `geral`, created and deleted like the others.
- Connection state lives in RAM, not in a database: it only exists while the process is alive. A database comes with history.
- `Room` stays in `package main` (`cmd/teachatty-server/room.go`) until something other than the server needs it.
- The client is full screen (tview), not inline. Inline, the chat would go into the terminal scrollback and get lost among command output.
- On screen: Enter sends, PgUp/PgDn scroll the messages, Ctrl+C quits. Your own message is shown by the client, not echoed back by the server.
- No mouse on screen, so selecting and copying chat text keeps working normally.
- The screen lives in `internal/ui`. The server's `Room` moves out of `package main` later, as a separate change.
- Pipe mode exits when the input ends and never opens the screen. Joining the chat afterwards would hang scripts and cron (nobody to press Ctrl+C), and stdin is already the pipe, not the keyboard.
- Pipe mode echoes each line to the terminal (like `tee`), so whoever runs the command still sees the output. Lines that arrive within 100 ms of each other are sent as one multi-line message (up to 64 KB): a build log is a few messages instead of thousands, and it stays under the server's rate limit.
- Default port 7347 (previously 23456).
- `-h` on both binaries, with defaults read from the flags themselves so the help never drifts out of date.
- Everything is in English: what the program shows in the terminal (help, logs, errors, startup messages, test output), code comments, the site and documentation (`docs/`), and these project notes. Until 2026-10-12, comments, docs and notes were in Portuguese.
- A command that does nothing (stray argument, unknown flag) shows the help on both binaries, with exit code 2. `help` shows the help with exit code 0.
- 1 MB limit per message on the server (`SetReadLimit`). 64 KB would be enough for typing, but would break large log lines in pipe mode (stack traces, JSON). Over the limit, the server closes only that connection (close 1009).
- `--addr` on the server in Go's `host:port` format: no host listens on all interfaces, a host listens only on that one.
- TLS before passwords, and on its own: TLS is useful without passwords, passwords are useless without TLS.
- TLS on by default (`--no-tls` to turn it off, for testing): optional security tends to stay off.
- Trust model: self-signed certificate + trust on first use, like SSH. The server generates an ECDSA P-256 certificate (10 years) in `~/.config/teachatty/` and prints its fingerprint (`SHA256:` + hash of the public key; pinning the key, not the certificate, survives a renewal with the same key). The client saves the fingerprint in `~/.config/teachatty/known_servers` on first connection and refuses a different one later. It accepts new servers without asking (like SSH's `accept-new`), so pipes and cron never block. TLS 1.3 minimum, standard library only, all in `internal/tlstrust`.
- An administrator can bring their own certificate (`--tls-cert`/`--tls-key`: Let's Encrypt, a company CA). The client checks every certificate against the machine's trusted authorities first, like a browser, with no pinning, so renewals with a new key keep working; only certificates that don't verify go through trust on first use. The server reloads the files when they change, so certbot renewals need no restart.
- Misconfigurations that would only break at the first renewal are warned about right away: the server warns at startup if the administrator's chain doesn't verify (typically `cert.pem` instead of `fullchain.pem`), and the client warns when a certificate that isn't self-signed falls back to first-use trust (typically connecting by IP instead of the domain name). Warnings, not errors, because a private company CA is legitimate.
- A stuck member is dropped after a 5-second write deadline instead of a per-connection send queue: 5 lines instead of a goroutine and a queue per member, and good enough until rooms get big.
- The server refuses requests with an `Origin` header: browsers always send one on WebSocket connections, the teachatty client never does.
- The command folders are named after the binaries (`cmd/teachatty`, `cmd/teachatty-server`), so `go install` produces the right names.
- `--version` reads the version Go stamps into the binary (a tag with `go install ...@v0.1.0`, a pseudo-version with the commit when built from a clone), or one set with `-ldflags` for release builds. Shared by both binaries in `internal/version`.
- Abuse limits, needed before running on the open internet: 5 messages/s per connection (bursts of 20) and 256 KB/s; over the limit the server reads more slowly instead of dropping or disconnecting. At most 10 connections per IP (`--max-per-ip`, 0 = no limit for companies behind NAT); over the cap the server answers 429. `golang.org/x/time/rate` for the limiters.
- `--no-pipes` lets a server keep pipe mode out (chat only, no build logs). The client declares pipe mode in the URL (`mode=pipe`); it's a courtesy rule, not a security boundary, so the rate limit stays the real defense against abuse.
- The server is meant to run as a systemd service: `contrib/systemd/teachatty-server.service` (own system user, restart on failure, journal, hardened: no capabilities, read-only system, writes only to `/var/lib/teachatty`) and `contrib/certbot/teachatty-deploy-hook.sh`, which copies Let's Encrypt's root-only key for the `teachatty` user after each renewal, so the server's reload picks it up without a restart.
- The client's look comes from the logo: mint borders, amber title, the terminal's own background and text color (so light and dark terminals both work), a teacup banner on join, gray timestamps (local time of arrival), a placeholder in the empty box. Mentions (`@alias`, whole word, any case) are highlighted in amber and ring the terminal bell.
- The project is called **teachatty** (domain: teachatty.com): binaries `teachatty` and `teachatty-server`, config in `~/.config/teachatty/`, systemd unit `teachatty-server.service` running as user `teachatty`. Renamed from teachat on 2026-10-12. The GitHub repository and the Go module path are still `daniel-cmdline/teachat` until the repository is renamed (the module path must match it for `go install`).
- `--server` without a port means 7347, so `teachatty --server teachatty.com` needs no `:7347`. The address is normalized before anything else, so `teachatty.com` and `teachatty.com:7347` are one server for trust on first use.
- teachatty.com serves both the chat (port 7347) and, later, the site from `docs/` (port 443) from the same machine: one hostname can only point to one place, so the site can't live on GitHub Pages under the same name.
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

About 17 KB per connection, almost all of it goroutine stack and WebSocket buffers. Broadcast time grows linearly with room size. A member that stops reading used to freeze its room for good (measured: stuck at 2,082 of 5,000 messages); now each write has a 5-second deadline and stuck members are dropped, so the room only pauses. A per-connection send queue would remove the pause.
