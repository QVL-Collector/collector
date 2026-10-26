package httpjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	"collector/libs/platform/internal/jsonscan"
	"collector/libs/platform/problem"
)

const (
	mediaType          = "application/json"
	invalidBodyType    = "urn:collector:probleme:corps-invalide"
	invalidBodyTitle   = "Corps de requête invalide"
	unknownFieldPrefix = "json: unknown field "
)

type Reason string

const (
	ReasonUnsupportedMediaType Reason = "type_de_contenu_non_pris_en_charge"
	ReasonTooLarge             Reason = "corps_trop_volumineux"
	ReasonMalformed            Reason = "json_invalide"
	ReasonDuplicateKey         Reason = "cle_dupliquee"
	ReasonNotAnObject          Reason = "objet_attendu"
	ReasonUnknownField         Reason = "champ_inconnu"
	ReasonInvalidType          Reason = "type_invalide"
)

type RejectionError struct {
	Reason Reason
}

func (e *RejectionError) Error() string {
	return fmt.Sprintf("corps de requête refusé (%s)", e.Reason)
}

func (e *RejectionError) Problem() problem.Problem {
	switch e.Reason {
	case ReasonUnsupportedMediaType:
		return problem.New(http.StatusUnsupportedMediaType, "Le corps doit être de type application/json.")
	case ReasonTooLarge:
		return problem.New(http.StatusRequestEntityTooLarge, "Le corps dépasse la taille autorisée.")
	case ReasonMalformed, ReasonDuplicateKey, ReasonNotAnObject, ReasonUnknownField, ReasonInvalidType:
		return problem.Typed(invalidBodyType, invalidBodyTitle, http.StatusBadRequest, rejectionDetails[e.Reason])
	}
	return problem.Typed(invalidBodyType, invalidBodyTitle, http.StatusBadRequest, "")
}

var rejectionDetails = map[Reason]string{
	ReasonMalformed:    "Le corps n'est pas un JSON valide.",
	ReasonDuplicateKey: "Le corps contient une clé présente plusieurs fois.",
	ReasonNotAnObject:  "Le corps doit être un objet JSON.",
	ReasonUnknownField: "Le corps contient un champ inconnu.",
	ReasonInvalidType:  "Un champ du corps n'a pas le type attendu.",
}

func Decode(w http.ResponseWriter, r *http.Request, limit int64, target any) error {
	if !acceptsMediaType(r.Header.Get("Content-Type")) {
		return reject(ReasonUnsupportedMediaType)
	}
	body, err := readLimited(w, r, limit)
	if err != nil {
		return err
	}
	if err := checkStructure(body, keyPolicy(target)); err != nil {
		return err
	}
	return decodeStrict(body, target)
}

func keyPolicy(target any) jsonscan.Policy {
	policy := jsonscan.Policy{Normalize: jsonscan.CaseFoldedKeys}
	if names, exact := exactFieldNames(target); exact {
		policy.Accept = names.contains
	}
	return policy
}

func acceptsMediaType(contentType string) bool {
	parsed, _, err := mime.ParseMediaType(contentType)
	return err == nil && parsed == mediaType
}

func readLimited(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, error) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err == nil {
		return body, nil
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return nil, reject(ReasonTooLarge)
	}
	return nil, reject(ReasonMalformed)
}

func checkStructure(body []byte, policy jsonscan.Policy) error {
	if !utf8.Valid(body) {
		return reject(ReasonMalformed)
	}
	if !bytes.HasPrefix(bytes.TrimLeft(body, " \t\r\n"), []byte("{")) {
		return reject(ReasonNotAnObject)
	}
	err := jsonscan.Check(body, policy)
	var scanError *jsonscan.Error
	if !errors.As(err, &scanError) {
		return err
	}
	return reject(scanReasons[scanError.Kind])
}

var scanReasons = map[jsonscan.Kind]Reason{
	jsonscan.Malformed:     ReasonMalformed,
	jsonscan.DuplicateKey:  ReasonDuplicateKey,
	jsonscan.UnexpectedKey: ReasonUnknownField,
}

func decodeStrict(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(target)
	if err == nil {
		return nil
	}
	var invalidTarget *json.InvalidUnmarshalError
	if errors.As(err, &invalidTarget) {
		return fmt.Errorf("cible de décodage invalide : %w", err)
	}
	var wrongType *json.UnmarshalTypeError
	if errors.As(err, &wrongType) {
		return reject(ReasonInvalidType)
	}
	if strings.HasPrefix(err.Error(), unknownFieldPrefix) {
		return reject(ReasonUnknownField)
	}
	return reject(ReasonMalformed)
}

func reject(reason Reason) error {
	return &RejectionError{Reason: reason}
}
