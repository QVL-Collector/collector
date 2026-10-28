package coordonnees

import (
	"regexp"
	"strings"
)

const (
	maxPhoneDigits = 13
	dateMarker     = "|"
)

var (
	separatorWord      = regexp.MustCompile(`\b(?:point|tiret|dot)\b`)
	slashedPhoneNumber = regexp.MustCompile(`\b0[1-9](?:\s?/\s?\d{2}){4}\b`)
	fullDate           = regexp.MustCompile(`\b(?:\d{1,2}[./-]\d{1,2}[./-](?:19|20)\d{2}|(?:19|20)\d{2}[./-]\d{1,2}[./-]\d{1,2})\b`)
	digitRun           = regexp.MustCompile(`\d+(?:[ .\-_()]{1,4}\d+)*`)
	digitGroup         = regexp.MustCompile(`\d+`)
)

var dialPrefixByLength = map[int]string{
	10: "0",
	11: "33",
	12: "330",
	13: "0033",
}

func containsPhoneNumber(normalized string) bool {
	digits := separatorWord.ReplaceAllString(spellOutNumbers(normalized), ".")
	if slashedPhoneNumber.MatchString(digits) {
		return true
	}
	for _, run := range digitRun.FindAllString(fullDate.ReplaceAllString(digits, dateMarker), -1) {
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
