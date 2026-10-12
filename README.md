<p align="center">
  <img src="assets/teachatty.png" alt="teachatty" width="520">
</p>

<p align="center">
  A minimalist terminal chat for developers and sysadmins.
</p>

---

teachatty is a chat that lives in your terminal. You join a room, type, press Enter, and everyone in the room sees it. There's no heavy desktop app and no noise: it's a small server and a small client, used like any other Unix tool.

It's built to be **self-hosted**. A team can run its own teachatty server on an internal machine, so messages never leave the company network.

## Status

**Early development.** teachatty already works for chatting between machines, but it's still taking shape. Expect breaking changes.

What works today:

- Rooms: joining a room that doesn't exist creates it, and a room disappears when its last member leaves. The default room, `local`, is the exception: it exists as long as the server runs.
- Aliases: your name appears in front of every message you send.
- Command-line flags for server, room and alias.
- A full-screen terminal UI: messages on top, an input box at the bottom that grows as you type. Incoming messages never overwrite what you're typing.
- Each alias gets its own color, the same on every client. Your own messages show your alias in bold.
- teachatty's look: mint borders and an amber title (the logo's colors) on your terminal's own background, a welcome banner with the teacup and the connection details, timestamps, and a hint in the empty input box.
- Mentions: when someone writes `@yourname`, the message stands out in amber and your terminal's bell rings, so a terminal in the background gets your attention.
- Pipe mode: `some-command | teachatty --room ops` sends the output to the room (lines arriving together go as one message), still prints it to your terminal, and exits when the input ends. A server can refuse pipes with `--no-pipes`.
- TLS by default, with trust on first use: no certificates to manage. Or bring your own (Let's Encrypt) with `--tls-cert`/`--tls-key`.
- A member that stops reading (laptop asleep, dropped network) is dropped after 5 seconds instead of freezing the room.
- Abuse limits for the open internet: 5 messages/s per connection (senders are slowed down, nothing is dropped), 10 connections per IP, 1 MB per message.
- A hardened systemd unit and a certbot hook for running the server as a service.
- `--version` on both binaries.
- Server and client as separate binaries, installable with `go install`.

What's coming next:

- A v0.1.0 release with prebuilt binaries, `.deb` and `.rpm` packages
- Access control: a server-wide token or password-protected rooms
- Message history

> 🔒 **Encrypted by default.** On first start the server creates a self-signed certificate and prints its fingerprint. Clients trust a server the first time they connect and refuse a different certificate afterwards, like SSH. The first connection is the one to verify: compare the fingerprint the client shows with the one the server prints. Running on a real domain? Pass your own certificate with `--tls-cert` and `--tls-key`.

## Try it now

There's a public teachatty server at **teachatty.com**. Install the client and jump in:

```bash
go install github.com/daniel-cmdline/teachat/cmd/teachatty@latest
teachatty --server teachatty.com
```

No port needed: the client uses teachatty's port, 7347, when you don't name one. Say hi in the `local` room.

## Install

```bash
go install github.com/daniel-cmdline/teachat/cmd/teachatty@latest          # the client
go install github.com/daniel-cmdline/teachat/cmd/teachatty-server@latest   # the server
```

## Quick start

Requires Go 1.27 or newer.

```bash
git clone https://github.com/daniel-cmdline/teachat.git
cd teachat

# start the server (listens on port 7347)
go run ./cmd/teachatty-server

# in another terminal: no flags needed, you land in the "local" room
go run ./cmd/teachatty

# help (also: `help`, or any unknown argument)
go run ./cmd/teachatty -h
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
make 2>&1 | go run ./cmd/teachatty --room ops     # each line goes to the room
tail -f /var/log/app.log | go run ./cmd/teachatty --room ops   # streams until you stop it
```

You still see every line in your terminal. Lines that arrive within 100 ms of each other go to the room as one message, so a long build log doesn't flood it. Empty lines are shown but not sent. Pipe mode never opens the chat screen, so it's safe in scripts and cron.

To connect from another machine, point the client at the server and make sure port `7347` is open in the server's firewall:

```bash
go run ./cmd/teachatty --server 192.168.0.10:7347
```

## Server flags

| Flag | Default | Description |
|---|---|---|
| `--addr` | `:7347` | Where to listen, `host:port`. No host means all interfaces (`--addr :9000`); a host limits it to that interface (`--addr 10.0.0.5:7347`) |
| `--no-tls` | off | Serve plain `ws://` without encryption (testing only) |
| `--max-per-ip` | `10` | Max simultaneous connections from one IP address; `0` = no limit (e.g. a company behind NAT) |
| `--no-pipes` | off | Refuse clients in pipe mode (`command \| teachatty`), for rooms that want chat and no build logs |
| `--tls-cert`, `--tls-key` | self-signed | Your own certificate and key (PEM), e.g. Let's Encrypt's `fullchain.pem` and `privkey.pem`. Clients that trust the authority verify it like a browser, and the files are reloaded when they change, so renewals need no restart |

Messages larger than 1 MB are refused, and only the connection that sent them is closed. Each connection may send 5 messages a second (bursts of 20) and 256 KB a second; above that, the server slows the sender down instead of dropping anything.

### Running with a Let's Encrypt certificate

```bash
teachatty-server --tls-cert /etc/letsencrypt/live/chat.example.com/fullchain.pem \
               --tls-key  /etc/letsencrypt/live/chat.example.com/privkey.pem
teachatty --server chat.example.com:7347
```

- Use **`fullchain.pem`**, not `cert.pem`: without the intermediate certificate, clients can't verify the chain. The server warns about this at startup.
- Clients should connect **by the domain name**, not the IP: the certificate is only valid for the name. The client warns if it isn't.
- `privkey.pem` is readable only by root. Use the certbot hook from `contrib/` (see "Running as a service" below), which copies it where the server can read it.
- Renewals need no restart: the server reloads the files when they change.

### Running as a service

The repo ships a hardened systemd unit and a certbot hook in [contrib/](contrib/). The server then starts on boot, restarts if it crashes, logs to the journal and runs as its own unprivileged user:

```bash
# 1. a system user for the server, and the binary
sudo useradd --system --home /var/lib/teachatty --shell /usr/sbin/nologin teachatty
go install github.com/daniel-cmdline/teachat/cmd/teachatty-server@latest
sudo install -m 0755 ~/go/bin/teachatty-server /usr/local/bin/

# 2. the service (from contrib/systemd/)
sudo cp contrib/systemd/teachatty-server.service /etc/systemd/system/

# 3a. with a domain: Let's Encrypt + the deploy hook (from contrib/certbot/)
sudo install -D -m 0755 contrib/certbot/teachatty-deploy-hook.sh /usr/local/lib/teachatty/teachatty-deploy-hook.sh
sudo certbot certonly --standalone -d chat.example.com \
  --deploy-hook /usr/local/lib/teachatty/teachatty-deploy-hook.sh

# 3b. without a domain: edit the service and switch to the plain ExecStart line
#     (self-signed certificate, created on first start)

# 4. start it, now and on every boot
sudo systemctl daemon-reload
sudo systemctl enable --now teachatty-server
journalctl -u teachatty-server -f      # logs, including the fingerprint
```

- **The hook** solves the `privkey.pem` permission problem: after every renewal, certbot runs it as root, and it copies the certificate and key into `/var/lib/teachatty/tls/`, owned by the `teachatty` user. The server notices the new files and reloads them by itself, so renewals don't disconnect anyone.
- **The unit is locked down:** the server needs no capabilities (port 7347 is above 1024) and can only write to `/var/lib/teachatty`. Everything else on the system is read-only or hidden from it.
- Behind a firewall, open port 7347 (TCP). For `certbot --standalone`, port 80 must be reachable while the certificate is issued and renewed.

## Client flags

| Flag | Default | Description |
|---|---|---|
| `--server` | `localhost:7347` | Address of the teachatty server |
| `--room` | `local` | Room to join. `local` is created when the server starts and always exists, so everyone who skips `--room` meets there |
| `--alias` | your login (`$USER`) | Name shown in front of your messages. Falls back to `anonymous` when `$USER` is unset |
| `--version` | | Print the version and exit |
| `--no-tls` | off | Connect over plain `ws://`, to a server started with `--no-tls` |

## Project layout

```
cmd/teachatty-server/  the server: rooms and message relay
cmd/teachatty/         the terminal client
contrib/             systemd unit and certbot hook for running the server as a service
internal/ui/  the client's screen (built on tview)
internal/tlstrust/  TLS certificate and trust on first use
docs/         project site and documentation
```

Documentation lives in [docs/](docs/): `index.html` is the project site and `docs.html` is the full documentation (usage, client and server reference, protocol, internals).

Run the tests with `go test ./...`.

The roadmap and the feature log are in [roadmap.md](roadmap.md) and [features.md](features.md).
