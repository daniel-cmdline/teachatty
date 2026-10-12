<p align="center">
  <img src="assets/teachat.png" alt="teachat" width="520">
</p>

<p align="center">
  A minimalist terminal chat for developers and sysadmins.
</p>

---

teachat is a chat that lives in your terminal. You join a room, type, press Enter, and everyone in the room sees it. There's no heavy desktop app and no noise: it's a small server and a small client, used like any other Unix tool.

It's built to be **self-hosted**. A team can run its own teachat server on an internal machine, so messages never leave the company network.

## Status

**Early development.** teachat already works for chatting between machines, but it's still taking shape. Expect breaking changes.

What works today:

- Rooms: joining a room that doesn't exist creates it, and a room disappears when its last member leaves. The default room, `local`, is the exception: it exists as long as the server runs.
- Aliases: your name appears in front of every message you send.
- Command-line flags for server, room and alias.
- A full-screen terminal UI: messages on top, an input box at the bottom that grows as you type. Incoming messages never overwrite what you're typing.
- Each alias gets its own color, the same on every client. Your own messages show your alias in bold.
- Pipe mode: `some-command | teachat --room ops` sends each line to the room, still prints it to your terminal, and exits when the input ends.
- TLS by default, with trust on first use: no certificates to manage.
- A member that stops reading (laptop asleep, dropped network) is dropped after 5 seconds instead of freezing the room.
- `--version` on both binaries.
- Server and client as separate binaries.

What's coming next:

- Password-protected rooms
- Message history

> 🔒 **Encrypted by default.** On first start the server creates a self-signed certificate and prints its fingerprint. Clients trust a server the first time they connect and refuse a different certificate afterwards, like SSH. The first connection is the one to verify: compare the fingerprint the client shows with the one the server prints. Running on a real domain? Pass your own certificate with `--tls-cert` and `--tls-key`.

## Install

```bash
go install github.com/daniel-cmdline/teachat/cmd/teachat@latest          # the client
go install github.com/daniel-cmdline/teachat/cmd/teachat-server@latest   # the server
```

## Quick start

Requires Go 1.27 or newer.

```bash
git clone https://github.com/daniel-cmdline/teachat.git
cd teachat

# start the server (listens on port 7347)
go run ./cmd/teachat-server

# in another terminal: no flags needed, you land in the "local" room
go run ./cmd/teachat

# help (also: `help`, or any unknown argument)
go run ./cmd/teachat -h
```

Run a second client in the same room and start typing.

| Key | Action |
|---|---|
| `Enter` | Send the message |
| `PgUp` / `PgDn` | Scroll through messages |
| `Ctrl+C` | Leave |

Typing anything jumps back to the newest messages.

When stdin isn't a terminal, the client skips the screen and runs in pipe mode instead:

```bash
make 2>&1 | go run ./cmd/teachat --room ops     # each line goes to the room
tail -f /var/log/app.log | go run ./cmd/teachat --room ops   # streams until you stop it
```

You still see every line in your terminal. Lines that arrive within 100 ms of each other go to the room as one message, so a long build log doesn't flood it. Empty lines are shown but not sent. Pipe mode never opens the chat screen, so it's safe in scripts and cron.

To connect from another machine, point the client at the server and make sure port `7347` is open in the server's firewall:

```bash
go run ./cmd/teachat --server 192.168.0.10:7347
```

## Server flags

| Flag | Default | Description |
|---|---|---|
| `--addr` | `:7347` | Where to listen, `host:port`. No host means all interfaces (`--addr :9000`); a host limits it to that interface (`--addr 10.0.0.5:7347`) |
| `--no-tls` | off | Serve plain `ws://` without encryption (testing only) |
| `--max-per-ip` | `10` | Max simultaneous connections from one IP address; `0` = no limit (e.g. a company behind NAT) |
| `--no-pipes` | off | Refuse clients in pipe mode (`command \| teachat`), for rooms that want chat and no build logs |
| `--tls-cert`, `--tls-key` | self-signed | Your own certificate and key (PEM), e.g. Let's Encrypt's `fullchain.pem` and `privkey.pem`. Clients that trust the authority verify it like a browser, and the files are reloaded when they change, so renewals need no restart |

Messages larger than 1 MB are refused, and only the connection that sent them is closed. Each connection may send 5 messages a second (bursts of 20) and 256 KB a second; above that, the server slows the sender down instead of dropping anything.

### Running with a Let's Encrypt certificate

```bash
teachat-server --tls-cert /etc/letsencrypt/live/chat.example.com/fullchain.pem \
               --tls-key  /etc/letsencrypt/live/chat.example.com/privkey.pem
teachat --server chat.example.com:7347
```

- Use **`fullchain.pem`**, not `cert.pem`: without the intermediate certificate, clients can't verify the chain. The server warns about this at startup.
- Clients should connect **by the domain name**, not the IP: the certificate is only valid for the name. The client warns if it isn't.
- `privkey.pem` is readable only by root. If teachat-server runs as a regular user, give it read access to the key (for example with a certbot deploy hook that copies the files somewhere it can read).
- Renewals need no restart: the server reloads the files when they change.

## Client flags

| Flag | Default | Description |
|---|---|---|
| `--server` | `localhost:7347` | Address of the teachat server |
| `--room` | `local` | Room to join. `local` is created when the server starts and always exists, so everyone who skips `--room` meets there |
| `--alias` | your login (`$USER`) | Name shown in front of your messages. Falls back to `anonymous` when `$USER` is unset |
| `--version` | | Print the version and exit |
| `--no-tls` | off | Connect over plain `ws://`, to a server started with `--no-tls` |

## Project layout

```
cmd/teachat-server/  the server: rooms and message relay
cmd/teachat/         the terminal client
internal/ui/  the client's screen (built on tview)
internal/tlstrust/  TLS certificate and trust on first use
docs/         project site and documentation
```

Documentation lives in [docs/](docs/): `index.html` is the project site and `docs.html` is the full documentation (usage, client and server reference, protocol, internals).

Run the tests with `go test ./...`.

The roadmap and the feature log are in [roadmap.md](roadmap.md) and [features.md](features.md).
