package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Tests for pipe mode (cmd | teachatty). They pin the behavior documented in
// roadmap.md: contiguous lines batch into one message, EOF flushes a final
// line without a newline, batches split at batchMax bytes, slow input keeps
// its order, blank lines are echoed but not sent, and failure paths report
// "connection closed" and exit 1 instead of hanging.
//
// Normal paths run pipe() in-process (stdin/stdout redirected). Paths that
// reach log.Fatal/os.Exit inside pipe() cannot run in-process (exit kills the
// test binary), so they spawn the test binary itself as a subprocess (the
// standard helper pattern): TestHelperPipeClient dials, calls pipe() and lets
// the real exit codes surface to the parent.

// same 1 MB limit as teachatty-server (cmd/teachatty-server/main.go)
const testServerMaxMessage = 1 << 20

// ---- test server: records what the client sends, with gorilla's default
// close handling unless told otherwise ----

type pipeServer struct {
	t             *testing.T
	srv           *httptest.Server
	mu            sync.Mutex
	client        *websocket.Conn // the pipe-mode conn, for tests to kill it
	msgC          chan string     // every message the client sends, in arrival order
	clientClosed  chan struct{}   // closed when the client's handler returns
	closeReceived chan int        // close-frame codes received FROM the client
	noAck         bool            // suppress gorilla's automatic close ack
}

func newPipeServer(t *testing.T, noAck bool) *pipeServer {
	s := &pipeServer{
		t:             t,
		msgC:          make(chan string, 4096),
		clientClosed:  make(chan struct{}),
		closeReceived: make(chan int, 8),
		noAck:         noAck,
	}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handler))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *pipeServer) addr() string {
	return strings.TrimPrefix(s.srv.Listener.Addr().String(), "http://")
}

func (s *pipeServer) handler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != "" { // like teachatty-server: no browsers
		http.Error(w, "no origin", http.StatusForbidden)
		return
	}
	conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(testServerMaxMessage) // 1 MB, like teachatty-server
	if s.noAck {                            // make a missing close ack possible
		conn.SetCloseHandler(func(code int, text string) error { return nil })
	}
	if r.URL.Query().Get("mode") == "pipe" {
		s.mu.Lock()
		s.client = conn
		s.mu.Unlock()
	}
	defer conn.Close() // like teachatty-server: FIN when the handler returns
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				code := 0
				if ce, ok := err.(*websocket.CloseError); ok {
					code = ce.Code
				}
				select {
				case s.closeReceived <- code:
				default:
				}
			}
			close(s.clientClosed)
			return
		}
		select {
		case s.msgC <- string(msg):
		default:
			s.t.Error("message record channel is full")
		}
	}
}

// killClient closes the client's TCP conn from the server side, like a server
// crash: the client sees the drop but no close frame.
func (s *pipeServer) killClient() {
	s.mu.Lock()
	c := s.client
	s.mu.Unlock()
	if c == nil {
		s.t.Fatal("client conn not registered")
	}
	c.NetConn().Close()
}

// within runs cond until it passes or the deadline expires; it fails via the
// current *testing.T of the helper rather than needing t directly.
func messagesOrdered(s *pipeServer, n int, timeout time.Duration) []string {
	got := make([]string, 0, n)
	poll := time.NewTimer(timeout)
	defer poll.Stop()
	for len(got) < n {
		select {
		case m := <-s.msgC:
			got = append(got, m)
		case <-poll.C:
			s.t.Helper()
			return nil // caller decides how to fail with partial data
		}
	}
	return got
}

// ---- in-process stdin/stdout redirection ----

func redirectStdin(t *testing.T, feed string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old; r.Close() })
	// write from a goroutine: big feeds exceed the OS pipe buffer and would
	// block until pipe()'s own stdin reader drains them (EOF closes w)
	go func() {
		if _, err := w.WriteString(feed); err != nil {
			t.Error(err)
		}
		w.Close()
	}()
}

type stdoutCapture struct {
	buf  *bytes.Buffer
	r, w *os.File
	old  *os.File
	done chan struct{}
}

// captureStdout redirects os.Stdout into the returned capture. Callers must
// invoke stop() before reading buf; failures in between go through the
// registered cleanup, which also stops it.
func captureStdout(t *testing.T) *stdoutCapture {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	c := &stdoutCapture{buf: new(bytes.Buffer), r: r, w: w, old: os.Stdout, done: make(chan struct{})}
	os.Stdout = w
	go func() { io.Copy(c.buf, r); close(c.done) }()
	t.Cleanup(c.stop)
	return c
}

// stop restores os.Stdout and waits until every written byte is drained.
func (c *stdoutCapture) stop() {
	if os.Stdout == c.w {
		os.Stdout = c.old
	}
	c.w.Close()
	<-c.done
	c.r.Close()
}

// runPipe runs pipe() with a deadline so a bug cannot hang the suite.
func runPipe(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	done := make(chan struct{})
	go func() { defer close(done); pipe(conn) }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("pipe() did not return within 15 s")
	}
}

// noMoreMessages proves nothing else was sent: it waits for the server's
// client-handler to have returned (no further messages can arrive after that)
// and then requires the stream to be empty. Deterministic: no sleeps.
func noMoreMessages(t *testing.T, s *pipeServer) {
	t.Helper()
	select {
	case <-s.clientClosed:
	case <-time.After(10 * time.Second):
		t.Fatal("client conn not closed on the server side within 10 s")
	}
	select {
	case extra := <-s.msgC:
		t.Fatalf("unexpected extra message after the conn closed: %q", extra)
	default:
	}
}

func dialPipe(t *testing.T, s *pipeServer, extraQuery string) *websocket.Conn {
	t.Helper()
	u := "ws://" + s.addr() + "/ws?room=pipeline&alias=pipeclient&mode=pipe" + extraQuery
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// ---- in-process scenarios (fast, deterministic) ----

func TestPipeBatchesContiguousLines(t *testing.T) {
	s := newPipeServer(t, false)
	conn := dialPipe(t, s, "")
	stdout := captureStdout(t)
	redirectStdin(t, "one\ntwo\nthree\n")

	runPipe(t, conn)
	stdout.stop()

	got := messagesOrdered(s, 1, 5*time.Second)
	if len(got) != 1 || got[0] != "one\ntwo\nthree" {
		t.Fatalf("got %q, want one batched message", got)
	}
	noMoreMessages(t, s)
	if out := stdout.buf.String(); !strings.Contains(out, "one") || !strings.Contains(out, "two") || !strings.Contains(out, "three") {
		t.Errorf("echo missing lines: %q", out)
	}
	if code := <-s.closeReceived; code != websocket.CloseNormalClosure {
		t.Errorf("client's close code was %d, want normal closure 1000", code)
	}
}

func TestPipeSendsFinalLineWithoutNewline(t *testing.T) {
	s := newPipeServer(t, false)
	conn := dialPipe(t, s, "")
	captureStdout(t)
	redirectStdin(t, "one\ntwo\nthree") // no trailing newline: EOF flushes it

	runPipe(t, conn)

	got := messagesOrdered(s, 1, 5*time.Second)
	if len(got) != 1 || got[0] != "one\ntwo\nthree" {
		t.Fatalf("got %q, want the final unterminated line sent", got)
	}
	noMoreMessages(t, s)
}

func TestPipeSplitsAtBatchLimit(t *testing.T) {
	s := newPipeServer(t, false)
	conn := dialPipe(t, s, "")
	captureStdout(t)
	// 65535 'a' bytes make size exactly batchMax; "tâil" (5 bytes in UTF-8,
	// 4 runes) must overflow it and force the split. Unicode must survive
	// the join byte-for-byte.
	redirectStdin(t, strings.Repeat("a", 65535)+"\n"+"tâil\n")

	runPipe(t, conn)

	got := messagesOrdered(s, 2, 5*time.Second)
	if len(got) != 2 {
		t.Fatalf("got %d messages, want 2", len(got))
	}
	if len(got[0]) != 65535 {
		t.Errorf("first message %d bytes, want 65535", len(got[0]))
	}
	if got[1] != "tâil" {
		t.Errorf("second message %q, want %q", got[1], "tâil")
	}
	noMoreMessages(t, s)
}

func TestPipeFastBurstKeepsEverything(t *testing.T) {
	s := newPipeServer(t, false)
	conn := dialPipe(t, s, "")
	captureStdout(t)
	var b strings.Builder
	for i := 1; i <= 300; i++ {
		fmt.Fprintf(&b, "i=%d\n", i)
	}
	redirectStdin(t, b.String())

	runPipe(t, conn)

	got := messagesOrdered(s, 1, 5*time.Second)
	if len(got) != 1 {
		t.Fatalf("got %d messages, want 1 for a single burst under batchMax", len(got))
	}
	joined := got[0]
	if n := strings.Count(joined, "i="); n != 300 {
		t.Errorf("delivered %d lines, want 300", n)
	}
	if !strings.Contains(joined, "i=1\n") || !strings.Contains(joined, "i=300") {
		t.Errorf("burst not intact: first=%q last=%q", joined[:8], joined[max(0, len(joined)-8):])
	}
	noMoreMessages(t, s)
}

func TestPipeSlowInputPreservesOrder(t *testing.T) {
	s := newPipeServer(t, false)
	conn := dialPipe(t, s, "")
	captureStdout(t)
	// Gaps far beyond batchWait so the batching timer flushes each line:
	// three separate messages, in input order. Channel bytes in UTF-8.
	lines := []string{"alpha", "bêta", "gamma"}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for _, l := range lines {
			w.WriteString(l + "\n")
			time.Sleep(400 * time.Millisecond) // > batchWait: force a flush
		}
		w.Close()
	}()
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old; r.Close() })

	runPipe(t, conn)

	got := messagesOrdered(s, 3, 5*time.Second)
	if len(got) != 3 || got[0] != "alpha" || got[1] != "bêta" || got[2] != "gamma" {
		t.Fatalf("got %q, want [alpha bêta gamma] in order", got)
	}
	noMoreMessages(t, s)
}

func TestPipeEchoesButDoesNotSendBlankLines(t *testing.T) {
	s := newPipeServer(t, false)
	conn := dialPipe(t, s, "")
	stdout := captureStdout(t)
	redirectStdin(t, "aa\n\nbb\n")

	runPipe(t, conn)

	got := messagesOrdered(s, 1, 5*time.Second)
	if len(got) != 1 || got[0] != "aa\nbb" {
		t.Fatalf("got %q, want a single aa\\nbb message without the blank line", got)
	}
	noMoreMessages(t, s)
	if out := stdout.buf.String(); strings.Count(out, "\n") != 3 {
		t.Errorf("echo should show all 3 stdin lines (blank included), got %q", out)
	}
}

// ---- subprocess scenarios (real os.Exit code paths) ----

// TestHelperPipeClient is not a test: it runs (as -test.run only selects it)
// in a child process playing the pipe client, so log.Fatal/os.Exit inside
// pipe() produce real process exit codes without touching the suite.
func TestHelperPipeClient(t *testing.T) {
	addr := os.Getenv("TEACHATTY_PIPE_TEST_SERVER")
	if addr == "" {
		return // running as part of the normal suite: do nothing
	}
	u := "ws://" + addr + "/ws?room=pipeline&alias=pipeclient&mode=pipe"
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		return // dial failure: exit 0 with empty stderr signals it upward
	}
	fmt.Fprintln(os.Stderr, "HELPER-DIAL-OK")
	pipe(conn)
	os.Exit(0) // main would return here (exit 0) after the normal path
}

// runHelper spawns the test binary as a subprocess and drives it. drive runs
// on the caller's goroutine and must finish its writes; waiting subprocess
// exit (deadline, kill on hang) happens after. The helper result reports the
// child's exit code, its stderr and whether the deadline hit.
type helperResult struct {
	code   int
	stuck  bool
	stderr string
}

func runHelper(t *testing.T, s *pipeServer, drive func(stdin io.WriteCloser)) helperResult {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperPipeClient$")
	cmd.Env = append(os.Environ(), "TEACHATTY_PIPE_TEST_SERVER="+s.addr())
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = errW
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	doneErr := make(chan struct{})
	go func() { io.Copy(&buf, errR); close(doneErr) }()

	drive(stdin)
	stdin.Close() // caller got the drives; EOF whenever drive didn't close it
	errW.Close()
	<-doneErr

	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	exitCh := make(chan int, 1)
	go func() { cmd.Wait(); exitCh <- cmd.ProcessState.ExitCode() }()
	select {
	case code := <-exitCh:
		return helperResult{code: code, stderr: buf.String()}
	case <-deadline.C:
		cmd.Process.Kill()
		<-exitCh
		return helperResult{code: -999, stuck: true, stderr: buf.String()}
	}
}

// awaitMsg waits for one recorded message through a deadline, so a stalled
// helper fails the test instead of hanging it.
func awaitMsg(t *testing.T, s *pipeServer) string {
	t.Helper()
	select {
	case m := <-s.msgC:
		return m
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for a message; server recorded nothing")
		return ""
	}
}

func TestPipeServerDisconnectWhileWaitingExits(t *testing.T) {
	s := newPipeServer(t, false)

	res := runHelper(t, s, func(stdin io.WriteCloser) {
		// establish the conn, then keep stdin open: the client is now
		// waiting for the next input line
		if _, err := stdin.Write([]byte("pre\n")); err != nil {
			t.Fatal(err)
		}
		awaitMsg(t, s) // wait on the channel, not on time: pre was received
		if _, err := stdin.Write([]byte("post\n")); err != nil {
			t.Fatal(err)
		}
		awaitMsg(t, s) // post received: both messages made it before the drop
		// the server dies while the client waits for input
		s.killClient()
	})

	if res.stuck {
		t.Fatalf("helper subprocess hung; stderr=%q", res.stderr)
	}
	if res.code != 1 {
		t.Fatalf("exit %d, want 1; stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "connection closed") {
		t.Fatalf("stderr %q, want a connection-closed report", res.stderr)
	}
}

func TestPipeOversizeSingleLineFailsCleanly(t *testing.T) {
	s := newPipeServer(t, false)

	res := runHelper(t, s, func(stdin io.WriteCloser) {
		// one line over the server's 1 MB limit: the server answers close
		// 1009 (gorilla does this automatically on ReadMessage over the
		// limit) and drops only this connection. The client must report
		// the failure and exit 1, not stall on the giant line.
		line := strings.Repeat("x", testServerMaxMessage+1)
		if _, err := stdin.Write([]byte(line + "\n")); err != nil {
			t.Fatal(err)
		}
	})

	if res.stuck {
		t.Fatalf("helper subprocess hung; stderr=%q", res.stderr)
	}
	if res.code != 1 {
		t.Fatalf("exit %d, want 1; stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "connection closed") {
		t.Fatalf("stderr %q, want a connection-closed report", res.stderr)
	}
}

func TestPipeRefusedCloseHandshakeIsReported(t *testing.T) {
	// gorilla's default close handler auto-acks close frames, so a client's
	// close is normally acked with 1000 on both ends. This test suppresses
	// the ack to verify the honest path when the handshake is interrupted:
	// the abnormal closure is reported, exit 1, no hang. (Not a common case
	// in production - both teachatty binaries use the default handler - but
	// it pins that a dropped close never turns into silent success.)
	s := newPipeServer(t, true)

	res := runHelper(t, s, func(stdin io.WriteCloser) {
		if _, err := stdin.Write([]byte("done\n")); err != nil {
			t.Fatal(err)
		}
	})

	if res.stuck {
		t.Fatalf("helper subprocess hung; stderr=%q", res.stderr)
	}
	if res.code != 1 {
		t.Fatalf("exit %d, want 1; stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "connection closed") {
		t.Fatalf("stderr %q, want a connection-closed report", res.stderr)
	}
}
