package jsonscan

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

type Kind int

const (
	Malformed Kind = iota + 1
	DuplicateKey
	UnexpectedKey
)

type Error struct {
	Kind   Kind
	Detail string
}

func (e *Error) Error() string {
	return e.Detail
}

type KeyNormalizer func(string) string

func ExactKeys(key string) string {
	return key
}

func CaseFoldedKeys(key string) string {
	return strings.ToLower(strings.ToUpper(key))
}

type Policy struct {
	Normalize KeyNormalizer
	Accept    func(key string) bool
}

func Check(body []byte, policy Policy) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	walker := walker{decoder: decoder, policy: policy}
	if err := walker.value(); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return malformed("contenu après la valeur JSON")
	}
	return nil
}

type walker struct {
	decoder *json.Decoder
	policy  Policy
}

func (w walker) value() error {
	token, err := w.decoder.Token()
	if err != nil {
		return malformed("syntaxe JSON invalide")
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		return w.object()
	case '[':
		return w.array()
	}
	return malformed("délimiteur inattendu")
}

func (w walker) object() error {
	seen := make(map[string]struct{})
	for w.decoder.More() {
		key, err := w.key()
		if err != nil {
			return err
		}
		normalized := w.policy.Normalize(key)
		if _, duplicated := seen[normalized]; duplicated {
			return &Error{Kind: DuplicateKey, Detail: "clé présente plusieurs fois dans un même objet"}
		}
		seen[normalized] = struct{}{}
		if err := w.value(); err != nil {
			return err
		}
	}
	return w.close()
}

func (w walker) key() (string, error) {
	token, err := w.decoder.Token()
	if err != nil {
		return "", malformed("syntaxe JSON invalide")
	}
	key, isKey := token.(string)
	if !isKey {
		return "", malformed("clé d'objet attendue")
	}
	if w.policy.Accept != nil && !w.policy.Accept(key) {
		return "", &Error{Kind: UnexpectedKey, Detail: "clé non prévue par le contrat"}
	}
	return key, nil
}

func (w walker) array() error {
	for w.decoder.More() {
		if err := w.value(); err != nil {
			return err
		}
	}
	return w.close()
}

func (w walker) close() error {
	if _, err := w.decoder.Token(); err != nil {
		return malformed("syntaxe JSON invalide")
	}
	return nil
}

func malformed(detail string) error {
	return &Error{Kind: Malformed, Detail: detail}
}
