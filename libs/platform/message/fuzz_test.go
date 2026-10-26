package message_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"collector/libs/platform/message"
)

type acceptedMessage struct {
	Type    string `json:"type"`
	Version int    `json:"version"`
	Data    struct {
		Issue  *string        `json:"issue"`
		Reason *string        `json:"reason"`
		Seller map[string]any `json:"seller"`
	} `json:"data"`
}

func FuzzValidate(f *testing.F) {
	for _, ex := range loadExamples(f) {
		f.Add(ex.body)
	}
	f.Add([]byte(""))
	f.Add([]byte("{}"))
	f.Add([]byte(`{"a":1,"a":2}`))
	validator := newValidator(f, message.DefaultMaxSize)
	f.Fuzz(func(t *testing.T, body []byte) {
		err := validator.Validate(body)
		if err != nil {
			reason, ok := message.ReasonOf(err)
			if !ok || !knownReasons[reason] {
				t.Fatalf("refus sans raison de la liste: %v", err)
			}
			return
		}
		expectAcceptedMessageIsInContract(t, body)
	})
}

func expectAcceptedMessageIsInContract(t *testing.T, body []byte) {
	t.Helper()
	if len(body) > message.DefaultMaxSize {
		t.Fatalf("message de %d octets accepté au-delà de la taille maximale", len(body))
	}
	var accepted acceptedMessage
	if err := json.Unmarshal(body, &accepted); err != nil {
		t.Fatalf("message accepté illisible: %v", err)
	}
	if accepted.Version != 1 || (accepted.Type != requestType && accepted.Type != resultType) {
		t.Fatalf("contrat inconnu accepté: %s v%d", accepted.Type, accepted.Version)
	}
	if accepted.Data.Issue != nil && !slices.Contains(closedIssues, *accepted.Data.Issue) {
		t.Fatalf("issue hors liste acceptée: %q", *accepted.Data.Issue)
	}
	if accepted.Data.Reason != nil && !slices.Contains(closedReasons, *accepted.Data.Reason) {
		t.Fatalf("motif hors liste accepté: %q", *accepted.Data.Reason)
	}
	for key := range accepted.Data.Seller {
		if strings.Contains(strings.ToLower(key), "mail") {
			t.Fatalf("email du vendeur accepté")
		}
	}
}
