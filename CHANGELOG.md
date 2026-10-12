# Changelog

All notable changes to teachatty, by release. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/). Before 1.0.0, a minor release (0.x.0) may break compatibility.

The detailed, commit-by-commit log is in [features.md](features.md).

## [Unreleased]

## [0.1.0] - 2026-10-12

The first release.

### Client (`teachatty`)

- Full-screen terminal chat: messages on top, an input box at the bottom that grows up to 5 lines. Incoming messages never overwrite what you type.
- `--server`, `--room`, `--alias`, `--no-tls`, `--version`. `--server` without a port uses 7347.
- Each alias has its own color, the same on every client; your own messages show your alias in bold. Timestamps on every message.
- Mentions: `@name` shows in amber and rings the mentioned person's terminal bell.
- `PgUp`/`PgDn` to scroll back; typing jumps to the newest messages. `Ctrl+C` to leave.
- Pipe mode: `command | teachatty` sends the input to the room, echoes it like `tee`, batches lines that arrive within 100 ms into one message, and exits when the input ends.

### Server (`teachatty-server`)

- Rooms in memory: created on join, removed when empty. The default room, `local`, always exists.
- TLS by default, with a self-signed certificate created on first start and SSH-style trust on first use in the client (`~/.config/teachatty/known_servers`). Or your own certificate with `--tls-cert`/`--tls-key`, reloaded when the files change, so Let's Encrypt renewals need no restart.
- Abuse limits: 1 MB per message, 5 messages/s (bursts of 20) and 256 KB/s per connection (senders are slowed down, nothing is dropped), 10 connections per IP (`--max-per-ip`), members that stop reading dropped after 5 seconds, browsers refused.
- `--addr` to choose the interface and port, `--no-pipes` to refuse pipe mode.

### Packaging

- `.deb` and `.rpm` packages for the client and the server (amd64, arm64), and Linux and macOS tarballs, with checksums, at https://teachatty.com/downloads/.
- The server package installs a hardened systemd unit, the `teachatty` system user, `/etc/default/teachatty-server` for flags, and a certbot deploy hook. It doesn't start the service until you enable it.
- Man pages: `teachatty(1)` and `teachatty-server(1)`.
- Builds with Go 1.26 or newer.

### Project

- Licensed under the GNU AGPL-3.0.
- A public server at teachatty.com.

[Unreleased]: https://github.com/daniel-cmdline/teachatty/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/daniel-cmdline/teachatty/releases/tag/v0.1.0
