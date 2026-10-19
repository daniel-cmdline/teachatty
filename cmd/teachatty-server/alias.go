package main

import (
	"cmp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxAlias caps each member's display name at 200 UTF-8 bytes: the alias is
// prefixed to every relayed message (alias+": message"), so it must have a
// modest, documented bound instead of leaving sizes to the handshake header
// limit (~1 MB). Bits are kept low so messages stay cheap for every member.
const maxAlias = 200

// anonymous is the alias of someone who sent none, or only controls.
const anonymous = "anonymous"

// sanitizeAlias defines what a display name may look like, at the one point
// aliases enter the room (handleWS). The policy is protocol-neutral on
// purpose: it names a display value, not a wire value, so a future structured
// message protocol can carry it (or its successor) without touching the rooms.
//
//	Control runes - the C0 set (newline, CR, ESC, BEL, ...), C1 and DEL -
//	are removed (unicode.IsControl): they corrupt the "alias: message" line
//	structure or inject terminal control output when relayed verbatim.
//
//	Everything else stays byte-for-byte: accents, emoji, punctuation,
//	whitespace and format characters (like ZWJ) all survive a name in any
//	script, so this is not an ASCII whitelist. Bytes that are not valid
//	UTF-8 (e.g. a lone %85 from a hand-built URL) are not displayable
//	text anywhere and are dropped.
//
//	Then the alias is capped to maxAlias bytes, cutting only between runes.
//
//	An alias that sanitizes to empty joins as `anonymous`.
func sanitizeAlias(alias string) string {
	// the query string may carry raw bytes that are not valid UTF-8 (%85,
	// standalone surrogates, ...): they are not displayable text in any
	// client, so they are dropped before runes are classified (acceptable:
	// strings.Map would turn them into U+FFFD replacement runes instead).
	clean := strings.ToValidUTF8(alias, "")
	clean = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1 // dropped: controls never reach the room
		}
		return r
	}, clean)
	return cmp.Or(cutRunes(clean, maxAlias), anonymous)
}

// cutRunes returns the longest prefix of s of at most max bytes, ending on a
// complete rune. DecodeRuneInString advances by 1 on invalid UTF-8, so the
// loop always moves forward.
func cutRunes(s string, max int) string {
	i := 0
	for i < len(s) {
		_, size := utf8.DecodeRuneInString(s[i:])
		if i+size > max {
			break
		}
		i += size
	}
	return s[:i]
}
