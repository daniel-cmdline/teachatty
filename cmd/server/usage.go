package main

import (
	"flag"
	"fmt"
	"os"
)

// usage é a ajuda do -h, em inglês para servir a qualquer usuário. Como no cliente,
// os padrões vêm das próprias flags (flag.VisitAll).
func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprint(out, `teachat-server: the server for teachat, a terminal chat

Usage:
  teachat-server [flags]

Flags:
`)
	flag.VisitAll(func(f *flag.Flag) {
		fmt.Fprintf(out, "  --%-8s %s\n  %-10s default: %s\n", f.Name, f.Usage, "", f.DefValue)
	})
	fmt.Fprintf(out, `
Rooms live in memory: joining a room that doesn't exist creates it, and it
disappears when the last member leaves. The %q room is the exception: it
starts with the server and exists as long as the server runs. Anyone who
connects without choosing a room lands there.

Messages larger than %d MB are refused and that connection is closed.
Traffic is not encrypted (ws://): only use it on networks you trust.

Examples:
  teachat-server                       listen on port 7347, all interfaces
  teachat-server --addr :9000          listen on port 9000
  teachat-server --addr 10.0.0.5:7347  listen only on the internal network
`, defaultRoom, maxMessage>>20)
}

// checkArgs trata o que sobrou depois das flags. Os dois binários não usam argumentos
// soltos, então qualquer um (ex.: "teachat-server start") não ativaria nada: mostra a ajuda
// em vez de seguir em silêncio. "help" é tratado como pedido de ajuda.
// Flag que não existe já cai na ajuda sozinha, pelo pacote flag.
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
