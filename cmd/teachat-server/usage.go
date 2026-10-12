package main

import (
	"flag"
	"fmt"
	"os"
)

// usage is the -h help. As in the client, defaults come from the flags themselves
// (flag.VisitAll).
func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprint(out, `teachat-server: the server for teachat, a terminal chat

Usage:
  teachat-server [flags]

Flags:
`)
	flag.VisitAll(func(f *flag.Flag) {
		fmt.Fprintf(out, "  --%-8s %s\n", f.Name, f.Usage)
		// no default line when there's nothing to show; on/off flags read better as "off"
		switch f.DefValue {
		case "":
		case "false":
			fmt.Fprintf(out, "  %-10s default: off\n", "")
		default:
			fmt.Fprintf(out, "  %-10s default: %s\n", "", f.DefValue)
		}
	})
	fmt.Fprintf(out, `
Rooms live in memory: joining a room that doesn't exist creates it, and it
disappears when the last member leaves. The %q room is the exception: it
starts with the server and exists as long as the server runs. Anyone who
connects without choosing a room lands there.

Messages larger than %d MB are refused and that connection is closed. Each
connection may send 5 messages a second (bursts of 20) and 256 KB a second;
above that the server reads it more slowly, so senders are slowed down, not cut
off. Each IP may hold --max-per-ip connections; use 0 behind a company NAT.

Traffic is encrypted with TLS by default. On first start the server creates
a self-signed certificate in ~/.config/teachat/ and prints its fingerprint
on every start. Clients trust it the first time they connect and refuse a
different one afterwards, like SSH. Share the fingerprint so people can
check it. --no-tls serves plain ws:// (testing only).

With a real certificate (Let's Encrypt, your company's CA), pass it with
--tls-cert and --tls-key. Clients that trust that authority verify it like
a browser, with no pinning, so renewals just work. The files are reloaded
when they change, so certbot renewals need no restart.

Examples:
  teachat-server                       listen on port 7347, all interfaces
  teachat-server --addr :9000          listen on port 9000
  teachat-server --addr 10.0.0.5:7347  listen only on the internal network
  teachat-server --tls-cert /etc/letsencrypt/live/chat.example.com/fullchain.pem \
                 --tls-key  /etc/letsencrypt/live/chat.example.com/privkey.pem
`, defaultRoom, maxMessage>>20)
}

// checkArgs handles whatever is left after the flags. Neither binary takes positional
// arguments, so any of them (e.g. "teachat-server start") would do nothing: show the help
// instead of carrying on silently. "help" is treated as a request for help.
// An unknown flag already lands on the help by itself, through the flag package.
func checkArgs() {
	if flag.NArg() == 0 {
		return
	}
	if flag.Arg(0) == "help" {
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()
		os.Exit(0)
	}
	fmt.Fprintf(flag.CommandLine.Output(), "unknown argument: %s\n\n", flag.Arg(0))
	flag.Usage()
	os.Exit(2)
}
