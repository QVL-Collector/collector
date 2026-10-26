package logging

import (
	"log/slog"
	"regexp"
	"strings"
)

const maskedValue = "[masqué]"

var sensitiveKeys = map[string]struct{}{
	"authorization":          {},
	"x-collector-user-token": {},
	"jeton":                  {},
	"mot_de_passe":           {},
	"secret":                 {},
	"dsn":                    {},
	"url_base_de_donnees":    {},
}

var (
	signedTokenPattern       = regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*`)
	connectionPasswordFormat = regexp.MustCompile(`(://[^:/@\s]+:)[^@\s]+@`)
	presignedParameterFormat = regexp.MustCompile(`(?i)(X-Amz-(?:Credential|Signature|Security-Token)=)[^&\s"'\\]+`)
)

func maskSensitive(_ []string, attr slog.Attr) slog.Attr {
	if _, sensitive := sensitiveKeys[strings.ToLower(attr.Key)]; sensitive {
		return slog.String(attr.Key, maskedValue)
	}
	if attr.Value.Kind() == slog.KindString {
		return slog.String(attr.Key, maskSecrets(attr.Value.String()))
	}
	if err, isError := attr.Value.Any().(error); isError {
		return slog.String(attr.Key, maskSecrets(err.Error()))
	}
	return attr
}

func maskSecrets(text string) string {
	withoutTokens := signedTokenPattern.ReplaceAllString(text, maskedValue)
	withoutSignatures := presignedParameterFormat.ReplaceAllString(withoutTokens, "${1}"+maskedValue)
	return connectionPasswordFormat.ReplaceAllString(withoutSignatures, "${1}"+maskedValue+"@")
}
