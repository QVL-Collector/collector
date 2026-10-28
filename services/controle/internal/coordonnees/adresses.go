package coordonnees

import (
	"regexp"
	"strings"
)

const (
	label         = `[a-z0-9][a-z0-9_-]*`
	opening       = `\s*[\(\[\{]\s*`
	closing       = `\s*[\)\]\}]\s*`
	writtenDot    = `(?:` + opening + `(?:point|dot|\.)` + closing + `|\s+(?:point|dot)\s+)`
	looseDot      = `(?:\s*\.\s*|` + writtenDot + `)`
	spacedDot     = `(?:\s\.\s?|` + writtenDot + `)`
	atSign        = `(?:\s*@\s*|` + opening + `(?:at|arobase?)` + closing + `|\s*\barobase?\b\s*)`
	mailTLD       = `(?:com|fr|net|org|io|eu|be|ch|uk|de|it|es|ca|me|co|info|biz)`
	webTLD        = `(?:com|fr|net|org|io|eu|be|ch|uk|tv|app|dev|ly|gg|biz|shop|store|xyz|online)`
	spacedWebTLD  = `(?:com|fr|net|org|io|eu)`
	mailProvider  = `(?:gmail|hotmail|yahoo|outlook|orange|free|laposte|sfr|wanadoo|icloud|live|proton(?:mail)?|gmx|aol|msn|bbox)`
	socialNetwork = `(?:instagram|insta|facebook|snapchat|whats\s?app|telegram|tiktok|twitter|messenger|discord|linkedin)`
)

var addressPattern = regexp.MustCompile(strings.Join([]string{
	atSign + label + `(?:` + looseDot + label + `)*` + looseDot + mailTLD + `\b`,
	atSign + mailProvider + `\b`,
	`@[a-z0-9_]{3}`,
	`\b(?:https?|ftp)\s?:\s?/\s?/`,
	`\bwww\b`,
	`\b` + label + `\.` + webTLD + `\b`,
	`\b` + label + spacedDot + spacedWebTLD + `\b`,
	`\b` + socialNetwork + `\b`,
}, "|"))

func containsAddress(normalized string) bool {
	return addressPattern.MatchString(normalized)
}
