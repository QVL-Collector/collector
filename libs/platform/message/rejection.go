package message

import (
	"errors"
	"fmt"
)

type Reason string

const (
	ReasonTooLarge        Reason = "taille_depassee"
	ReasonMalformed       Reason = "json_invalide"
	ReasonDuplicateKey    Reason = "cle_dupliquee"
	ReasonUnknownContract Reason = "contrat_inconnu"
	ReasonSchemaViolation Reason = "schema_non_respecte"
)

type RejectionError struct {
	Reason Reason
	Detail string
}

func (e *RejectionError) Error() string {
	return fmt.Sprintf("message refusé (%s) : %s", e.Reason, e.Detail)
}

func ReasonOf(err error) (Reason, bool) {
	var rejection *RejectionError
	if errors.As(err, &rejection) {
		return rejection.Reason, true
	}
	return "", false
}

func reject(reason Reason, format string, args ...any) error {
	return &RejectionError{Reason: reason, Detail: fmt.Sprintf(format, args...)}
}
