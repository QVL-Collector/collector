package serviceclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type variable string

const (
	grantEndpointVariable    variable = "OIDC_JETON_URL"
	clientIdentifierVariable variable = "OIDC_CLIENT_ID"
	clientKeyVariable        variable = "OIDC_CLIENT_SECRET"
	keycloakGrantPath                 = "/protocol/openid-connect/token"
	grantFetchTimeout                 = 5 * time.Second
)

var errRedirectRefused = errors.New("redirection refusée : le jeton de service ne suit pas")

type Credentials struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

func LoadCredentials(lookup func(name string) string, issuer, defaultClientID string, scopes ...string) (Credentials, error) {
	read := func(name variable) string {
		return strings.TrimSpace(lookup(string(name)))
	}
	grantEndpoint := valueOrDefault(read(grantEndpointVariable), strings.TrimSuffix(issuer, "/")+keycloakGrantPath)
	if !absoluteHTTPURL(grantEndpoint) {
		return Credentials{}, fmt.Errorf("%s n'est pas une URL absolue http ou https", grantEndpointVariable)
	}
	clientKey := read(clientKeyVariable)
	if clientKey == "" {
		return Credentials{}, fmt.Errorf("%s absent", clientKeyVariable)
	}
	return Credentials{
		TokenURL:     grantEndpoint,
		ClientID:     valueOrDefault(read(clientIdentifierVariable), defaultClientID),
		ClientSecret: clientKey,
		Scopes:       scopes,
	}, nil
}

func absoluteHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func New(ctx context.Context, credentials Credentials, timeout time.Duration) *http.Client {
	fetchContext := context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: grantFetchTimeout})
	config := clientcredentials.Config{
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientSecret,
		TokenURL:     credentials.TokenURL,
		Scopes:       credentials.Scopes,
		AuthStyle:    oauth2.AuthStyleInHeader,
	}
	return &http.Client{
		Timeout:       timeout,
		Transport:     &oauth2.Transport{Source: config.TokenSource(fetchContext), Base: http.DefaultTransport},
		CheckRedirect: refuseRedirects,
	}
}

func refuseRedirects(*http.Request, []*http.Request) error {
	return errRedirectRefused
}

func valueOrDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
