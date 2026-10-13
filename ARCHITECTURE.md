# ARCHITECTURE.md

Technical guide, design decisions, and developer workflow for teachatty.

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

Packages (`.deb`/`.rpm` for client and server, tarballs, checksums) are built by goreleaser from `.goreleaser.yaml`; server-package-only files (unit with `EnvironmentFile=/etc/default/teachatty-server`, sysusers, install scripts) live in `packaging/`. `goreleaser check`, then `goreleaser release --snapshot --clean` builds into `dist/` between releases (version `<next patch>-dev.<commit>`). A release: add it to [CHANGELOG.md](CHANGELOG.md), bump the date/version in `man/*.1`, tag `vX.Y.Z` on `main` (annotated), push the tag, `goreleaser release --clean`, then upload `dist/` to the server's downloads and relink the `*latest*` names. Man pages are hand-written roff in `man/` (check with `groff -man -Tutf8 -ww -z`); goreleaser gzips them into the packages. `packaging/teachatty-server.service` and `contrib/systemd/teachatty-server.service` share the same hardening; keep them in sync.

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

`contrib/` has a hardened systemd unit and a certbot deploy hook that copies the root-only key into the service's own state directory, readable by the unprivileged service user.

## Production Architecture

The public instance at teachatty.com runs on a single Linux host:

- **Relay server.** `teachatty-server` is built from a source checkout and runs as a systemd service based on the hardened unit in `contrib/systemd/`, under its own unprivileged system user. A drop-in override supplies the production flags: a CA-signed certificate instead of the self-signed one, `--no-pipes`, and a per-IP connection cap.
- **TLS.** Certificates come from Let's Encrypt. A certbot deploy hook (`contrib/certbot/`) copies each renewed certificate and key to where the service user can read them; the server notices the changed files and reloads them without a restart.
- **Web and downloads.** Nginx serves the static website and hosts the release packages. Stable `*latest*` names point to the newest build, so download links don't change between releases.
- **Network.** Only SSH, HTTP, HTTPS and the relay port (7347) are exposed.

## Conventions

- **Everything is in English:** user-facing strings, code comments, docs and notes. Comments are plentiful and explain *why* (see any file). Match that density.
- Neither binary takes positional arguments. A stray argument or unknown flag prints the help and exits 2; `help` exits 0. The `-h` text in each `usage.go` reads defaults from the flags via `flag.VisitAll`. Update the prose sections there when behavior changes.
- Work happens on `dev`; PRs target `dev`, and `main` is the stable branch that `go install ...@latest` resolves to. See [CONTRIBUTING.md](CONTRIBUTING.md).
- **Every commit adds an entry at the top of [features.md](features.md)** (`## YYYY-MM-DD HH:MM — title`, what changed and how it was tested). User-visible changes also update [README.md](README.md), [roadmap.md](roadmap.md) ("Current state", "Next steps", "Decisions made") and the website. **The website is not in this repo**: it lives in the private repository `daniel-cmdline/teachatty-site` (`index.html`, `docs.html`, `style.css`, `assets/`) and is deployed separately; the deploy must never upload `.git/`. `assets/demo.gif` here and the site's demo video come from the recording kit in the site repo's `demo/` (`demo/record.sh`).
- Commit messages should use concise imperative subject lines with bulleted detail in the body. Keep commits focused purely on technical changes without auxiliary footers or metadata tags.
- Before adding a feature, read `roadmap.md`'s principles and "Decisions made". The project is deliberately minimalist (flags instead of URLs, pipe-friendly, a feature only goes in if it pays for itself), and many design questions are already settled there.
