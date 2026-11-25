package config

import (
	"fmt"
	"strings"
	"time"

	"collector/libs/platform/broker"
	"collector/libs/platform/database"
	"collector/libs/platform/ratelimit"
	"collector/libs/platform/token"
	"collector/services/place-de-marche/internal/reservations"
)

const (
	envAPIAddress              = "ADRESSE_API"
	envInternalAddress         = "ADRESSE_INTERNE"
	envManagementAddress       = "ADRESSE_GESTION"
	envMetricsAddress          = "ADRESSE_METRIQUES"
	envResultPublisher         = "CANAL_EMETTEUR_ATTENDU"
	envReservationDuration     = "RESERVATION_DUREE"
	envReservationCeiling      = "RESERVATION_DUREE_MAX"
	envReservationCaller       = "RESERVATION_CLIENT_AUTORISE"
	defaultResultPublisher     = "controle"
	defaultAPIAddress          = ":8080"
	defaultInternalAddress     = ":8081"
	defaultManagementAddress   = ":9090"
	defaultMetricsAddress      = ":9091"
	defaultAudience            = "place-de-marche"
	defaultReservationDuration = 10 * time.Minute
	defaultReservationCeiling  = 30 * time.Minute
	defaultReservationCaller   = "transactions"
	minimumReservationDuration = time.Second
)

type Lookup func(name string) string

type Channel struct {
	Endpoint        broker.Endpoint
	ResultPublisher string
}

type Reservation struct {
	Term   reservations.Term
	Caller string
}

type Server struct {
	APIAddress        string
	InternalAddress   string
	ManagementAddress string
	MetricsAddress    string
	Database          database.Config
	Token             token.Config
	Channel           Channel
	RateLimit         ratelimit.Policy
	Reservation       Reservation
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
	reservation, err := loadReservation(lookup)
	if err != nil {
		return Server{}, err
	}
	return Server{
		APIAddress:        valueOrDefault(lookup(envAPIAddress), defaultAPIAddress),
		InternalAddress:   valueOrDefault(lookup(envInternalAddress), defaultInternalAddress),
		ManagementAddress: valueOrDefault(lookup(envManagementAddress), defaultManagementAddress),
		MetricsAddress:    valueOrDefault(lookup(envMetricsAddress), defaultMetricsAddress),
		Database:          databaseConfig,
		Token:             tokenConfig,
		Channel: Channel{
			Endpoint:        endpoint,
			ResultPublisher: valueOrDefault(lookup(envResultPublisher), defaultResultPublisher),
		},
		RateLimit:   rateLimit,
		Reservation: reservation,
	}, nil
}

func loadReservation(lookup Lookup) (Reservation, error) {
	requested, err := reservationDuration(lookup(envReservationDuration), envReservationDuration, defaultReservationDuration)
	if err != nil {
		return Reservation{}, err
	}
	ceiling, err := reservationDuration(lookup(envReservationCeiling), envReservationCeiling, defaultReservationCeiling)
	if err != nil {
		return Reservation{}, err
	}
	return Reservation{
		Term:   reservations.Term{Requested: requested, Ceiling: ceiling},
		Caller: valueOrDefault(lookup(envReservationCaller), defaultReservationCaller),
	}, nil
}

func reservationDuration(raw, name string, fallback time.Duration) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(trimmed)
	if err != nil || value < minimumReservationDuration {
		return 0, fmt.Errorf("%s : durée d'au moins %s attendue (ex. 10m, 5s)", name, minimumReservationDuration)
	}
	return value, nil
}

func valueOrDefault(raw, fallback string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return fallback
}
