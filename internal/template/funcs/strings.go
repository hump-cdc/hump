package funcs

import (
	"fmt"
	"math/rand"
	"strings"
	"unicode"
)

// tmplTrim removes leading and trailing whitespace from s.
func tmplTrim(s string) string { return strings.TrimSpace(s) }

// tmplTrimAll removes any character contained in cutset from both ends of s.
// Sprig argument order is (cutset, s) for pipeline use.
func tmplTrimAll(cutset, s string) string { return strings.Trim(s, cutset) }

// tmplTrimPrefix removes prefix from the start of s if present.
func tmplTrimPrefix(prefix, s string) string { return strings.TrimPrefix(s, prefix) }

// tmplTrimSuffix removes suffix from the end of s if present.
func tmplTrimSuffix(suffix, s string) string { return strings.TrimSuffix(s, suffix) }

// tmplSwapCase inverts the case of every cased rune in s.
func tmplSwapCase(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsUpper(r):
			return unicode.ToLower(r)
		case unicode.IsLower(r):
			return unicode.ToUpper(r)
		}
		return r
	}, s)
}

// tmplSubstr returns the substring s[start:end] using rune indices. Negative
// or out-of-range bounds are clamped. When end is negative, it is treated as
// len(runes)+end.
func tmplSubstr(start, end int, s string) string {
	r := []rune(s)
	n := len(r)
	if end < 0 {
		end = n + end
	}
	if start < 0 {
		start = 0
	}
	if end > n {
		end = n
	}
	if start > end {
		return ""
	}
	return string(r[start:end])
}

// tmplTrunc caps s at length runes. Negative length counts from the end:
// trunc -3 "abcdef" → "def".
func tmplTrunc(length int, s string) string {
	r := []rune(s)
	n := len(r)
	if length < 0 {
		if -length >= n {
			return s
		}
		return string(r[n+length:])
	}
	if length >= n {
		return s
	}
	return string(r[:length])
}

// tmplAbbrev shortens s to width runes total, ending with "..." when
// shortened. width must be at least 4; smaller values return s unchanged.
func tmplAbbrev(width int, s string) string {
	r := []rune(s)
	if width < 4 || len(r) <= width {
		return s
	}
	return string(r[:width-3]) + "..."
}

// tmplAbbrevBoth shortens s to maxWidth runes, with "..." on both sides
// starting at offset. offset must be ≥4 and maxWidth must be ≥7.
func tmplAbbrevBoth(offset, maxWidth int, s string) string {
	r := []rune(s)
	n := len(r)
	if maxWidth < 7 || offset < 4 || n <= maxWidth {
		return s
	}
	if offset+maxWidth-3 >= n {
		offset = n - (maxWidth - 3)
	}
	if offset <= 4 {
		return string(r[:maxWidth-3]) + "..."
	}
	// end is always < n here: when the guard above adjusted offset,
	// offset+maxWidth-3 == n so end == n-3; otherwise offset+maxWidth-3 < n
	// so end < n-3. No upper clamp is needed.
	end := offset + maxWidth - 6
	return "..." + string(r[offset:end]) + "..."
}

// tmplNoSpace strips every Unicode whitespace rune from s.
func tmplNoSpace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

// tmplInitials returns the first rune of each whitespace-separated word in s.
func tmplInitials(s string) string {
	var b strings.Builder
	inWord := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			inWord = false
			continue
		}
		if !inWord {
			b.WriteRune(r)
			inWord = true
		}
	}
	return b.String()
}

// tmplWrap word-wraps s so no line exceeds width runes.
func tmplWrap(width int, s string) string { return tmplWrapWith(width, "\n", s) }

// tmplWrapWith word-wraps s so no line exceeds width runes, joining lines
// with sep.
func tmplWrapWith(width int, sep, s string) string {
	if width <= 0 {
		return s
	}
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		words := strings.Fields(line)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		var cur strings.Builder
		curLen := 0
		for _, w := range words {
			wl := len([]rune(w))
			if curLen == 0 {
				cur.WriteString(w)
				curLen = wl
				continue
			}
			if curLen+1+wl > width {
				lines = append(lines, cur.String())
				cur.Reset()
				cur.WriteString(w)
				curLen = wl
				continue
			}
			cur.WriteByte(' ')
			cur.WriteString(w)
			curLen += 1 + wl
		}
		if cur.Len() > 0 {
			lines = append(lines, cur.String())
		}
	}
	return strings.Join(lines, sep)
}

// tmplContains reports whether sub is a substring of s.
func tmplContains(sub, s string) bool { return strings.Contains(s, sub) }

// tmplHasPrefix reports whether s starts with prefix.
func tmplHasPrefix(prefix, s string) bool { return strings.HasPrefix(s, prefix) }

// tmplHasSuffix reports whether s ends with suffix.
func tmplHasSuffix(suffix, s string) bool { return strings.HasSuffix(s, suffix) }

// tmplQuote wraps each value in double quotes, separated by spaces.
func tmplQuote(items ...any) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = fmt.Sprintf("%q", strArg(it))
	}
	return strings.Join(parts, " ")
}

// tmplSquote wraps each value in single quotes, separated by spaces.
func tmplSquote(items ...any) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = "'" + strArg(it) + "'"
	}
	return strings.Join(parts, " ")
}

// tmplCat space-joins all arguments after coercing to string.
func tmplCat(items ...any) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = strArg(it)
	}
	return strings.Join(parts, " ")
}

// tmplIndent prepends spaces count of " " characters to the start of v and
// after every newline, matching Helm/sprig `indent` semantics. The first line
// is also indented; use a leading newline if you want pipeline-style output.
// A nil source is treated as "".
func tmplIndent(spaces int, v any) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.ReplaceAll(strArg(v), "\n", "\n"+pad)
}

// tmplNindent prefixes a newline then indents v with count spaces on every
// line, like sprig nindent.
func tmplNindent(spaces int, v any) string { return "\n" + tmplIndent(spaces, v) }

// tmplRepeat returns v concatenated count times. A non-positive count yields
// the empty string instead of panicking like strings.Repeat would on
// negatives. A nil source is treated as "".
func tmplRepeat(count int, v any) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat(strArg(v), count)
}

// tmplReplace returns s with every occurrence of old replaced by new.
func tmplReplace(old, new, s string) string { return strings.ReplaceAll(s, old, new) }

// tmplPlural picks one when count == 1, otherwise many.
func tmplPlural(one, many string, count int) string {
	if count == 1 {
		return one
	}
	return many
}

const (
	asciiLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	asciiDigits  = "0123456789"
)

func randString(n int, alphabet string) string {
	if n <= 0 || alphabet == "" {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}

// tmplRandAlpha returns a random string of n ASCII letters.
func tmplRandAlpha(n int) string { return randString(n, asciiLetters) }

// tmplRandAlphaNum returns a random string of n ASCII letters and digits.
func tmplRandAlphaNum(n int) string { return randString(n, asciiLetters+asciiDigits) }

// tmplRandNumeric returns a random string of n ASCII digits.
func tmplRandNumeric(n int) string { return randString(n, asciiDigits) }

// tmplRandAscii returns a random string of n printable ASCII characters
// (codes 32-126 inclusive).
func tmplRandAscii(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(32 + rand.Intn(95))
	}
	return string(b)
}

// tmplShuffle returns s with its runes randomly reordered.
func tmplShuffle(s string) string {
	r := []rune(s)
	rand.Shuffle(len(r), func(i, j int) { r[i], r[j] = r[j], r[i] })
	return string(r)
}
