package message

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"
)

type Envelope struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	EmittedAt     time.Time       `json:"emitted_at"`
	CorrelationID string          `json:"correlation_id"`
	Data          json.RawMessage `json:"data"`
}

type Contract struct {
	Type    string
	Version int
}

func NewEnvelope(contract Contract, correlationID string, data any, emittedAt time.Time) (Envelope, error) {
	id, err := newID()
	if err != nil {
		return Envelope{}, err
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return Envelope{}, fmt.Errorf("encodage des données de %s : %w", contract.Type, err)
	}
	return Envelope{
		ID:            id,
		Type:          contract.Type,
		Version:       contract.Version,
		EmittedAt:     emittedAt.UTC(),
		CorrelationID: correlationID,
		Data:          raw,
	}, nil
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("génération d'identifiant de message : %w", err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
