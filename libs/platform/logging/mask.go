package logging

import (
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

const maskedValue = "[masqué]"

var sensitiveKeyFragments = []string{
	"authorization",
	"cookie",
	"jeton",
	"token",
	"password",
	"passwd",
	"mot_de_passe",
	"secret",
	"credential",
	"signature",
	"dsn",
	"url_base_de_donnees",
	"chaine_de_connexion",
	"connection_string",
}

var (
	signedTokenPattern       = regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*`)
	authorizationSchemeValue = regexp.MustCompile(`(?i)\b((?:Bearer|Basic)\s+)[A-Za-z0-9._~+/=-]+`)
	presignedParameterFormat = regexp.MustCompile(`(?i)(X-Amz-(?:Credential|Signature|Security-Token)=)[^&\s"'\\]+`)
	secretParameterFormat    = regexp.MustCompile(`(?i)\b((?:password|passwd|pwd|mot_de_passe|secret|client_secret|access_token|refresh_token|token|jeton)=)[^&\s"'\\;]+`)
	connectionPasswordFormat = regexp.MustCompile(`(://[^:/@\s]+:)[^@\s]+@`)
)

func maskSensitive(_ []string, attr slog.Attr) slog.Attr {
	if sensitiveKey(attr.Key) {
		return slog.String(attr.Key, maskedValue)
	}
	if attr.Value.Kind() == slog.KindString {
		return slog.String(attr.Key, maskSecrets(attr.Value.String()))
	}
	if attr.Value.Kind() == slog.KindAny {
		return maskAny(attr)
	}
	return attr
}

func sensitiveKey(key string) bool {
	normalized := strings.ReplaceAll(strings.ToLower(key), "-", "_")
	return slices.ContainsFunc(sensitiveKeyFragments, func(fragment string) bool {
		return strings.Contains(normalized, fragment)
	})
}

func maskAny(attr slog.Attr) slog.Attr {
	switch value := attr.Value.Any().(type) {
	case http.Header:
		return slog.Attr{Key: attr.Key, Value: multiValueGroup(value)}
	case url.Values:
		return slog.Attr{Key: attr.Key, Value: multiValueGroup(value)}
	case error:
		return slog.String(attr.Key, maskSecrets(value.Error()))
	case fmt.Stringer:
		return slog.String(attr.Key, maskSecrets(value.String()))
	default:
		return attr
	}
}

func multiValueGroup(values map[string][]string) slog.Value {
	attrs := make([]slog.Attr, 0, len(values))
	for _, name := range slices.Sorted(maps.Keys(values)) {
		attrs = append(attrs, slog.String(name, strings.Join(values[name], ", ")))
	}
	return slog.GroupValue(attrs...)
}

func maskSecrets(text string) string {
	masked := signedTokenPattern.ReplaceAllString(text, maskedValue)
	masked = authorizationSchemeValue.ReplaceAllString(masked, "${1}"+maskedValue)
	masked = presignedParameterFormat.ReplaceAllString(masked, "${1}"+maskedValue)
	masked = secretParameterFormat.ReplaceAllString(masked, "${1}"+maskedValue)
	return connectionPasswordFormat.ReplaceAllString(masked, "${1}"+maskedValue+"@")
}
