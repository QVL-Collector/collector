package config

import (
	"strings"

	"collector/libs/platform/broker"
	"collector/libs/platform/database"
	"collector/libs/platform/token"
)

const (
	envAPIAddress            = "ADRESSE_API"
	envManagementAddress     = "ADRESSE_GESTION"
	envDecisionPublisher     = "CANAL_EMETTEUR_ATTENDU"
	defaultDecisionPublisher = "marche"
	defaultAPIAddress        = ":8080"
	defaultManagementAddress = ":9090"
	defaultAudience          = "interactions"
)

type Lookup func(name string) string

type Channel struct {
	Endpoint          broker.Endpoint
	DecisionPublisher string
}

type Server struct {
	APIAddress        string
	ManagementAddress string
	Database          database.Config
	Token             token.Config
	Channel           Channel
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
	return Server{
		APIAddress:        valueOrDefault(lookup(envAPIAddress), defaultAPIAddress),
		ManagementAddress: valueOrDefault(lookup(envManagementAddress), defaultManagementAddress),
		Database:          databaseConfig,
		Token:             tokenConfig,
		Channel: Channel{
			Endpoint:          endpoint,
			DecisionPublisher: valueOrDefault(lookup(envDecisionPublisher), defaultDecisionPublisher),
		},
	}, nil
}

func valueOrDefault(raw, fallback string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return fallback
}
