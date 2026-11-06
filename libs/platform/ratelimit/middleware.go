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
		address := clientKeyOf(r)
		refused, admitted := g.store.admit(g.demandsOf(r.Context(), address))
		if !admitted {
			g.reject(w, r, address, refused)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (g *Guard) demandsOf(ctx context.Context, address string) []demand {
	limit := g.limitFor(address)
	demands := []demand{{scope: scopeAddress, key: scopeAddress + ":" + address, limit: limit}}
	if claims, found := token.ClaimsFrom(ctx); found && claims.Subject != "" {
		demands = append(demands, demand{scope: scopeAccount, key: scopeAccount + ":" + claims.Subject, limit: limit})
	}
	return demands
}

func (g *Guard) limitFor(address string) Limit {
	if g.loadTestKey != "" && address == g.loadTestKey {
		return g.loadTest
	}
	return g.nominal
}

func (g *Guard) reject(w http.ResponseWriter, r *http.Request, address string, refused refusal) {
	g.logger.InfoContext(r.Context(), "débit dépassé",
		slog.String("event", "http.debit_depasse"),
		slog.String("adresse_client", address),
		slog.String("limite", refused.scope),
		slog.Int("reessayer_dans", refused.retryAfterSeconds),
	)
	w.Header().Set(retryAfterHeader, strconv.Itoa(refused.retryAfterSeconds))
	problem.Write(w, problem.TooManyRequests())
}

func clientKeyOf(r *http.Request) string {
	address, found := httpserver.ClientAddress(r)
	if !found {
		return unknownAddress
	}
	return addressKey(address)
}

func loadTestKeyOf(address netip.Addr) string {
	if !address.IsValid() {
		return ""
	}
	return addressKey(address)
}

func addressKey(address netip.Addr) string {
	normalized := address.Unmap().WithZone("")
	if normalized.Is6() {
		return netip.PrefixFrom(normalized, ipv6PrefixBits).Masked().String()
	}
	return normalized.String()
}
