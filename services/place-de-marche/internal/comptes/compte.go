package comptes

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrAccountNotFound = errors.New("compte introuvable")

type AccountID uuid.UUID

func ParseAccountID(raw string) (AccountID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return AccountID{}, fmt.Errorf("identifiant de compte invalide : %w", err)
	}
	return AccountID(parsed), nil
}

func (id AccountID) String() string {
	return uuid.UUID(id).String()
}

type Status struct {
	RegisteredAt time.Time
	Suspended    bool
}
