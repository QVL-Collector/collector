package config

import (
	"strings"

	"collector/libs/platform/broker"
	"collector/libs/platform/database"
	"collector/libs/platform/ratelimit"
	"collector/libs/platform/token"
)

const (
	envAPIAddress            = "ADRESSE_API"
	envManagementAddress     = "ADRESSE_GESTION"
	envResultPublisher       = "CANAL_EMETTEUR_ATTENDU"
	defaultResultPublisher   = "controle"
	defaultAPIAddress        = ":8080"
	defaultManagementAddress = ":9090"
	defaultAudience          = "place-de-marche"
)

type Lookup func(name string) string

type Channel struct {
	Endpoint        broker.Endpoint
	ResultPublisher string
}

type Server struct {
	APIAddress        string
	ManagementAddress string
	Database          database.Config
	Token             token.Config
	Channel           Channel
	RateLimit         ratelimit.Policy
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
	endpoint, err := broker.LoadEndpoint(lookup)
	if err != nil {
		return Server{}, err
	}
	rateLimit, err := ratelimit.LoadPolicy(lookup)
	if err != nil {
		return Server{}, err
	}
	return Server{
		APIAddress:        valueOrDefault(lookup(envAPIAddress), defaultAPIAddress),
		ManagementAddress: valueOrDefault(lookup(envManagementAddress), defaultManagementAddress),
		Database:          databaseConfig,
		Token:             tokenConfig,
		Channel: Channel{
			Endpoint:        endpoint,
			ResultPublisher: valueOrDefault(lookup(envResultPublisher), defaultResultPublisher),
		},
		RateLimit: rateLimit,
	}, nil
}

func valueOrDefault(raw, fallback string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return fallback
}
