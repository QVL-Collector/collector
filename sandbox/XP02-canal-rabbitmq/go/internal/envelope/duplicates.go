package envelope

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func rejectDuplicateKeys(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := walkValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w : contenu après la valeur JSON", ErrInvalid)
	}
	return nil
}

func walkValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("%w : %w", ErrInvalid, err)
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
	return fmt.Errorf("%w : délimiteur inattendu %q", ErrInvalid, delimiter)
}

func walkObject(decoder *json.Decoder) error {
	seen := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("%w : %w", ErrInvalid, err)
		}
		key, isKey := token.(string)
		if !isKey {
			return fmt.Errorf("%w : clé d'objet attendue", ErrInvalid)
		}
		if _, duplicated := seen[key]; duplicated {
			return fmt.Errorf("%w : %q", ErrDuplicateKey, key)
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
		return fmt.Errorf("%w : %w", ErrInvalid, err)
	}
	return nil
}
