package config

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"collector/libs/platform/token"
)

const (
	envAPIAddress            = "ADRESSE_API"
	envManagementAddress     = "ADRESSE_GESTION"
	envDatabaseURL           = "BASE_DE_DONNEES_URL"
	envDatabaseTLS           = "BASE_DE_DONNEES_TLS_EXIGE"
	envIssuer                = "OIDC_EMETTEUR"
	envKeySetURL             = "OIDC_JEU_DE_CLES_URL"
	envAudience              = "OIDC_AUDIENCE"
	envAuthorizedParty       = "OIDC_CLIENT_AUTORISE"
	defaultAPIAddress        = ":8080"
	defaultManagementAddress = ":9090"
	defaultAudience          = "place-de-marche"
	defaultAuthorizedParty   = "collector-web"
	keycloakKeySetPath       = "/protocol/openid-connect/certs"
	verifiedTLSMode          = "verify-full"
)

type Lookup func(name string) string

type Database struct {
	URL string
}

type Server struct {
	APIAddress        string
	ManagementAddress string
	Database          Database
	Token             token.Config
}

func LoadDatabase(lookup Lookup) (Database, error) {
	raw := strings.TrimSpace(lookup(envDatabaseURL))
	if raw == "" {
		return Database{}, fmt.Errorf("%s absente", envDatabaseURL)
	}
	requireTLS, err := boolOrDefault(lookup(envDatabaseTLS), true)
	if err != nil {
		return Database{}, fmt.Errorf("%s : %w", envDatabaseTLS, err)
	}
	if err := checkDatabaseURL(raw, requireTLS); err != nil {
		return Database{}, fmt.Errorf("%s : %w", envDatabaseURL, err)
	}
	return Database{URL: raw}, nil
}

func LoadServer(lookup Lookup) (Server, error) {
	database, err := LoadDatabase(lookup)
	if err != nil {
		return Server{}, err
	}
	tokenConfig, err := loadToken(lookup)
	if err != nil {
		return Server{}, err
	}
	return Server{
		APIAddress:        valueOrDefault(lookup(envAPIAddress), defaultAPIAddress),
		ManagementAddress: valueOrDefault(lookup(envManagementAddress), defaultManagementAddress),
		Database:          database,
		Token:             tokenConfig,
	}, nil
}

func loadToken(lookup Lookup) (token.Config, error) {
	issuer := strings.TrimSuffix(strings.TrimSpace(lookup(envIssuer)), "/")
	if issuer == "" {
		return token.Config{}, fmt.Errorf("%s absent", envIssuer)
	}
	if _, err := url.ParseRequestURI(issuer); err != nil {
		return token.Config{}, fmt.Errorf("%s n'est pas une URL absolue", envIssuer)
	}
	return token.Config{
		Issuer:          issuer,
		Audience:        valueOrDefault(lookup(envAudience), defaultAudience),
		AuthorizedParty: valueOrDefault(lookup(envAuthorizedParty), defaultAuthorizedParty),
		KeySetURL:       valueOrDefault(lookup(envKeySetURL), issuer+keycloakKeySetPath),
	}, nil
}

func checkDatabaseURL(raw string, requireTLS bool) error {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return errors.New("URL postgres:// attendue")
	}
	if requireTLS && parsed.Query().Get("sslmode") != verifiedTLSMode {
		return fmt.Errorf("sslmode=%s exigé", verifiedTLSMode)
	}
	return nil
}

func boolOrDefault(raw string, fallback bool) (bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(trimmed)
	if err != nil {
		return fallback, errors.New("booléen attendu (true ou false)")
	}
	return value, nil
}

func valueOrDefault(raw, fallback string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return fallback
}
