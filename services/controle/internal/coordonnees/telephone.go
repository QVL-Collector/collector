package coordonnees

import (
	"regexp"
	"strings"
)

const maxPhoneDigits = 13

var (
	separatorWord = regexp.MustCompile(`\b(?:point|tiret|dot)\b`)
	digitRun      = regexp.MustCompile(`\d+(?:[ .\-_()]{1,4}\d+)*`)
	digitGroup    = regexp.MustCompile(`\d+`)
	yearGroup     = regexp.MustCompile(`^(?:19|20)\d{2}$`)
)

var dialPrefixByLength = map[int]string{
	10: "0",
	11: "33",
	12: "330",
	13: "0033",
}

func containsPhoneNumber(normalized string) bool {
	digits := separatorWord.ReplaceAllString(spellOutNumbers(normalized), ".")
	for _, run := range digitRun.FindAllString(digits, -1) {
		if containsPhoneSequence(digitGroup.FindAllString(run, -1)) {
			return true
		}
	}
	return false
}

func containsPhoneSequence(groups []string) bool {
	for start := range groups {
		if startsWithPhoneNumber(groups[start:]) {
			return true
		}
	}
	return false
}

func startsWithPhoneNumber(groups []string) bool {
	var digits strings.Builder
	for _, group := range groups {
		if yearGroup.MatchString(group) {
			return false
		}
		digits.WriteString(group)
		if digits.Len() > maxPhoneDigits {
			return false
		}
		if isPhoneNumber(digits.String()) {
			return true
		}
	}
	return false
}

func isPhoneNumber(digits string) bool {
	prefix, known := dialPrefixByLength[len(digits)]
	return known && strings.HasPrefix(digits, prefix) && digits[len(prefix)] != '0'
}
