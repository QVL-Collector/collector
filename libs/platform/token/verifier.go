package token

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	acceptedAlgorithm     = jwa.RS256
	expectedTypClaim      = "Bearer"
	clockSkew             = 30 * time.Second
	keySetRefreshInterval = 15 * time.Minute
	keySetFetchTimeout    = 5 * time.Second
	realmAccessClaim      = "realm_access"
	rolesMember           = "roles"
)

var ErrInvalidToken = errors.New("jeton refusé")

type Config struct {
	Issuer          string
	Audience        string
	AuthorizedParty string
	KeySetURL       string
}

func (c Config) validate() error {
	var missing []error
	if c.Issuer == "" {
		missing = append(missing, errors.New("émetteur absent"))
	}
	if c.Audience == "" {
		missing = append(missing, errors.New("audience absente"))
	}
	if c.AuthorizedParty == "" {
		missing = append(missing, errors.New("client autorisé absent"))
	}
	if _, err := url.ParseRequestURI(c.KeySetURL); err != nil {
		missing = append(missing, errors.New("adresse du jeu de clés invalide"))
	}
	return errors.Join(missing...)
}

type Claims struct {
	Subject string
	Roles   []string
}

func (c Claims) HasRole(role string) bool {
	return slices.Contains(c.Roles, role)
}

type Verifier struct {
	options []jwt.ParseOption
}

func NewVerifier(ctx context.Context, cfg Config) (*Verifier, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration de la vérification des jetons : %w", err)
	}
	cache := jwk.NewCache(ctx)
	client := &http.Client{Timeout: keySetFetchTimeout}
	if err := cache.Register(cfg.KeySetURL, jwk.WithHTTPClient(client), jwk.WithMinRefreshInterval(keySetRefreshInterval)); err != nil {
		return nil, fmt.Errorf("enregistrement du jeu de clés : %w", err)
	}
	keys := jwk.NewCachedSet(cache, cfg.KeySetURL)
	return &Verifier{options: parseOptions(cfg, keys)}, nil
}

func parseOptions(cfg Config, keys jwk.Set) []jwt.ParseOption {
	return []jwt.ParseOption{
		jwt.WithKeySet(keys),
		jwt.WithValidate(true),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithAudience(cfg.Audience),
		jwt.WithAcceptableSkew(clockSkew),
		jwt.WithRequiredClaim(jwt.SubjectKey),
		jwt.WithRequiredClaim(jwt.ExpirationKey),
		jwt.WithClaimValue("typ", expectedTypClaim),
		jwt.WithClaimValue("azp", cfg.AuthorizedParty),
	}
}

func (v *Verifier) Verify(ctx context.Context, raw string) (Claims, error) {
	if err := requireSignatureAlgorithm(raw); err != nil {
		return Claims{}, err
	}
	parsed, err := jwt.ParseString(raw, append(slices.Clone(v.options), jwt.WithContext(ctx))...)
	if err != nil {
		return Claims{}, fmt.Errorf("%w : %w", ErrInvalidToken, err)
	}
	return Claims{Subject: parsed.Subject(), Roles: realmRoles(parsed)}, nil
}

func requireSignatureAlgorithm(raw string) error {
	message, err := jws.ParseString(raw)
	if err != nil {
		return fmt.Errorf("%w : structure illisible", ErrInvalidToken)
	}
	signatures := message.Signatures()
	if len(signatures) != 1 || signatures[0].ProtectedHeaders().Algorithm() != acceptedAlgorithm {
		return fmt.Errorf("%w : algorithme de signature refusé", ErrInvalidToken)
	}
	return nil
}

func realmRoles(parsed jwt.Token) []string {
	claim, found := parsed.Get(realmAccessClaim)
	if !found {
		return nil
	}
	access, isObject := claim.(map[string]any)
	if !isObject {
		return nil
	}
	values, isList := access[rolesMember].([]any)
	if !isList {
		return nil
	}
	roles := make([]string, 0, len(values))
	for _, value := range values {
		if role, isText := value.(string); isText {
			roles = append(roles, role)
		}
	}
	return roles
}
