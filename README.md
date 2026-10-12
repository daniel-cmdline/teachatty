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

- Rooms: joining a room that doesn't exist creates it, and a room disappears when its last member leaves.
- Aliases: your name appears in front of every message you send.
- Command-line flags for server, room and alias.
- A full-screen terminal UI: messages on top, an input box at the bottom that grows as you type. Incoming messages never overwrite what you're typing.
- Server and client as separate binaries.

What's coming next:

- Configurable server port
- Pipe mode: `some-command | teachat --room ops`
- Password-protected rooms and TLS (`wss://`)
- Message history

> ⚠️ **No encryption yet.** Traffic is plain `ws://`, so only use teachat on networks you trust until TLS lands.

## Quick start

Requires Go 1.27 or newer.

```bash
git clone https://github.com/daniel-cmdline/teachat.git
cd teachat

# start the server (listens on port 23456)
go run ./cmd/server

# in another terminal, or on another machine
go run ./cmd/client --room dev --alias daniel
```

Run a second client in the same room and start typing.

| Key | Action |
|---|---|
| `Enter` | Send the message |
| `PgUp` / `PgDn` | Scroll through messages |
| `Ctrl+C` | Leave |

Typing anything jumps back to the newest messages.

The client needs a real terminal for now, so piping into it (`echo hi | ...`) fails until pipe mode lands.

To connect from another machine, point the client at the server and make sure port `23456` is open in the server's firewall:

```bash
go run ./cmd/client --server 192.168.0.10:23456 --room dev --alias mateus
```

## Client flags

| Flag | Default | Description |
|---|---|---|
| `--server` | `localhost:23456` | Address of the teachat server |
| `--room` | `geral` | Room to join |
| `--alias` | `anonimo` | Name shown in front of your messages |

## Project layout

```
cmd/server/   the server: rooms and message relay
cmd/client/   the terminal client
internal/ui/  the client's screen (built on tview)
docs/         project site and documentation
```

Documentation lives in [docs/](docs/): `index.html` is the project site and `docs.html` is the full documentation (usage, client and server reference, protocol, internals). Both are in Portuguese.

The roadmap and the feature log are in [roadmap.md](roadmap.md) and [features.md](features.md). Both are written in Portuguese for now.
