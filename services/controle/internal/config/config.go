package config

import (
	"strings"

	"collector/libs/platform/broker"
)

const (
	envManagementAddress     = "ADRESSE_GESTION"
	envRequestPublisher      = "CANAL_EMETTEUR_ATTENDU"
	defaultManagementAddress = ":9090"
	defaultRequestPublisher  = "marche"
)

type Lookup func(name string) string

type Server struct {
	ManagementAddress string
	Endpoint          broker.Endpoint
	RequestPublisher  string
}

func LoadServer(lookup Lookup) (Server, error) {
	endpoint, err := broker.LoadEndpoint(lookup)
	if err != nil {
		return Server{}, err
	}
	return Server{
		ManagementAddress: valueOrDefault(lookup(envManagementAddress), defaultManagementAddress),
		Endpoint:          endpoint,
		RequestPublisher:  valueOrDefault(lookup(envRequestPublisher), defaultRequestPublisher),
	}, nil
}

func valueOrDefault(raw, fallback string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return fallback
}
