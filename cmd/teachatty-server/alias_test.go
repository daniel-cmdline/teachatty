package main

// Alias sanitization (alias.go): the display name policy applied once, at the
// WebSocket boundary, before the alias reaches any room, relay or screen.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

func TestSanitizeAliasOrdinaryName(t *testing.T) {
	if got := sanitizeAlias("ana"); got != "ana" {
		t.Errorf("sanitizeAlias(ana) = %q, want ana unchanged", got)
	}
}

func TestSanitizeAliasKeepsScriptsEmojiAndPunctuation(t *testing.T) {
	want := "bêta 🙂·გამარჯობა"
	if got := sanitizeAlias(want); got != want {
		t.Errorf("sanitizeAlias kept %q, want %q unchanged (accents, emoji, punctuation, non-Latin script)", got, want)
	}
	if got := sanitizeAlias("a͏b"); got != "a͏b" { // combining overline (not a control rune)
		t.Errorf("sanitizeAlias(broken combining) = %q, want unchanged", got)
	}
}

func TestSanitizeAliasRemovesControlRunes(t *testing.T) {
	// C0 (incl. newline, CR, ESC and BEL), C1 and DEL never relay
	for _, tc := range []struct {
		name  string
		alias string
		want  string
	}{
		{"newline", "bad\nalias", "badalias"},
		{"carriage return", "bad\ralias", "badalias"},
		{"escape", "bad\x1b[31malias", "bad[31malias"},
		{"bell", "bad\x07alias", "badalias"},
		{"delete", "bad\x7falias", "badalias"},
		{"c1 control", "bad\x85alias", "badalias"},
		{"vertical tab", "bad\x0balias", "badalias"},
	} {
		if got := sanitizeAlias(tc.alias); got != tc.want {
			t.Errorf("%s: sanitizeAlias(%q) = %q, want %q", tc.name, tc.alias, got, tc.want)
		}
	}
}

func TestSanitizeAliasAtLimitStaysIntact(t *testing.T) {
	atLimit := strings.Repeat("a", maxAlias)
	if got := sanitizeAlias(atLimit); got != atLimit {
		t.Errorf("sanitizeAlias(%d bytes) = %d bytes, want intact", maxAlias, len(got))
	}
}

func TestSanitizeAliasTruncatesASCII(t *testing.T) {
	got := sanitizeAlias(strings.Repeat("a", 1000))
	if len(got) != maxAlias {
		t.Errorf("sanitizeAlias(1000 a's) = %d bytes, want %d", len(got), maxAlias)
	}
	if got != strings.Repeat("a", maxAlias) {
		t.Errorf("truncate changed the prefix: %q", got[:12])
	}
}

func TestSanitizeAliasTruncatesOnRuneBoundary(t *testing.T) {
	// 195 a's + emoji: the first emoji lands at bytes 195-198 (fits, exactly
	// ending the 199-byte prefix), the second would overshoot 200 — byte-level
	// truncation would split it in half; rune-boundary truncation cannot.
	alias := strings.Repeat("a", 195) + "🙂🙂🙂"
	got := sanitizeAlias(alias)
	if len(got) != 199 {
		t.Errorf("got %d bytes, want the complete 199-byte prefix (195 a's + one 4-byte emoji)", len(got))
	}
	if !utf8.ValidString(got) {
		t.Fatalf("truncation produced invalid UTF-8: %q", got)
	}
	if !strings.HasSuffix(got, "🙂") {
		t.Errorf("cut split the emoji: %q...%q", got[:8], got[len(got)-8:])
	}
	// a multibyte alias aiming exactly at the limit stays intact
	atLimit := strings.Repeat("é", 100) // 100 runes × 2 bytes = 200 bytes
	if got := sanitizeAlias(atLimit); got != atLimit {
		t.Errorf("sanitizeAlias(200-byte multibyte alias) = %d bytes, want intact", len(got))
	}
	if got := sanitizeAlias(strings.Repeat("é", 101)); len(got) != 200 {
		t.Errorf("sanitizeAlias(202-byte multibyte alias) = %d bytes, want 200 (whole runes)", len(got))
	}
	if !utf8.ValidString(sanitizeAlias(strings.Repeat("é", 101))) {
		t.Fatal("multibyte truncation produced invalid UTF-8")
	}
	// raw bytes that are not valid UTF-8 (a lone C1 byte, as %85 in a URL)
	// must be dropped, not turned into visible replacement runes
	if got := sanitizeAlias("bad" + string(rune(0x85))); got != "bad" {
		t.Errorf("sanitizeAlias of a raw C1 byte = %q, want %q", got, "bad")
	}
}

func TestSanitizeAliasEmptyBecomesAnonymous(t *testing.T) {
	for _, tc := range []struct {
		name  string
		alias string
	}{
		{"empty flag", ""},
		{"only newline", "\n"},
		{"only controls", "\x1b\x07\r"},
	} {
		if got := sanitizeAlias(tc.alias); got != anonymous {
			t.Errorf("%s: sanitizeAlias(%q) = %q, want %q", tc.name, tc.alias, got, anonymous)
		}
	}
	if got := sanitizeAlias("ana\x1b"); got != "ana" {
		t.Errorf("sanitizeAlias(ana+ESC) = %q, want ana (sanitize+truncate combine)", got)
	}
}

// ---- handler integration: unsafe values cannot reach the room as sent ----

func connectAudit(t *testing.T, srv *httptest.Server, query string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?" + query
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func readAlone(t *testing.T, conn *websocket.Conn, timeout time.Duration) string {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(msg)
}

func auditServer(t *testing.T) *httptest.Server {
	t.Helper()
	noRateLimit(t)
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	t.Cleanup(srv.Close)
	return srv
}

func TestHandlerControlAliasSanitized(t *testing.T) {
	srv := auditServer(t)
	watcher := connectAudit(t, srv, "room=aliasroom&alias=watcher")
	eventually(t, "watcher joins", func() bool { return members("aliasroom") == 1 })

	bad := connectAudit(t, srv, "room=aliasroom&alias=bad%0Aalias")
	eventually(t, "bad alias joins", func() bool { return members("aliasroom") == 2 })

	bad.WriteMessage(websocket.TextMessage, []byte("sanity"))
	got := readAlone(t, watcher, 2*time.Second)
	if got != "badalias: sanity" {
		t.Errorf("relay %q, want sanitized 'badalias: sanity'", got)
	}
	bad.Close()
	watcher.Close()
	eventually(t, "room empties", func() bool { return members("aliasroom") == -1 })
}

func TestHandlerEscAndC1AliasSanitized(t *testing.T) {
	srv := auditServer(t)
	watcher := connectAudit(t, srv, "room=escroom&alias=watcher")
	eventually(t, "watcher joins", func() bool { return members("escroom") == 1 })

	hostile := connectAudit(t, srv, "room=escroom&alias=%1B[31mred%85")
	eventually(t, "hostile alias joins", func() bool { return members("escroom") == 2 })

	hostile.WriteMessage(websocket.TextMessage, []byte("x"))
	got := readAlone(t, watcher, 2*time.Second)
	if strings.ContainsAny(got, "\x1b\x85\n\r") {
		t.Errorf("relay carries control bytes: %q", []byte(got))
	}
	if !strings.Contains(got, "[31mred: x") {
		t.Errorf("relay %q: printable parts should survive sanitization", got)
	}
	hostile.Close()
	watcher.Close()
	eventually(t, "room empties", func() bool { return members("escroom") == -1 })
}

func TestHandlerHugeAliasCapped(t *testing.T) {
	srv := auditServer(t)
	watcher := connectAudit(t, srv, "room=hugeroom&alias=watcher")
	eventually(t, "watcher joins", func() bool { return members("hugeroom") == 1 })

	huge := connectAudit(t, srv, "room=hugeroom&alias="+strings.Repeat("a", 1000000))
	eventually(t, "huge alias joins", func() bool { return members("hugeroom") == 2 })

	huge.WriteMessage(websocket.TextMessage, []byte("hi"))
	got := readAlone(t, watcher, 2*time.Second)
	// a 3-byte text relays within a few dozen bytes, not a million
	if len(got) > maxAlias+len("watcher: hi")+4 {
		t.Errorf("relay is %d bytes for a 2-byte text: alias not capped", len(got))
	}
	if got[:maxAlias] != strings.Repeat("a", maxAlias) {
		t.Errorf("capped alias lost its first %d bytes", maxAlias)
	}
	huge.Close()
	watcher.Close()
	eventually(t, "room empties", func() bool { return members("hugeroom") == -1 })
}
