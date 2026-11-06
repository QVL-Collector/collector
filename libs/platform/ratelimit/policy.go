package ratelimit

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

const (
	envPerMinute             = "LIMITE_DEBIT_PAR_MINUTE"
	envBurst                 = "LIMITE_DEBIT_RAFALE"
	envLoadTestAddress       = "LIMITE_DEBIT_TIR_ADRESSE"
	envLoadTestPerMinute     = "LIMITE_DEBIT_TIR_PAR_MINUTE"
	envLoadTestBurst         = "LIMITE_DEBIT_TIR_RAFALE"
	defaultPerMinute         = 30
	defaultBurst             = 10
	defaultLoadTestPerMinute = 6000
	defaultLoadTestBurst     = 200
	secondsPerMinute         = 60
)

type Limit struct {
	Rate  float64
	Burst int
}

func PerMinute(requests, burst int) Limit {
	return Limit{Rate: float64(requests) / secondsPerMinute, Burst: burst}
}

func (l Limit) capacity() float64 {
	return float64(l.Burst)
}

type Policy struct {
	Nominal         Limit
	LoadTest        Limit
	LoadTestAddress netip.Addr
}

type setting struct {
	name     string
	fallback int
}

func LoadPolicy(lookup func(name string) string) (Policy, error) {
	nominal, err := loadLimit(lookup, setting{envPerMinute, defaultPerMinute}, setting{envBurst, defaultBurst})
	if err != nil {
		return Policy{}, err
	}
	loadTest, err := loadLimit(lookup,
		setting{envLoadTestPerMinute, defaultLoadTestPerMinute},
		setting{envLoadTestBurst, defaultLoadTestBurst},
	)
	if err != nil {
		return Policy{}, err
	}
	address, err := optionalAddress(lookup(envLoadTestAddress))
	if err != nil {
		return Policy{}, fmt.Errorf("%s : %w", envLoadTestAddress, err)
	}
	return Policy{Nominal: nominal, LoadTest: loadTest, LoadTestAddress: address}, nil
}

func loadLimit(lookup func(name string) string, rate, burst setting) (Limit, error) {
	requests, err := positiveInteger(lookup, rate)
	if err != nil {
		return Limit{}, err
	}
	capacity, err := positiveInteger(lookup, burst)
	if err != nil {
		return Limit{}, err
	}
	return PerMinute(requests, capacity), nil
}

func positiveInteger(lookup func(name string) string, s setting) (int, error) {
	trimmed := strings.TrimSpace(lookup(s.name))
	if trimmed == "" {
		return s.fallback, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("%s : entier strictement positif attendu", s.name)
	}
	return value, nil
}

func optionalAddress(raw string) (netip.Addr, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return netip.Addr{}, nil
	}
	address, err := netip.ParseAddr(trimmed)
	if err != nil {
		return netip.Addr{}, errors.New("adresse IP attendue")
	}
	return address, nil
}
