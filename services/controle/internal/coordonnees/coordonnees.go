package coordonnees

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func Detected(text string) bool {
	normalized := normalize(text)
	return containsAddress(normalized) || containsPhoneNumber(normalized)
}

func normalize(text string) string {
	var normalized strings.Builder
	normalized.Grow(len(text))
	for _, r := range norm.NFKD.String(text) {
		if invisible(r) {
			continue
		}
		normalized.WriteRune(canonical(r))
	}
	return normalized.String()
}

func invisible(r rune) bool {
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Cf, r)
}

func canonical(r rune) rune {
	if unicode.IsSpace(r) {
		return ' '
	}
	return unicode.ToLower(r)
}
