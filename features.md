# Features

Log of shipped features, newest first. Add an entry with every commit.

## 2026-10-12 18:52 — The website gets its own private repository

- The site and documentation now live in the private repository `daniel-cmdline/teachatty-site`, so they keep a history outside this one. CLAUDE.md has the deploy command, which never uploads `.git/`.
- teachatty.com's nginx now refuses any dotfile path (`/.git`, `/.env`, ...), in case one is ever uploaded by mistake. **Tested:** `/.git/config` answers 404, the site still loads, and `certbot renew --dry-run` still succeeds.

## 2026-10-12 18:40 — The website leaves the repository

- `docs/` (the site's `index.html`, the documentation's `docs.html` and `style.css`) is no longer in the repository. It's large hand-written HTML that nobody needs to read or build the code; it now lives outside the repo, with its own copy of the logo, and is deployed to teachatty.com from there.
- The README, CONTRIBUTING.md, CLAUDE.md and the roadmap point to https://teachatty.com/docs.html instead of `docs/`. Documentation fixes go through issues.
- `assets/` stays: the README shows the logo from it.
- **Tested:** the site redeployed from its new place and served over HTTPS, logo included; `go build ./...` and `go test -race ./...` pass.

## 2026-10-12 18:22 — Open source (AGPL-3.0), packages, teachatty.com live

- **License: GNU AGPL-3.0** (`LICENSE`). teachatty is open source and stays that way: whoever distributes a modified version, or runs one as a service for others, must share its source.
- **`.deb` and `.rpm` packages** for the client (`teachatty`) and the server (`teachatty-server`), amd64 and arm64, plus Linux and macOS tarballs and `checksums.txt`, all built by goreleaser (`.goreleaser.yaml`). The server package installs a hardened systemd unit that reads its flags from `/etc/default/teachatty-server` (kept on upgrades), creates the `teachatty` system user (sysusers.d), ships the certbot hook in `/usr/share/teachatty-server/`, and doesn't start the service until the admin enables it; upgrades restart a running server, removal stops it and keeps `/var/lib/teachatty`. Files in `packaging/`. Development builds (`0.1.0-dev.<commit>`) until v0.1.0 is tagged.
- **teachatty.com is live**: nginx serves the site and docs over HTTPS (Let's Encrypt, HTTP redirects to HTTPS) and the packages at https://teachatty.com/downloads/, where `*latest*` names always point to the newest build. teachatty-server runs as the hardened systemd service with the same certificate, `--no-pipes` and `--max-per-ip 10`. Firewall open on 22, 80, 443 and 7347 only; SSH accepts keys only.
- **The site's pitch:** "The tea is for everyone. Pull up a chair." A new section on the home page: join the public lobby, make a room for your friends with `--room`, keep it on your own network, and open source forever. A new install section (`.deb`, `.rpm`, `go install`, the server as a service) replaces "Start in a minute". The docs, README and roadmap follow, with the public server's `--no-pipes` noted.
- **CONTRIBUTING.md**: how to report bugs and security issues, the development setup, branches (PRs target `dev`; `main` is stable), code and commit conventions, and the pull request checklist.
- **Tested:** `goreleaser check`; packages installed and removed in clean Debian stable and Fedora containers (versions, `teachatty` user, unit, `/etc/default` kept on removal, AGPL in the rpm metadata); on teachatty.com, two clients exchanging messages over TLS verified against the system's CAs, pipe mode refused with the server's reason, the 11th connection from one IP refused, HTTPS and the downloads over IPv4 and IPv6, and `certbot renew --dry-run`.

## 2026-10-12 16:07 — Repository and module renamed to teachatty

- The GitHub repository is now `daniel-cmdline/teachatty`, and the Go module path follows it: `github.com/daniel-cmdline/teachatty`, with every internal import, link and install command updated (`go install github.com/daniel-cmdline/teachatty/cmd/teachatty@latest`). GitHub redirects the old address.
- Commit messages no longer carry a co-author line, and it was removed from the earlier commits too (only the messages were rewritten; code, dates and order are unchanged).

## 2026-10-12 16:02 — teachat is now teachatty

- **New name, new domain: teachatty.com.** Everything renamed: the binaries (`teachatty`, `teachatty-server`; folders `cmd/teachatty`, `cmd/teachatty-server`), every message and help screen, the config directory (`~/.config/teachatty/`), the self-signed certificate's name, the systemd unit (`teachatty-server.service`, user `teachatty`, `/var/lib/teachatty`), the certbot hook (`teachatty-deploy-hook.sh`, `TEACHATTY_USER`/`TEACHATTY_TLS_DIR`), `.gitignore`, the README, the roadmap, the site, the docs, and the older entries of this log.
- **Not renamed yet:** the GitHub repository and the Go module path (`github.com/daniel-cmdline/teachat`). `go install` requires the module path to match the repository, so both change together once the repository is renamed on GitHub. Until then: `go install github.com/daniel-cmdline/teachat/cmd/teachatty@latest`.
- **New logo** with the new name, redrawn as SVG in the original's style (the neon mint cup with its `>_` prompt, the amber speech bubble and cursor, the dark grid) and rendered to PNG at the same size: `assets/teachatty.svg` (source, editable) and `assets/teachatty.png`.
- **`--server` without a port uses 7347**, so the pitch is just `teachatty --server teachatty.com`. The address is normalized first, so `teachatty.com` and `teachatty.com:7347` are the same server for trust on first use.
- **The public server is advertised**: "Try it now" in the README, the site's hero and "Start in a minute", the docs' "Getting started", and the client's `-h`.
- Upgrading from a teachat build: the config moved from `~/.config/teachat/` to `~/.config/teachatty/`, so a server generates a new self-signed certificate (clients trust it as a new server) unless the files are moved over.
- **Tested:** `--server localhost` connects to `localhost:7347`, and a second connection as `localhost:7347` is recognized as the same, already trusted server; the config lands in `~/.config/teachatty/`. All 19 tests pass with `-race`.

## 2026-10-12 15:39 — teachatty's look: colors, welcome banner, timestamps, mentions

- **The logo's colors:** mint borders (`#72e3a6`), an amber title (`#f2b04f`), on the terminal's own background and text color instead of forced black and white, so light terminals look right too.
- **Welcome banner:** on joining, the teacup in text next to the server and room, how the connection is secured (TLS version and the start of the server's fingerprint, read from the live connection; or "not encrypted" with `--no-tls`), the keys, and a hint about mentions. The first-use trust notice follows it.
- **Placeholder:** `type a message…` in gray while the input box is empty.
- **Timestamps:** each message starts with the local time it arrived (`15:38`, gray). The server sends no time; the client adds it.
- **Mentions:** `@yourname` (whole word, any case; `email@yourname.com` doesn't count) in someone else's message shows in amber and bold and rings the terminal bell, so a terminal in another tab or window gets your attention. To ring the bell, `Run` now creates the tcell screen itself and hands it to tview (`SetScreen`), keeping a reference, since tview doesn't expose its screen. The beep runs inside the event loop.
- **Tested end to end** with two clients in tmux: the banner, the colors (mint and amber escape codes on screen), the placeholder, the timestamps, a mention highlighted in amber with the bell flag set on the mentioned user's window only, a non-mention left alone. No data races.
- The client's `-h` explains mentions too.
- **Tested live** with a person in the room: the mention beeped their terminal, and piping `/etc/passwd` arrived as a single batched message with the alias on the first line only.
- **Docs:** README, roadmap (the UI ideas still open), the site (the screen mockup now shows the colors, timestamps and a mention; new "Mentions ring your bell" point), and the docs ("joining" and "mentions" in the client reference, a "teachatty's look" section in client internals). Also brings README, roadmap and both pages up to date (release next, current state, ~1,250 lines).

## 2026-10-12 15:31 — Running as a systemd service

- **`contrib/systemd/teachatty-server.service`:** starts on boot, restarts on failure, logs to the journal, runs as its own `teachatty` system user. Keeps its files in `/var/lib/teachatty` (`StateDirectory`, with `XDG_CONFIG_HOME=/var/lib`, so the self-signed certificate lands there). Hardened: no capabilities (port 7347 is above 1024), `ProtectSystem=strict`, `ProtectHome`, private `/tmp`, `NoNewPrivileges`, only network sockets, and more. Two `ExecStart` lines: with a Let's Encrypt certificate, or self-signed.
- **`contrib/certbot/teachatty-deploy-hook.sh`:** Let's Encrypt's `privkey.pem` is readable only by root. certbot runs the hook as root after every issuance and renewal; it copies the key and then the certificate into `/var/lib/teachatty/tls/`, owned by `teachatty`. The server's file-watching reload picks them up, so renewals disconnect nobody. It refuses to run outside certbot (no `RENEWED_LINEAGE`).
- **Tested:** `systemd-analyze verify` on the unit. The server run as a transient systemd service with the same restart policy and hardening (`systemd-run --user`): it starts, logs to the journal, accepts a client, and after a `kill -9` systemd restarts it with the same certificate (same fingerprint, the client still trusts it). The hook, run through a full certbot-style renewal against a live server: the same process started serving the new certificate without a restart, and the client connected. Two notes from testing: a user-mode systemd needs `PrivateUsers=yes` for the namespace-based hardening (the real system service doesn't), and `PrivateTmp` hides `/tmp` from the service, which is why the binary goes in `/usr/local/bin` and the state in `/var/lib`.
- **Docs:** README and the HTML ("Run the server as a service", the step-by-step setup, linked from the sidebar), roadmap.

## 2026-10-12 15:20 — Ready for the open internet: rate limits, per-IP cap, batched pipes, --no-pipes

- **Rate limit per connection:** 5 messages a second with bursts of 20, and 256 KB a second with bursts up to one 1 MB message. Over the limit, the server waits before reading that connection again: nothing is dropped and nobody is disconnected, TCP slows the sender down. A script flooding a room is simply slowed to the limit. Code in `cmd/teachatty-server/limits.go`, with `golang.org/x/time/rate` (new dependency, Go team, pure Go).
- **Per-IP connection cap:** at most 10 simultaneous connections per IP (each costs ~17 KB), configurable with `--max-per-ip` (`0` = no cap, for a company behind NAT). Over the cap the server answers 429 before the WebSocket handshake, and the client says `the server refused the connection: too many connections from your address`.
- **Pipe mode batches lines:** lines that arrive within 100 ms of the first one go out as one multi-line message (up to 64 KB). `seq 1 5000 | teachatty` is now 1 message instead of 5,000, so build logs neither flood the room nor hit the rate limit. Slow output (a line every 300 ms) still goes out line by line. In the full screen, a batch shows the alias on its first line only. The "large logs flood the room" limitation is gone.
- **`--no-pipes` on the server:** refuses clients in pipe mode, for rooms that want chat and no build logs. The client now declares pipe mode in the connection URL (`mode=pipe`); a `--no-pipes` server answers 403 with the reason. A courtesy rule, not a security boundary: a client that doesn't declare the mode isn't stopped, and the rate limit is what holds back abuse.
- **Refused connections show the server's own reason.** For the deliberate refusals (403, 429), the client prints the body the server sent, e.g. `the server refused the connection: this server doesn't accept pipe mode (it was started with --no-pipes)`, instead of a generic `bad handshake`.
- **Tests:** 3 new server tests: 30 messages sent at once all arrive, but take ~2 s (burst of 20, then 5/s); the third connection from one IP gets 429 with the cap at 2, and connecting works again once one leaves; with `--no-pipes`, a pipe-mode connection gets 403 with the reason while an interactive one gets in. The stuck-member test lifts the rate limit, since it pushes 20 MB on purpose. 19 tests in total, with `-race`.
- **Tested end to end:** `seq 1 5000` arrives as 1 message with all 5,000 lines; slow output stays one message per line; the 2 MB line is still refused; the 4th connection from one IP with `--max-per-ip 3` is refused with the clear message, and works again after the others leave; a batch looks right in the full screen.
- **Docs:** both help screens, README, roadmap, and the HTML (server flags and rate, `--no-pipes`, 403/429 exit codes, batched pipe mode, a reverse-proxy caveat for the per-IP cap, code excerpts regenerated).

## 2026-10-12 15:11 — Stuck members no longer freeze rooms, origin check, binary names, --version

- **Bug fixed: one stuck member froze its whole room for good.** Broadcast writes to members one at a time while holding the room's lock; a member that stops reading (a laptop asleep, a network that dropped without closing, a teachatty pipe suspended with Ctrl+Z) filled its buffers and blocked broadcast forever. Each write now has a 5-second deadline (`writeWait`): a member that misses it is removed and its connection closed, and the room carries on after one pause.
- Correction to something said while planning: the full-screen client can't be suspended with Ctrl+Z, because raw mode delivers Ctrl+Z as an ordinary key. The real triggers are the ones above.
- **Pipe mode exits as soon as its connection drops**, even while waiting for input. Before, a quiet `tail -f | teachatty` whose connection dropped (for example, dropped by the server after being suspended) hung until the next line arrived.
- **Origin check:** the server refuses requests with an `Origin` header (403). Browsers always send one on WebSocket connections and the teachatty client never does, so web pages can't connect behind a user's back.
- **Binary names:** `cmd/client` → `cmd/teachatty`, `cmd/server` → `cmd/teachatty-server`, so `go install github.com/daniel-cmdline/teachat/cmd/teachatty@latest` installs `teachatty`. The root `.gitignore` now ignores `/teachatty` and `/teachatty-server`.
- **`--version`** on both binaries (`internal/version`): a release build's `-ldflags` value, else the version Go stamps (a tag, or `v0.0.0-<date>-<commit>+dirty` from a clone), else `dev` (go run).
- **Tests:** 2 new server tests: a member that never reads is dropped and the others receive all 40 messages of 512 KB (checked that it fails cleanly, with "the room froze", when the deadline is removed); a request with an `Origin` header gets 403. 16 tests in total, with `-race`.
- **Tested end to end:** a suspended pipe client in a room flooded with 40 lines of 500 KB: the flood finished in ~6 s (one 5 s pause), a full-screen watcher received the last line, and the suspended client, once resumed, exited with `connection closed`. A quiet pipe exits right away when the server stops.
- **Docs:** README (install with `go install`, new paths, `--version`), roadmap (release is next; access control decided after it, with the hash + salt requirement noted), and the HTML: new paths, install section, origin, the stuck-member limitation, `--version`, code excerpts regenerated.

## 2026-10-12 14:38 — Bring your own certificate

- **Server:** `--tls-cert` and `--tls-key` serve the administrator's certificate instead of the self-signed one, for example Let's Encrypt's `fullchain.pem` and `privkey.pem`. The startup line describes it: `TLS · certificate for chat.example.com, issued by R11, valid until 2026-12-30`. Both flags go together, and neither combines with `--no-tls`.
- **Renewals without restarts:** the files are checked on each new connection and reloaded when they change. If a reload fails halfway (certificate written, key not yet), the previous certificate keeps being served.
- **Client, no new flag:** every certificate is first checked against the machine's trusted authorities, like a browser does (chain + hostname). If it passes, it's accepted with no pinning, so renewals with a new key just work. Only certificates that don't verify (self-signed) go through trust on first use.
- **Why not just pin everything:** Let's Encrypt usually renews with a new key every ~60 days. Pinning would make every client refuse the legitimate server after each renewal.
- **Tests:** 5 new in `internal/tlstrust` (an authority-signed certificate isn't pinned and survives a renewal with a new key; one for another hostname falls back to first-use trust, with the warning; the files are reloaded when they change; cert and key must go together; the server warns when the chain doesn't verify). 14 tests in total, with `-race`. A test-only pitfall was found and documented: `httptest.StartTLS` adds its own example certificate when the config uses `GetCertificate`, so the tests wrap the listener with `tls.NewListener`, like the real server.
- **Tested end to end** with a fake authority made with `openssl`: the client trusting it connects with nothing pinned; renewing the certificate with a new key while the server runs is picked up without a restart and the client keeps connecting; a client that doesn't know the authority falls back to first-use trust; bad flag combinations and missing files fail clearly.
- **Warnings for the two silent mistakes.** Both used to work at first and break at the first renewal, ~60 days later:
  - **Server, at startup:** if the administrator's chain doesn't verify against the machine's trusted authorities, it prints a warning, with `use fullchain.pem, not cert.pem` when the file has no intermediate certificate. A warning, not an error: a private company CA installed on every client is legitimate and fails this check on the server.
  - **Client, on first use:** when a certificate that isn't self-signed falls back to first-use trust (connecting by IP instead of the domain name, or a missing intermediate), the notice comes with a warning that it will be refused after the next renewal.
- **Bug found by the new tests:** teachatty's own self-signed certificate triggered the client warning. `CheckSignatureFrom` refuses a signer that isn't marked as an authority, so the self-signed check now verifies the signature directly with the certificate's own key.
- **Tested end to end with certbot's layout** (`live/` symlinks into `archive/`, renewal by creating `fullchain2.pem`/`privkey2.pem` with a new key and repointing the links): the server picked up the renewal without a restart and the client kept connecting with nothing pinned. Also: `cert.pem` by mistake (startup warning), connecting by IP (client warning), default self-signed (no warnings).
- `cmd/server/main_test.go` (deleted from the working tree, never committed as deleted) was restored.
- **Docs:** README, roadmap, both help screens (`-h` also skips empty defaults and shows on/off flags as `off`), and the HTML security section.

## 2026-10-12 14:28 — TLS by default, with trust on first use

- Traffic is encrypted with TLS 1.3 by default. No certificates to manage: the server generates a self-signed one on first start, and clients trust a server the first time they connect, like SSH.
- **Server:** on first start, creates `~/.config/teachatty/server.crt` and `server.key` (ECDSA P-256, valid 10 years; the key is mode 600) and reuses them. The startup line now ends with `· TLS · fingerprint SHA256:...`.
- **Client:** connects over `wss://`. The first connection to a server (keyed by what you typed in `--server`) saves its fingerprint in `~/.config/teachatty/known_servers` and shows `* trusting new server ...`: in gray inside the chat, or on stderr in pipe mode. Later connections are silent. A different fingerprint is refused with an SSH-style warning showing both fingerprints, and exit code 1.
- New servers are accepted without asking (like SSH's `accept-new`): a yes/no question would block pipe mode and cron.
- The fingerprint pins the public key (`SHA256:` + base64 of the hash of the key), so a certificate renewed with the same key keeps it.
- `--no-tls` on both binaries for plain `ws://` (testing). If only one side uses TLS, the error comes with a hint (`try --no-tls` / `drop --no-tls`).
- **All TLS code lives in `internal/tlstrust`**, standard library only (`crypto/tls`, `crypto/x509`). The binaries only call `tlstrust.Server(...)` and `tlstrust.Client(server, notice)`.
- **Bug found while testing:** on a first connection, the full screen never opened. The notice was shown with `Print`, which queues the update and waits for tview's event loop, and before `Run` there's no loop yet, so `main` blocked forever. Fixed with a new `ui.Notice`, which writes directly and is only called before `Run`.
- **Tests:** `internal/tlstrust` has 3 (the certificate is generated once and reused, with the key at mode 600; trust on first use then silent; a different certificate under the same server name is refused), with `-race`.
- **Tested end to end** with the binaries in a throwaway `HOME`: first start, first and second connection, server restart (same fingerprint), regenerated certificate (refused), TLS on one side only (hint), `--no-tls` on both. `openssl s_client` confirms TLS 1.3 with AES-128-GCM.
- **Docs:** README, roadmap (passwords are now next step 1; TLS decisions recorded), and the HTML: a new "Security: TLS and trust on first use" section, flags and exit codes, the connection section, layers and repository map, and the "no encryption" limitation replaced by the trust-on-first-use caveat.

## 2026-10-12 13:05 — Everything in English

- Code comments, the site and documentation (`docs/index.html`, `docs/docs.html`), `roadmap.md` and this log are now in English, like the terminal output already was. Older entries below were translated; quoted strings that used to be in Portuguese (`conexão encerrada`, `teachatty-server escutando em ...`) are kept as they were, since that's what the program printed back then.
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

- A stray argument (`teachatty dev`, `teachatty-server start`) was silently ignored: the server started and the client tried to connect. Now both print `unknown argument: ...` and the help, and exit with code 2 without doing anything.
- `teachatty help` and `teachatty-server help` show the help with code 0.
- An unknown flag (`--romm`) already showed the help, through Go's `flag` package.
- Code: `checkArgs()` in each binary's `usage.go`, called right after `flag.Parse()`.
- Docs: exit code table and server reference in `docs.html`, README and roadmap.

## 2026-10-12 11:43 — Terminal output in English

- Everything the program prints to the terminal moved to English: `connection closed` (previously `conexão encerrada`), `error reading input`, the server's startup line (`teachatty-server listening on :7347 · default room: local`), and test names and messages.
- The alias for users without `$USER` changed from `anonimo` to `anonymous`, since it shows up in the chat.
- Code comments, the site and documentation (`docs/`), and these notes stayed in Portuguese at this point.
- The site and documentation were adjusted to quote the new messages, and the client code excerpts were regenerated.

## 2026-10-12 11:38 — Port 7347, -h help and first tests

- **Port 7347** on the server and as the client default (previously 23456). The port became the `addr` constant on the server.
- **`-h` on both binaries**, in English: what it is, how to use it, flags with defaults, keys and examples. Flag descriptions went to English too, since they show up in the help. In the client, defaults come from the flags themselves (`flag.Lookup`/`VisitAll`), so the help follows any change. The server didn't read flags at all: `server -h` started the server. Now it shows the help and exits.
- The server's startup message states the port and default room: `teachatty-server escutando em :7347 · sala padrão: local` (translated to English in the next entry).
- The `/ws` handler moved out of `main` into the `handleWS` function, so the tests can use it. No behavior change.
- **First automated tests** (`cmd/server/main_test.go`, `go test ./...`): the `local` room exists before any connection; connecting without a room lands in it and it survives empty; other rooms disappear when they empty; a message in `local` arrives as `ana: oi`. They run with `-race`. Checked that they catch bugs: removing the `local` protection in `leave` makes a test fail.
- **Tested end to end** with the binaries: a client with no flags opens on `local · del`; another client in `local` receives the message; `echo ... | teachatty` with no flags reaches `local`; everyone leaves, someone joins again, and `local` is still there.
- **Docs:** README, roadmap and the HTML (port, default room `local`, `-h`, tests in the repository map, and the server code regenerated from `handleWS`).

## 2026-10-12 11:29 — Default room `local`

- The server creates the `local` room when it starts, and it's never deleted, even when empty. Other rooms keep appearing and disappearing as before.
- Without `--room`, the client joins `local` (previously `geral`). On the server, a connection with no room in the URL also lands in `local`, instead of a room with an empty name.
- **Why:** just run `teachatty` and everyone on the same server is in the same room, without agreeing on a name.
- **Why permanent:** creating the room at server start would be pointless if it got deleted on the first departure.
- In the code: the `defaultRoom` constant, the `rooms` map starts with it, `cmp.Or` for the empty name, and `leave` doesn't delete the default room.
- **Tested:** a client without `--room` opens with the title `local · ana`; a raw connection with no room receives its messages; leaving and coming back keeps the room working; room `dev` is unchanged. No races on the server.
- **Docs (HTML):** updated in the next entry.

## 2026-10-12 11:14 — Site and documentation up to date

- Clears the pending HTML updates for alias colors, the default alias and pipe mode.
- **Site (`docs/index.html`):** the screen mockup shows aliases in their real colors (computed with the same hash as the client) and the screen's owner in bold. New highlight "Each person gets a color". The pipes card lost its "coming soon" tag and gained an example; "Start in a minute" shows `uptime | teachatty`; pipe mode left "What's next".
- **Documentation (`docs/docs.html`):**
  - Client reference: `--alias` defaulting to `$USER`, colors in the message area, pipe mode exit codes and a new **Pipe mode** subsection (usage, echo, one message per line, no size limit, safe in scripts). Linked from the sidebar.
  - Client internals: new subsections **Colors per alias** (hash, bold, `tview.Escape`, the palette) and **Pipe mode internals** (the two goroutines, why it reads the room, the close at the end, the flooded-room test). Code shown was regenerated from the current files, with the whole `pipe.go`.
  - Architecture: `pipe.go` in the repository map, `golang.org/x/term` in the dependencies, updated line count (~380).
  - Limitations: "pipe without a terminal fails" removed, "large logs flood the room" added; "slow client" now with the measured number (2,082 of 5,000).

## 2026-10-12 10:58 — Pipe mode

- When stdin is not a terminal (`command | teachatty`), the client doesn't open the screen: it sends each line to the room, echoes it to the terminal (like `tee`) and exits when the input ends. Code in `cmd/client/pipe.go`; `main.go` only gained `term.IsTerminal`.
- `tail -f app.log | teachatty --room ops` works as a continuous stream, until Ctrl+C.
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
  - `index.html`, the site: what teachatty is, the screen, the principles, the measured numbers, the animation of how a message travels, how to get started and what's next. A **docs** button in the top bar and the hero.
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
- **Broke:** `echo oi | teachatty` fails, because tview requires a terminal. It comes back with the line-by-line fallback, the first step of pipe mode.
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
- `--server` lets a company point the client at its own teachatty instance.
- Messages only go to people in the sender's room, with the alias in front (`daniel: oi`).
- Joining a room that doesn't exist creates it. A room is deleted when its last member leaves.

## 2026-10-11 09:34 — First commit: WebSocket chat

- Server (`cmd/server`) and client (`cmd/client`) as separate binaries in one module.
- WebSocket with `gorilla/websocket`. The server relays each message to every other connected client.
- The client sends a message on every Enter and shows incoming messages in parallel.
