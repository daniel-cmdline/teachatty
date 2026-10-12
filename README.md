<p align="center">
  <img src="assets/teachatty.png" alt="teachatty" width="520">
</p>

<p align="center">
  A minimalist terminal chat for developers and sysadmins.
</p>

---

teachatty is a chat that lives in your terminal. You join a room, type, press Enter, and everyone in the room sees it. There's no heavy desktop app and no noise: it's a small server and a small client, used like any other Unix tool.

It's built to be **self-hosted**. A team can run its own teachatty server on an internal machine, so messages never leave the company network.

teachatty is **free and open source software under the [GNU AGPL-3.0](LICENSE)**, and it will stay that way.

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
- Server and client as separate binaries, installable with `go install`, or as `.deb` and `.rpm` packages from [teachatty.com/downloads](https://teachatty.com/downloads/).
- A public server at teachatty.com.

What's coming next:

- Signed apt and dnf repositories, AUR and Homebrew, so updates arrive with your system's upgrades
- Access control: a server-wide token or password-protected rooms
- Message history

> 🔒 **Encrypted by default.** On first start the server creates a self-signed certificate and prints its fingerprint. Clients trust a server the first time they connect and refuse a different certificate afterwards, like SSH. The first connection is the one to verify: compare the fingerprint the client shows with the one the server prints. Running on a real domain? Pass your own certificate with `--tls-cert` and `--tls-key`.

## Try it now

There's a public teachatty server at **teachatty.com**, open to everyone. [Install](#install) the client and jump in:

```bash
teachatty --server teachatty.com                            # the lobby: the local room
teachatty --server teachatty.com --room tea-with-friends    # your own room
```

No port needed: the client uses teachatty's port, 7347, when you don't name one. Without `--room` you land in `local`, the lobby everyone shares. To meet friends, make up a room name and send them the same command: the room appears when the first person joins and vanishes when the last one leaves. Rooms have no passwords yet, so anyone who knows the name can join; pick one that's hard to guess. Pipe mode is turned off on the public server (`--no-pipes`).

## Install

teachatty comes in two parts. **The client** (`teachatty`) is what you chat with, and it's all you need to join teachatty.com. **The server** (`teachatty-server`) is optional: install it only if you want to host rooms of your own.

### 1 · Download the client &nbsp;—&nbsp; everyone needs this

| Your system | amd64 · x86_64 · Intel | arm64 · aarch64 · Apple Silicon |
|---|---|---|
| **Debian · Ubuntu · Mint** | [⬇ `teachatty_latest_amd64.deb`](https://teachatty.com/downloads/teachatty_latest_amd64.deb) | [⬇ `teachatty_latest_arm64.deb`](https://teachatty.com/downloads/teachatty_latest_arm64.deb) |
| **Fedora · RHEL · Rocky · Alma** | [⬇ `teachatty-latest.x86_64.rpm`](https://teachatty.com/downloads/teachatty-latest.x86_64.rpm) | [⬇ `teachatty-latest.aarch64.rpm`](https://teachatty.com/downloads/teachatty-latest.aarch64.rpm) |
| **macOS** | [⬇ `teachatty_latest_darwin_amd64.tar.gz`](https://teachatty.com/downloads/teachatty_latest_darwin_amd64.tar.gz) | [⬇ `teachatty_latest_darwin_arm64.tar.gz`](https://teachatty.com/downloads/teachatty_latest_darwin_arm64.tar.gz) |
| **Any other Linux** | [⬇ `teachatty_latest_linux_amd64.tar.gz`](https://teachatty.com/downloads/teachatty_latest_linux_amd64.tar.gz) | [⬇ `teachatty_latest_linux_arm64.tar.gz`](https://teachatty.com/downloads/teachatty_latest_linux_arm64.tar.gz) |

Or install it straight from the terminal:

**Debian · Ubuntu · Mint**
```bash
curl -LO https://teachatty.com/downloads/teachatty_latest_amd64.deb
sudo apt install ./teachatty_latest_amd64.deb
```

**Fedora · RHEL · Rocky · Alma**
```bash
sudo dnf install https://teachatty.com/downloads/teachatty-latest.x86_64.rpm
```

**Anywhere with Go 1.26+**
```bash
go install github.com/daniel-cmdline/teachatty/cmd/teachatty@latest
```

> [!TIP]
> **That's it, nothing else to set up.** The packages put `teachatty` in `/usr/bin`, which is already in your `PATH`. Open any terminal, in any folder, and run:
>
> ```bash
> teachatty --server teachatty.com
> ```
>
> With `go install`, it lands in `~/go/bin`, which Go's standard setup adds to your `PATH`. If `teachatty` isn't found, add `export PATH=$PATH:$HOME/go/bin` to your `~/.bashrc`. From a tarball, copy `teachatty` into a folder in your `PATH`, such as `/usr/local/bin`. On macOS, a binary downloaded with a browser isn't signed yet, so clear the quarantine flag once: `xattr -d com.apple.quarantine /usr/local/bin/teachatty`.

### 2 · Download the server &nbsp;—&nbsp; optional, to host your own rooms

For your team, your homelab or your own network. You don't need it to chat on teachatty.com.

| Your system | amd64 · x86_64 · Intel | arm64 · aarch64 · Apple Silicon |
|---|---|---|
| **Debian · Ubuntu · Mint** | [⬇ `teachatty-server_latest_amd64.deb`](https://teachatty.com/downloads/teachatty-server_latest_amd64.deb) | [⬇ `teachatty-server_latest_arm64.deb`](https://teachatty.com/downloads/teachatty-server_latest_arm64.deb) |
| **Fedora · RHEL · Rocky · Alma** | [⬇ `teachatty-server-latest.x86_64.rpm`](https://teachatty.com/downloads/teachatty-server-latest.x86_64.rpm) | [⬇ `teachatty-server-latest.aarch64.rpm`](https://teachatty.com/downloads/teachatty-server-latest.aarch64.rpm) |
| **macOS** | [⬇ `teachatty-server_latest_darwin_amd64.tar.gz`](https://teachatty.com/downloads/teachatty-server_latest_darwin_amd64.tar.gz) | [⬇ `teachatty-server_latest_darwin_arm64.tar.gz`](https://teachatty.com/downloads/teachatty-server_latest_darwin_arm64.tar.gz) |
| **Any other Linux** | [⬇ `teachatty-server_latest_linux_amd64.tar.gz`](https://teachatty.com/downloads/teachatty-server_latest_linux_amd64.tar.gz) | [⬇ `teachatty-server_latest_linux_arm64.tar.gz`](https://teachatty.com/downloads/teachatty-server_latest_linux_arm64.tar.gz) |

**Debian · Ubuntu · Mint**
```bash
curl -LO https://teachatty.com/downloads/teachatty-server_latest_amd64.deb
sudo apt install ./teachatty-server_latest_amd64.deb
sudo systemctl enable --now teachatty-server
```

**Fedora · RHEL · Rocky · Alma**
```bash
sudo dnf install https://teachatty.com/downloads/teachatty-server-latest.x86_64.rpm
sudo systemctl enable --now teachatty-server
```

**Anywhere with Go 1.26+** (runs in the foreground; see [Running as a service](#running-as-a-service))
```bash
go install github.com/daniel-cmdline/teachatty/cmd/teachatty-server@latest
teachatty-server
```

> [!NOTE]
> **The packages set everything up:** a locked-down systemd service that starts on boot, its own `teachatty` system user, TLS on port 7347 with a certificate created on first start, the certbot hook, and a config file for flags, `/etc/default/teachatty-server`. Open port 7347/TCP in the firewall, and everyone on your network connects with:
>
> ```bash
> teachatty --server 192.168.0.10
> ```

Every file, with `checksums.txt`, is at **[teachatty.com/downloads](https://teachatty.com/downloads/)**. The `latest` names always point to the newest release, currently **v0.1.0** (see the [changelog](CHANGELOG.md)). Each package includes a man page: `man teachatty`, `man teachatty-server`.

## Your own network

Run the server on any machine in your office, homelab or VPN, and everyone points the client at it. Messages never leave your network, and nothing needs internet access:

```bash
teachatty-server                          # on one machine (port 7347, TLS on)
teachatty --server 192.168.0.10           # on everyone else's
```

## Quick start

Requires Go 1.26 or newer.

```bash
git clone https://github.com/daniel-cmdline/teachatty.git
cd teachatty

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

With the `teachatty-server` package (see [Install](#install)), this is already set up: edit `/etc/default/teachatty-server`, and for Let's Encrypt use `--deploy-hook /usr/share/teachatty-server/teachatty-deploy-hook.sh`. For a manual install, the repo ships a hardened systemd unit and a certbot hook in [contrib/](contrib/). The server then starts on boot, restarts if it crashes, logs to the journal and runs as its own unprivileged user:

```bash
# 1. a system user for the server, and the binary
sudo useradd --system --home /var/lib/teachatty --shell /usr/sbin/nologin teachatty
go install github.com/daniel-cmdline/teachatty/cmd/teachatty-server@latest
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
packaging/           the server package's unit, config and install scripts (.goreleaser.yaml builds the packages)
internal/ui/  the client's screen (built on tview)
internal/tlstrust/  TLS certificate and trust on first use
```

The full documentation (usage, client and server reference, protocol, internals) is at **[teachatty.com/docs.html](https://teachatty.com/docs.html)**. The website is not part of this repository.

Run the tests with `go test -race ./...`. Build the packages locally with `goreleaser release --snapshot --clean` (output in `dist/`).

The roadmap and the feature log are in [roadmap.md](roadmap.md) and [features.md](features.md).

## Contributing

Bug reports, ideas and pull requests are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) first.

## License

teachatty is free software: you can redistribute it and/or modify it under the terms of the [GNU Affero General Public License v3.0](LICENSE). If you run a modified teachatty server for others, the AGPL asks you to offer them its source too.
