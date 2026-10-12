package main

import (
	"flag"
	"fmt"
	"os"
)

// usage é a ajuda do -h, em inglês para servir a qualquer usuário. Os padrões vêm
// das próprias flags (flag.VisitAll), então a ajuda nunca fica desatualizada.
func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprint(out, `teachat: a terminal chat

Usage:
  teachat [flags]               open the full-screen chat
  command | teachat [flags]     pipe mode: send each line to the room and exit

Flags:
`)
	flag.VisitAll(func(f *flag.Flag) {
		fmt.Fprintf(out, "  --%-8s %s\n  %-10s default: %s\n", f.Name, f.Usage, "", f.DefValue)
	})
	fmt.Fprint(out, `
Keys (full screen):
  Enter        send
  PgUp/PgDn    scroll messages; typing jumps back to the newest
  Ctrl+C       quit

Examples:
  teachat                                    join the local room on this machine's server
  teachat --server chat.company:7347 --room dev
  make 2>&1 | teachat --room ops             send the build output to the room
`)
}

// checkArgs trata o que sobrou depois das flags. Os dois binários não usam argumentos
// soltos, então qualquer um (ex.: "teachat start") não ativaria nada: mostra a ajuda
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
