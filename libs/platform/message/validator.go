package message

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"unicode/utf8"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const DefaultMaxSize = 64 << 10

type Validator struct {
	maxSize   int
	contracts catalog
}

func NewValidator(contracts fs.FS, maxSize int) (*Validator, error) {
	if maxSize < 1 {
		return nil, fmt.Errorf("taille maximale de message invalide : %d", maxSize)
	}
	loaded, err := loadCatalog(contracts)
	if err != nil {
		return nil, err
	}
	return &Validator{maxSize: maxSize, contracts: loaded}, nil
}

func (v *Validator) Validate(body []byte) error {
	if len(body) > v.maxSize {
		return reject(ReasonTooLarge, "%d octets pour %d autorisés", len(body), v.maxSize)
	}
	if !utf8.Valid(body) {
		return reject(ReasonMalformed, "encodage UTF-8 invalide")
	}
	if err := rejectDuplicateKeys(body); err != nil {
		return err
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		return reject(ReasonMalformed, "syntaxe JSON invalide")
	}
	schema, err := v.contractOf(document)
	if err != nil {
		return err
	}
	if err := schema.Validate(document); err != nil {
		return schemaViolation(err)
	}
	return nil
}

func (v *Validator) Decode(body []byte, data any) (Envelope, error) {
	if err := v.Validate(body); err != nil {
		return Envelope{}, err
	}
	var envelope Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Envelope{}, reject(ReasonMalformed, "enveloppe non décodable")
	}
	if err := json.Unmarshal(envelope.Data, data); err != nil {
		return Envelope{}, reject(ReasonMalformed, "données non décodables dans le type attendu")
	}
	return envelope, nil
}

func (v *Validator) Encode(envelope Envelope) ([]byte, error) {
	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("encodage du message %s : %w", envelope.Type, err)
	}
	if err := v.Validate(body); err != nil {
		return nil, fmt.Errorf("message %s non conforme à son contrat : %w", envelope.Type, err)
	}
	return body, nil
}

func (v *Validator) contractOf(document any) (*jsonschema.Schema, error) {
	members, isObject := document.(map[string]any)
	if !isObject {
		return nil, reject(ReasonSchemaViolation, "le message n'est pas un objet JSON")
	}
	messageType, hasType := members["type"].(string)
	if !hasType {
		return nil, reject(ReasonUnknownContract, "type de message absent ou non textuel")
	}
	version, err := contractVersion(members["version"])
	if err != nil {
		return nil, err
	}
	schema, known := v.contracts[contractKey{messageType: messageType, version: version}]
	if !known {
		return nil, reject(ReasonUnknownContract, "aucun contrat pour ce type et cette version")
	}
	return schema, nil
}

func contractVersion(raw any) (int64, error) {
	number, isNumber := raw.(json.Number)
	if !isNumber {
		return 0, reject(ReasonUnknownContract, "version de contrat absente ou non numérique")
	}
	version, err := number.Int64()
	if err != nil {
		return 0, reject(ReasonUnknownContract, "version de contrat non entière")
	}
	return version, nil
}

func schemaViolation(err error) error {
	var violation *jsonschema.ValidationError
	if !errors.As(err, &violation) {
		return reject(ReasonSchemaViolation, "message refusé par son schéma")
	}
	leaf := firstLeaf(violation)
	return reject(ReasonSchemaViolation, "valeur /%s refusée par le mot-clé %s",
		strings.Join(leaf.InstanceLocation, "/"), keywordOf(leaf))
}

func firstLeaf(violation *jsonschema.ValidationError) *jsonschema.ValidationError {
	for len(violation.Causes) > 0 {
		violation = violation.Causes[0]
	}
	return violation
}

func keywordOf(violation *jsonschema.ValidationError) string {
	if violation.ErrorKind == nil {
		return "inconnu"
	}
	return strings.Join(violation.ErrorKind.KeywordPath(), "/")
}
