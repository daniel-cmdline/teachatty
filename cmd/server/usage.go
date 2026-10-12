package main

import (
	"flag"
	"fmt"
	"os"
)

// usage é a ajuda do -h, em inglês para servir a qualquer usuário
func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), `teachat-server: the server for teachat, a terminal chat

Usage:
  teachat-server

Listens on %s, on all interfaces. Rooms live in memory: joining a room
that doesn't exist creates it, and it disappears when the last member leaves.
The %q room is the exception: it starts with the server and exists as long
as the server runs. Anyone who connects without choosing a room lands there.

No flags yet. The port is fixed and traffic is not encrypted (ws://):
only use it on networks you trust.
`, addr, defaultRoom)
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
