package envelope

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"collector/sandbox/xp02/internal/ident"
)

var (
	ErrInvalid      = errors.New("message non conforme")
	ErrTooLarge     = fmt.Errorf("%w : taille maximale dépassée", ErrInvalid)
	ErrDuplicateKey = fmt.Errorf("%w : clé JSON dupliquée", ErrInvalid)
)

var members = []string{"id", "type", "version", "emitted_at", "correlation_id", "data"}

type Envelope struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	EmittedAt     time.Time       `json:"emitted_at"`
	CorrelationID string          `json:"correlation_id"`
	Data          json.RawMessage `json:"data"`
}

func New(eventType string, version int, correlationID string, data any, emittedAt time.Time) (Envelope, error) {
	id, err := ident.New()
	if err != nil {
		return Envelope{}, err
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return Envelope{}, fmt.Errorf("encodage des données de %s : %w", eventType, err)
	}
	env := Envelope{
		ID:            id,
		Type:          eventType,
		Version:       version,
		EmittedAt:     emittedAt.UTC(),
		CorrelationID: correlationID,
		Data:          raw,
	}
	if err := env.validate(); err != nil {
		return Envelope{}, err
	}
	return env, nil
}

func (e Envelope) Encode() ([]byte, error) {
	body, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("encodage de l'enveloppe %s : %w", e.ID, err)
	}
	return body, nil
}

func Decode(body []byte, maxSize int) (Envelope, error) {
	if len(body) > maxSize {
		return Envelope{}, fmt.Errorf("%w : %d octets pour %d autorisés", ErrTooLarge, len(body), maxSize)
	}
	if err := rejectDuplicateKeys(body); err != nil {
		return Envelope{}, err
	}
	if err := requireExactMembers(body); err != nil {
		return Envelope{}, err
	}
	var env Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return Envelope{}, fmt.Errorf("%w : %w", ErrInvalid, err)
	}
	if err := env.validate(); err != nil {
		return Envelope{}, err
	}
	return env, nil
}

func requireExactMembers(body []byte) error {
	var present map[string]json.RawMessage
	if err := json.Unmarshal(body, &present); err != nil {
		return fmt.Errorf("%w : %w", ErrInvalid, err)
	}
	for name := range present {
		if !slices.Contains(members, name) {
			return fmt.Errorf("%w : membre inconnu %q", ErrInvalid, name)
		}
	}
	for _, name := range members {
		if _, ok := present[name]; !ok {
			return fmt.Errorf("%w : membre manquant %q", ErrInvalid, name)
		}
	}
	return nil
}

func (e Envelope) validate() error {
	switch {
	case !ident.Valid(e.ID):
		return fmt.Errorf("%w : identifiant d'événement invalide", ErrInvalid)
	case e.Type == "":
		return fmt.Errorf("%w : type d'événement vide", ErrInvalid)
	case e.Version < 1:
		return fmt.Errorf("%w : version de contrat %d invalide", ErrInvalid, e.Version)
	case e.EmittedAt.IsZero():
		return fmt.Errorf("%w : date d'émission absente", ErrInvalid)
	case !ident.Valid(e.CorrelationID):
		return fmt.Errorf("%w : identifiant de corrélation invalide", ErrInvalid)
	case !isObject(e.Data):
		return fmt.Errorf("%w : données absentes ou non objet", ErrInvalid)
	}
	return nil
}

func isObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '{'
}
