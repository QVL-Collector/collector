package config

import (
	"fmt"
	"net/url"
	"strings"

	"collector/libs/platform/database"
	"collector/libs/platform/ratelimit"
	"collector/libs/platform/serviceclient"
	"collector/libs/platform/token"
)

const (
	envAPIAddress            = "ADRESSE_API"
	envManagementAddress     = "ADRESSE_GESTION"
	envMetricsAddress        = "ADRESSE_METRIQUES"
	envMarketplaceURL        = "PLACE_DE_MARCHE_URL_INTERNE"
	defaultAPIAddress        = ":8080"
	defaultManagementAddress = ":9090"
	defaultMetricsAddress    = ":9091"
	defaultAudience          = "transactions"
	defaultServiceClient     = "transactions"
	reservationScope         = "reserver"
)

type Lookup func(name string) string

type Server struct {
	APIAddress         string
	ManagementAddress  string
	MetricsAddress     string
	Database           database.Config
	Token              token.Config
	ServiceCredentials serviceclient.Credentials
	MarketplaceURL     string
	RateLimit          ratelimit.Policy
}

func LoadServer(lookup Lookup) (Server, error) {
	databaseConfig, err := database.LoadConfig(lookup)
	if err != nil {
		return Server{}, err
	}
	tokenConfig, err := token.LoadConfig(lookup, defaultAudience)
	if err != nil {
		return Server{}, err
	}
	credentials, err := serviceclient.LoadCredentials(lookup, tokenConfig.Issuer, defaultServiceClient, reservationScope)
	if err != nil {
		return Server{}, err
	}
	marketplaceURL, err := absoluteURL(lookup(envMarketplaceURL), envMarketplaceURL)
	if err != nil {
		return Server{}, err
	}
	rateLimit, err := ratelimit.LoadPolicy(lookup)
	if err != nil {
		return Server{}, err
	}
	return Server{
		APIAddress:         valueOrDefault(lookup(envAPIAddress), defaultAPIAddress),
		ManagementAddress:  valueOrDefault(lookup(envManagementAddress), defaultManagementAddress),
		MetricsAddress:     valueOrDefault(lookup(envMetricsAddress), defaultMetricsAddress),
		Database:           databaseConfig,
		Token:              tokenConfig,
		ServiceCredentials: credentials,
		MarketplaceURL:     marketplaceURL,
		RateLimit:          rateLimit,
	}, nil
}

func absoluteURL(raw, name string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s absente", name)
	}
	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("%s n'est pas une URL absolue", name)
	}
	return trimmed, nil
}

func valueOrDefault(raw, fallback string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return fallback
}
