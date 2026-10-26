package message

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func rejectDuplicateKeys(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := walkValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return reject(ReasonMalformed, "contenu après la valeur JSON")
	}
	return nil
}

func walkValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return reject(ReasonMalformed, "syntaxe JSON invalide")
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		return walkObject(decoder)
	case '[':
		return walkArray(decoder)
	}
	return reject(ReasonMalformed, "délimiteur inattendu")
}

func walkObject(decoder *json.Decoder) error {
	seen := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return reject(ReasonMalformed, "syntaxe JSON invalide")
		}
		key, isKey := token.(string)
		if !isKey {
			return reject(ReasonMalformed, "clé d'objet attendue")
		}
		if _, duplicated := seen[key]; duplicated {
			return reject(ReasonDuplicateKey, "clé présente plusieurs fois dans un même objet")
		}
		seen[key] = struct{}{}
		if err := walkValue(decoder); err != nil {
			return err
		}
	}
	return closeContainer(decoder)
}

func walkArray(decoder *json.Decoder) error {
	for decoder.More() {
		if err := walkValue(decoder); err != nil {
			return err
		}
	}
	return closeContainer(decoder)
}

func closeContainer(decoder *json.Decoder) error {
	if _, err := decoder.Token(); err != nil {
		return reject(ReasonMalformed, "syntaxe JSON invalide")
	}
	return nil
}
