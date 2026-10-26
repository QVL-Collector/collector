package message

import (
	"errors"

	"collector/libs/platform/internal/jsonscan"
)

func rejectDuplicateKeys(body []byte) error {
	err := jsonscan.Check(body, jsonscan.Policy{Normalize: jsonscan.ExactKeys})
	var scanError *jsonscan.Error
	if !errors.As(err, &scanError) {
		return err
	}
	if scanError.Kind == jsonscan.DuplicateKey {
		return reject(ReasonDuplicateKey, "%s", scanError.Detail)
	}
	return reject(ReasonMalformed, "%s", scanError.Detail)
}
