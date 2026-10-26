package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Env struct {
	errs []error
}

func (e *Env) Required(name string) string {
	value := os.Getenv(name)
	if value == "" {
		e.errs = append(e.errs, fmt.Errorf("variable d'environnement %s manquante", name))
	}
	return value
}

func (e *Env) String(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func (e *Env) Int(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s : entier attendu : %w", name, err))
	}
	return value
}

func (e *Env) Duration(name string, fallback time.Duration) time.Duration {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s : durée attendue (ex. 30s) : %w", name, err))
	}
	return value
}

func (e *Env) Time(name string, fallback time.Time) time.Time {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s : date RFC 3339 attendue : %w", name, err))
	}
	return value
}

func (e *Env) Require(ok bool, reason string) {
	if !ok {
		e.errs = append(e.errs, errors.New(reason))
	}
}

func (e *Env) Err() error {
	return errors.Join(e.errs...)
}
