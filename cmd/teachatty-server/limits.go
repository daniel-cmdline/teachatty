package main

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// Limits that keep one person with a script from taking the server down. A company
// network trusts its coworkers; the open internet doesn't.

// vars, not consts, so the tests can loosen them
var (
	// messages per connection: 5 a second, with bursts of up to 20 (pasting a few lines,
	// a pipe catching up). Going over doesn't drop anything or disconnect anyone: the
	// server just reads that connection more slowly, and TCP slows the sender down.
	msgRate  rate.Limit = 5
	msgBurst            = 20

	// bytes per connection: 256 KB a second, with bursts up to one maximum-size message,
	// so 1 MB messages can't be sent back to back (each one is copied to every member)
	byteRate rate.Limit = 256 << 10
)

// limiters returns one connection's message and byte budgets
func limiters() (msgs, bytes *rate.Limiter) {
	return rate.NewLimiter(msgRate, msgBurst), rate.NewLimiter(byteRate, maxMessage)
}

// noPipes refuses clients that declare pipe mode (--no-pipes)
var noPipes = false

// maxPerIP caps simultaneous connections from one IP address (each costs ~17 KB of
// memory). Set by --max-per-ip; 0 means no cap, for a company behind NAT where everyone
// reaches the server from the same address.
var maxPerIP = 10

var (
	ipMu  sync.Mutex
	perIP = map[string]int{}
)

// acquireIP reserves a connection slot for the request's IP. It returns a release
// function, or false if that IP is already at the cap.
func acquireIP(r *http.Request) (release func(), ok bool) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	ipMu.Lock()
	defer ipMu.Unlock()
	if maxPerIP > 0 && perIP[ip] >= maxPerIP {
		return nil, false
	}
	perIP[ip]++
	return func() {
		ipMu.Lock()
		defer ipMu.Unlock()
		if perIP[ip]--; perIP[ip] <= 0 {
			delete(perIP, ip)
		}
	}, true
}
