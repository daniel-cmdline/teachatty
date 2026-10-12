package main

import (
	"flag"
	"fmt"
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
