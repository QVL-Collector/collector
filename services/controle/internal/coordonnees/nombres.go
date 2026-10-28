package coordonnees

import (
	"regexp"
	"strconv"
	"strings"
)

const numberWord = `(?:zero|un|deux|trois|quatre|cinq|six|sept|huit|neuf|dix|onze|douze|treize|quatorze|quinze|seize|vingts?|trente|quarante|cinquante|soixante|et)`

var spelledNumbers = regexp.MustCompile(`\b` + numberWord + `(?:[ -]+` + numberWord + `)*\b`)

var numberWordValues = map[string]int{
	"zero": 0, "un": 1, "deux": 2, "trois": 3, "quatre": 4,
	"cinq": 5, "six": 6, "sept": 7, "huit": 8, "neuf": 9,
	"dix": 10, "onze": 11, "douze": 12, "treize": 13, "quatorze": 14,
	"quinze": 15, "seize": 16, "vingt": 20, "vingts": 20, "trente": 30,
	"quarante": 40, "cinquante": 50, "soixante": 60,
}

type spelledNumber struct {
	value int
	last  int
}

func spellOutNumbers(normalized string) string {
	return spelledNumbers.ReplaceAllStringFunc(normalized, digitsOfWords)
}

func digitsOfWords(words string) string {
	numbers := parseNumbers(strings.FieldsFunc(words, isWordSeparator))
	if len(numbers) == 0 {
		return words
	}
	digits := make([]string, len(numbers))
	for i, number := range numbers {
		digits[i] = strconv.Itoa(number.value)
	}
	return strings.Join(digits, " ")
}

func isWordSeparator(r rune) bool {
	return r == ' ' || r == '-'
}

func parseNumbers(words []string) []spelledNumber {
	var numbers []spelledNumber
	for _, word := range words {
		value, isNumber := numberWordValues[word]
		if !isNumber {
			continue
		}
		last := len(numbers) - 1
		if last >= 0 && numbers[last].accepts(value) {
			numbers[last] = numbers[last].extendedBy(value)
			continue
		}
		numbers = append(numbers, spelledNumber{value: value, last: value})
	}
	return numbers
}

func (n spelledNumber) accepts(value int) bool {
	switch {
	case n.value == 4 && value == 20:
		return true
	case isTens(n.last):
		return isUnit(value) || n.takesTeens() && isTeen(value)
	case n.last == 10:
		return value >= 7 && value <= 9
	default:
		return false
	}
}

func (n spelledNumber) takesTeens() bool {
	return n.value == 60 || n.value == 80
}

func (n spelledNumber) extendedBy(value int) spelledNumber {
	if n.value == 4 && value == 20 {
		return spelledNumber{value: 80, last: 20}
	}
	return spelledNumber{value: n.value + value, last: value}
}

func isUnit(value int) bool {
	return value >= 1 && value <= 9
}

func isTeen(value int) bool {
	return value >= 10 && value <= 16
}

func isTens(value int) bool {
	return value >= 20 && value <= 60 && value%10 == 0
}
