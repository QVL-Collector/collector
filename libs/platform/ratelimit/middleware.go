package ratelimit

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"strconv"

	"collector/libs/platform/httpserver"
	"collector/libs/platform/problem"
	"collector/libs/platform/token"
)

const (
	retryAfterHeader = "Retry-After"
	scopeAddress     = "adresse"
	scopeAccount     = "compte"
	unknownAddress   = "inconnue"
	ipv6PrefixBits   = 64
)

type Guard struct {
	nominal     Limit
	loadTest    Limit
	loadTestKey string
	store       *store
	logger      *slog.Logger
}

func NewGuard(policy Policy, logger *slog.Logger, clock Clock) *Guard {
	return &Guard{
		nominal:     policy.Nominal,
		loadTest:    policy.LoadTest,
		loadTestKey: loadTestKeyOf(policy.LoadTestAddress),
		store:       newStore(clock, maxTrackedKeys),
		logger:      logger,
	}
}

func (g *Guard) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client := clientOf(r)
		refused, admitted := g.store.admit(g.demandsOf(r.Context(), client.key))
		if !admitted {
			g.reject(w, r, client, refused)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (g *Guard) demandsOf(ctx context.Context, clientKey string) []demand {
	limit := g.limitFor(clientKey)
	demands := []demand{{scope: scopeAddress, key: scopeAddress + ":" + clientKey, limit: limit}}
	if claims, found := token.ClaimsFrom(ctx); found && claims.Subject != "" {
		demands = append(demands, demand{scope: scopeAccount, key: scopeAccount + ":" + claims.Subject, limit: limit})
	}
	return demands
}

func (g *Guard) limitFor(clientKey string) Limit {
	if g.loadTestKey != "" && clientKey == g.loadTestKey {
		return g.loadTest
	}
	return g.nominal
}

func (g *Guard) reject(w http.ResponseWriter, r *http.Request, client clientIdentity, refused refusal) {
	g.logger.InfoContext(r.Context(), "débit dépassé",
		slog.String("event", "http.debit_depasse"),
		slog.String("adresse_client", client.address),
		slog.String("cle", client.key),
		slog.String("limite", refused.scope),
		slog.Int("reessayer_dans", refused.retryAfterSeconds),
	)
	w.Header().Set(retryAfterHeader, strconv.Itoa(refused.retryAfterSeconds))
	problem.Write(w, problem.TooManyRequests())
}

type clientIdentity struct {
	address string
	key     string
}

func clientOf(r *http.Request) clientIdentity {
	address, found := httpserver.ClientAddress(r)
	if !found {
		return clientIdentity{address: unknownAddress, key: unknownAddress}
	}
	return clientIdentity{address: address.String(), key: addressKey(address)}
}

func loadTestKeyOf(address netip.Addr) string {
	if !address.IsValid() {
		return ""
	}
	return addressKey(address)
}

func addressKey(address netip.Addr) string {
	if address.Is6() {
		return netip.PrefixFrom(address, ipv6PrefixBits).Masked().String()
	}
	return address.String()
}
