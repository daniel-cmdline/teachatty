# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

teachatty is a minimalist, self-hostable terminal chat: a WebSocket relay server (`teachatty-server`) and a full-screen terminal client (`teachatty`). Go module `github.com/daniel-cmdline/teachatty`, licensed AGPL-3.0. Public server and site at teachatty.com.

## Commands

```bash
go build ./cmd/teachatty ./cmd/teachatty-server   # binaries land at the repo root (gitignored)
go test -race ./...                               # the project runs its tests with -race
go test -race -run TestOtherRoomsVanishWhenEmpty ./cmd/teachatty-server
go vet ./...

# local end-to-end: plain ws:// skips the TLS/known_servers setup
go run ./cmd/teachatty-server --no-tls
go run ./cmd/teachatty --no-tls --room dev --alias ana
make 2>&1 | go run ./cmd/teachatty --no-tls --room ops   # pipe mode (stdin not a terminal)
```

Packages (`.deb`/`.rpm` for client and server, tarballs, checksums) are built by goreleaser from `.goreleaser.yaml`; server-package-only files (unit with `EnvironmentFile=/etc/default/teachatty-server`, sysusers, install scripts) live in `packaging/`. `goreleaser check`, then `goreleaser release --snapshot --clean` builds into `dist/` without a tag (version `0.1.0-dev.<commit>`). `packaging/teachatty-server.service` and `contrib/systemd/teachatty-server.service` share the same hardening; keep them in sync.

The release version is injected with `-ldflags "-X github.com/daniel-cmdline/teachatty/internal/version.version=vX.Y.Z"`. Without it, `internal/version` falls back to Go's build info, then to `dev`.

## Architecture

**Wire protocol.** One endpoint, `/ws`. The client puts everything in the query string: `room`, `alias` and `mode=pipe` (pipe mode only). Messages are plain WebSocket text frames. On relay, the server prefixes `alias + ": "`. The client's `ui.Print` splits on the first `": "` to color the alias and detect `@mentions`, so changing that format breaks the client. The server does **not** echo a message back to its sender; the client prints its own messages locally (bold alias). Timestamps and colors are client-side only, and alias colors come from an FNV hash so every client agrees on them.

**Server (`cmd/teachatty-server`, all `package main`).**
- `main.go`: `handleWS` is a named function so the tests can mount it on `httptest`. A global `mu` guards only the `rooms` map. Rooms are created on join and deleted when empty, except `local` (`defaultRoom`), which exists for the server's whole life. Requests with an `Origin` header are refused, which blocks browsers.
- `room.go`: each `Room` has its own mutex, so one room's broadcast doesn't block the others. `broadcast` sets a write deadline (`writeWait`, 5s) and drops and closes members that don't read in time, so one stuck client can't freeze the room while it holds the lock.
- `limits.go`: per-connection `rate.Limiter`s (messages and bytes) *slow the reader down* rather than dropping messages. There is also a per-IP connection cap (`--max-per-ip`) and the `--no-pipes` courtesy check. Refusals happen before the upgrade, as HTTP 403 or 429 with a reason in the body, which the client shows.
- Limits and timeouts are `var`s, not `const`s, on purpose, so the tests can change them.

**Client (`cmd/teachatty`).** `main.go` builds the URL from flags (it normalizes `--server` to `host:7347` when no port is given, which matters for known_servers). It then picks a mode by checking whether stdin is a terminal:
- Full screen, via `internal/ui`, which holds everything tview/tcell. `main` only calls `New/Welcome/Notice/Print/Run/Stop`. Only the screen's Enter callback writes to the conn (gorilla allows one writer). The reader goroutine must not call `log.Fatal`, because that would leave the terminal broken. It signals `main` through a closed channel, and `main` exits after `Run` returns.
- Pipe mode (`pipe.go`): no screen. It echoes stdin like `tee`, batches lines that arrive within 100ms into one message (up to 64KB), and **must keep reading** incoming messages so the server's broadcast doesn't stall on it. At EOF it sends a proper close frame and waits for the reply, so the last lines aren't lost.

**TLS (`internal/tlstrust`).** TLS is on by default. The server either generates and reuses a self-signed cert in `os.UserConfigDir()/teachatty/`, or serves `--tls-cert/--tls-key`, reloading the files when their mtime changes (certbot renewals need no restart). The client verifies CA-signed certs normally; self-signed ones get SSH-style trust on first use, pinned by fingerprint in `~/.config/teachatty/known_servers`. Unexported variants (`clientConfig`, `providedConfig`) take the root pool as a parameter so the tests can inject their own CA.

`contrib/` has a hardened systemd unit and a certbot deploy hook that copies the root-only key into `/var/lib/teachatty/tls/`.

**Production (teachatty.com, Ubuntu 24.04, SSH key-only as root).** Source checkout in `/opt/teachatty` (builds `/usr/local/bin/teachatty-server`). The service uses the contrib unit plus a drop-in `/etc/systemd/system/teachatty-server.service.d/teachatty.com.conf` (Let's Encrypt cert, `--no-pipes --max-per-ip 10`). nginx serves the site from `/var/www/teachatty`, and packages from `/var/www/teachatty/downloads/`, where `*latest*` names are symlinks to the newest build. Certbot's deploy hook is in `/etc/letsencrypt/renewal-hooks/deploy/`. ufw allows 22, 80, 443 and 7347.

## Conventions

- **Everything is in English:** user-facing strings, code comments, docs and notes. Comments are plentiful and explain *why* (see any file). Match that density.
- Neither binary takes positional arguments. A stray argument or unknown flag prints the help and exits 2; `help` exits 0. The `-h` text in each `usage.go` reads defaults from the flags via `flag.VisitAll`. Update the prose sections there when behavior changes.
- Work happens on `dev`; PRs target `dev`, and `main` is the stable branch that `go install ...@latest` resolves to. See [CONTRIBUTING.md](CONTRIBUTING.md).
- **Every commit adds an entry at the top of [features.md](features.md)** (`## YYYY-MM-DD HH:MM — title`, what changed and how it was tested). User-visible changes also update [README.md](README.md), [roadmap.md](roadmap.md) ("Current state", "Next steps", "Decisions made") and the website. **The website is not in the repo**: it lives in `~/docs` (`index.html`, `docs.html`, `style.css`, `assets/`) and is deployed with `rsync -a --delete --exclude downloads/ ~/docs/ root@teachatty.com:/var/www/teachatty/`.
- Commit messages have a short subject line and a body of bullet points. **Never add a co-author or attribution line** (no `Co-Authored-By`, no "Generated with" footers) to commits, PRs or anything else: the owner removed them from history on purpose.
- Before adding a feature, read `roadmap.md`'s principles and "Decisions made". The project is deliberately minimalist (flags instead of URLs, pipe-friendly, a feature only goes in if it pays for itself), and many design questions are already settled there.
