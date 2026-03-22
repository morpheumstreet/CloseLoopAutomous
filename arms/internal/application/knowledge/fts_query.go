package knowledge

import (
	"strings"
	"unicode/utf8"
)

// SanitizeFTS5Query turns free text into a conservative FTS5 OR-of-phrases pattern.
// Returns empty when there is no usable token (caller should fall back to list-only or skip search).
func SanitizeFTS5Query(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	const maxRunes = 512
	if utf8.RuneCountInString(raw) > maxRunes {
		raw = string([]rune(raw)[:maxRunes])
	}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return ""
	}
	var b strings.Builder
	n := 0
	for _, w := range fields {
		w = strings.Trim(w, "\"'`")
		if w == "" {
			continue
		}
		// Drop pure punctuation tokens
		alnum := false
		for _, r := range w {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				alnum = true
				break
			}
		}
		if !alnum {
			continue
		}
		if n > 0 {
			b.WriteString(" OR ")
		}
		b.WriteByte('"')
		b.WriteString(strings.ReplaceAll(w, `"`, `""`))
		b.WriteByte('"')
		n++
	}
	return b.String()
}
