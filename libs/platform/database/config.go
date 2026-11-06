package database

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	envDatabaseURL  = "BASE_DE_DONNEES_URL"
	envDatabaseTLS  = "BASE_DE_DONNEES_TLS_EXIGE"
	verifiedTLSMode = "verify-full"
)

type Config struct {
	URL string
}

func LoadConfig(lookup func(name string) string) (Config, error) {
	raw := strings.TrimSpace(lookup(envDatabaseURL))
	if raw == "" {
		return Config{}, fmt.Errorf("%s absente", envDatabaseURL)
	}
	requireTLS, err := boolOrDefault(lookup(envDatabaseTLS), true)
	if err != nil {
		return Config{}, fmt.Errorf("%s : %w", envDatabaseTLS, err)
	}
	if err := checkURL(raw, requireTLS); err != nil {
		return Config{}, fmt.Errorf("%s : %w", envDatabaseURL, err)
	}
	return Config{URL: raw}, nil
}

func checkURL(raw string, requireTLS bool) error {
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
