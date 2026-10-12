# Contributing to teachatty

Thanks for wanting to help. teachatty is a small, minimalist project, and every contribution counts: a bug report, a typo fix, a test, an idea, or a feature.

By contributing, you agree that your contribution is licensed under the project's license, the [GNU AGPL-3.0](LICENSE).

## Ways to help

- **Use it and report what breaks.** Join the public server (`teachatty --server teachatty.com`) or run your own.
- **Report a bug** or **suggest a feature** by opening an [issue](https://github.com/daniel-cmdline/teachatty/issues).
- **Improve the docs**: the [README](README.md) and the site and documentation in [`docs/`](docs/).
- **Send code**: fix a bug, add tests, or pick something from the [roadmap](roadmap.md).

## Before you start

teachatty follows a few principles (see [roadmap.md](roadmap.md)): flags instead of URLs, works with pipes, self-hostable, and **a feature only goes in if it pays for itself**. The roadmap's "Decisions made" section records questions that are already settled, along with the reasons.

For anything bigger than a small fix, **open an issue first** and describe what you'd like to change and why. Agreeing on the approach before you write the code saves everyone's time, and it's how we keep the codebase small.

## Reporting bugs

A good bug report includes:

- what you did (the exact commands and flags), what you expected, and what happened instead
- `teachatty --version` and `teachatty-server --version`
- your OS, terminal emulator and how you installed teachatty (`.deb`, `.rpm`, `go install`, from source)
- any error output; for the server, `journalctl -u teachatty-server`

### Security issues

**Please don't open public issues for security vulnerabilities.** Report them privately through GitHub's [private vulnerability reporting](https://github.com/daniel-cmdline/teachatty/security/advisories/new). You'll get a reply, and a fix will be released before the details are made public.

## Development setup

You need Go 1.27 or newer and a UTF-8 terminal.

```bash
git clone https://github.com/daniel-cmdline/teachatty.git
cd teachatty
git switch dev

go build ./...                 # build everything
go test -race ./...            # all tests, with the race detector
go vet ./...                   # static checks
gofmt -l .                     # must print nothing
```

Run it locally (plain `ws://` skips the TLS trust-on-first-use step):

```bash
go run ./cmd/teachatty-server --no-tls
go run ./cmd/teachatty --no-tls --room dev --alias you     # in another terminal
uptime | go run ./cmd/teachatty --no-tls --room dev         # pipe mode
```

Run a single test:

```bash
go test -race -run TestOtherRoomsVanishWhenEmpty ./cmd/teachatty-server
```

If you change the packaging (`.goreleaser.yaml` or `packaging/`), build the packages locally with [GoReleaser](https://goreleaser.com) and try installing them:

```bash
goreleaser check
goreleaser release --snapshot --clean     # output in dist/
```

## Branches

- `main` is the stable branch: what users get with `go install ...@latest`.
- `dev` is where work comes together. **Open pull requests against `dev`.** It's merged into `main` when it's ready to ship.

Create your branch from `dev` in your fork, with a short descriptive name such as `fix/pipe-hangs-on-eof` or `feat/room-list`.

## Making changes

- **Keep it small and focused.** One pull request does one thing. Small PRs are reviewed faster.
- **Format with `gofmt`**, and keep `go vet` clean.
- **Add or update tests** for behavior you change, especially on the server. The tests in [`cmd/teachatty-server/main_test.go`](cmd/teachatty-server/main_test.go) show how to start a test server with `httptest` and connect real WebSocket clients. Run them with `-race`: the server and client are full of goroutines.
- **Comment the why, not the what.** Look at any file: comments explain the reasons behind a choice (why a lock is held, why a limit has that value). New code should read the same way.
- **Everything is in English**: code, comments, messages printed by the programs, and docs.
- **New dependencies need a good reason.** teachatty is a pure Go, single-binary program with a short dependency list, and it should stay that way.
- **Keep the user-facing text in sync.** If you change a flag or a behavior, update the `-h` help (`usage.go` in each `cmd/` folder), the [README](README.md) and the docs in [`docs/`](docs/).
- **Log what you shipped.** Add an entry at the top of [features.md](features.md) describing the change and how you tested it.

## Commit messages

Follow the usual Git conventions:

- A short subject line (about 50 characters, at most 72) in the imperative mood: "Add room listing", not "Added room listing".
- A blank line, then a body explaining **what** changed and **why**, wrapped at 72 columns. Bullet points are fine.
- Reference issues where relevant: `Fixes #42`.

```
Drop members that stop reading after 5 seconds

A laptop that went to sleep filled its network buffers, and broadcast
then blocked forever while holding the room's lock, freezing the room.

- set a write deadline on every broadcast write
- close and remove the member when it expires

Fixes #12
```

## Pull requests

1. Make sure `go test -race ./...`, `go vet ./...` and `gofmt -l .` all pass.
2. Open the PR against `dev`. In the description, say what it changes, why, and how you tested it (for UI changes, a screenshot or a short recording helps).
3. Link the issue it addresses.
4. Be ready for review comments. Discussion is part of the process, and suggestions are about the code, never about you.

## Be kind

Be respectful and patient with everyone. We're all here to make a small, good tool together. Harassment or personal attacks aren't tolerated.
