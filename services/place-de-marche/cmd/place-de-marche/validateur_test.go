package main

import (
	"testing"

	"collector/contracts"
	"collector/libs/platform/message"
)

func validateurDeContrats(t testing.TB) *message.Validator {
	t.Helper()
	validateur, err := message.NewValidator(contracts.Messages, message.DefaultMaxSize)
	if err != nil {
		t.Fatalf("validateur de contrats : %v", err)
	}
	return validateur
}
