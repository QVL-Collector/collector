package token

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	envIssuer              = "OIDC_EMETTEUR"
	envKeySetURL           = "OIDC_JEU_DE_CLES_URL"
	envAudience            = "OIDC_AUDIENCE"
	envAuthorizedParty     = "OIDC_CLIENT_AUTORISE"
	defaultAuthorizedParty = "collector-web"
	keycloakKeySetPath     = "/protocol/openid-connect/certs"
)

func LoadConfig(lookup func(name string) string, defaultAudience string) (Config, error) {
	issuer := strings.TrimSuffix(strings.TrimSpace(lookup(envIssuer)), "/")
	if issuer == "" {
		return Config{}, fmt.Errorf("%s absent", envIssuer)
	}
	if _, err := url.ParseRequestURI(issuer); err != nil {
		return Config{}, fmt.Errorf("%s n'est pas une URL absolue", envIssuer)
	}
	return Config{
		Issuer:          issuer,
		Audience:        valueOrDefault(lookup(envAudience), defaultAudience),
		AuthorizedParty: valueOrDefault(lookup(envAuthorizedParty), defaultAuthorizedParty),
		KeySetURL:       valueOrDefault(lookup(envKeySetURL), issuer+keycloakKeySetPath),
	}, nil
}

func valueOrDefault(raw, fallback string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return fallback
}
