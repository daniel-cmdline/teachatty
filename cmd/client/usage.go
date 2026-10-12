package main

import (
	"flag"
	"fmt"
	"os"
)

// usage is the -h help. Defaults come from the flags themselves (flag.VisitAll),
// so the help never drifts out of date.
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

// checkArgs handles whatever is left after the flags. Neither binary takes positional
// arguments, so any of them (e.g. "teachat start") would do nothing: show the help
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
