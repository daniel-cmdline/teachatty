// Package version tells which build of teachat is running, for --version.
package version

import "runtime/debug"

// version can be set at build time, which is what release builds do:
//
//	go build -ldflags "-X github.com/daniel-cmdline/teachat/internal/version.version=v0.1.0"
var version = ""

// String returns the version, from the first source that has one:
//  1. set at build time with -ldflags (release builds)
//  2. the version Go stamps into the binary: the tag when installed with
//     go install ...@v0.1.0, or, when built from a clone, a pseudo-version with the
//     date and commit (v0.0.0-20261012174815-d2eea5cc802b, "+dirty" if there were
//     uncommitted changes)
//  3. "dev", when Go doesn't stamp one (go run)
func String() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
