package broker

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const (
	envHost      = "CANAL_HOTE"
	envPort      = "CANAL_PORT"
	envVhost     = "CANAL_VHOST"
	envUser      = "CANAL_UTILISATEUR"
	envUserKey   = "CANAL_MOT_DE_PASSE"
	defaultPort  = "5672"
	defaultVhost = "collector"
	amqpScheme   = "amqp"
	maxPort      = 65535
)

type Endpoint struct {
	url            string
	User           string
	ConnectionName string
}

func LoadEndpoint(lookup func(name string) string) (Endpoint, error) {
	host, err := required(lookup, envHost)
	if err != nil {
		return Endpoint{}, err
	}
	user, err := required(lookup, envUser)
	if err != nil {
		return Endpoint{}, err
	}
	password := lookup(envUserKey)
	if password == "" {
		return Endpoint{}, fmt.Errorf("%s absent", envUserKey)
	}
	port := valueOrDefault(lookup(envPort), defaultPort)
	if number, err := strconv.Atoi(port); err != nil || number < 1 || number > maxPort {
		return Endpoint{}, fmt.Errorf("%s : port entre 1 et %d attendu", envPort, maxPort)
	}
	target := url.URL{
		Scheme: amqpScheme,
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + valueOrDefault(lookup(envVhost), defaultVhost),
	}
	return Endpoint{url: target.String(), User: user}, nil
}

func (e Endpoint) Named(connectionName string) Endpoint {
	e.ConnectionName = connectionName
	return e
}

func required(lookup func(name string) string, name string) (string, error) {
	value := strings.TrimSpace(lookup(name))
	if value == "" {
		return "", fmt.Errorf("%s absent", name)
	}
	return value, nil
}

func valueOrDefault(raw, fallback string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return fallback
}
