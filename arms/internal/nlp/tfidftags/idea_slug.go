package tfidftags

import "strings"

const (
	defaultMaxSlugTokens = 4
	maxSlugTokensCap     = 8
	defaultMaxSlugRunes  = 48
)

// SlugPrefixFromTags builds a hyphenated slug from the highest-ranked tag tokens (letters/digits only).
// Empty string when no usable tokens remain after sanitization.
func SlugPrefixFromTags(tags []TagScore, maxTokens, maxRunes int) string {
	if maxTokens <= 0 {
		maxTokens = defaultMaxSlugTokens
	}
	if maxTokens > maxSlugTokensCap {
		maxTokens = maxSlugTokensCap
	}
	if maxRunes <= 0 {
		maxRunes = defaultMaxSlugRunes
	}
	var parts []string
	for _, ts := range tags {
		if len(parts) >= maxTokens {
			break
		}
		p := sanitizeSlugPart(ts.Token)
		if len(p) < 2 {
			continue
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		return ""
	}
	s := strings.Join(parts, "-")
	for len(s) > maxRunes {
		if i := strings.LastIndexByte(s, '-'); i > 0 {
			s = s[:i]
		} else {
			s = s[:maxRunes]
			break
		}
	}
	s = strings.Trim(s, "-")
	return s
}

func sanitizeSlugPart(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
