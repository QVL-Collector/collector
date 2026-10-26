package message_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"collector/libs/platform/message"
)

const (
	examplesRoot   = "messages"
	validDir       = "valides"
	invalidDir     = "invalides"
	requestType    = "controle.demande"
	resultType     = "controle.resultat"
	minLeakedValue = 3
)

var (
	knownReasons = map[message.Reason]bool{
		message.ReasonTooLarge:        true,
		message.ReasonMalformed:       true,
		message.ReasonDuplicateKey:    true,
		message.ReasonUnknownContract: true,
		message.ReasonSchemaViolation: true,
	}
	closedIssues  = []string{"publie", "refuse", "a_verifier"}
	closedReasons = []string{"annonce_incomplete", "coordonnees_detectees", "compte_non_autorise", "prix_a_confirmer", "vendeur_a_verifier", "categorie_inconnue"}
	stringValueRe = regexp.MustCompile(`:\s*"((?:[^"\\]|\\.)*)"`)
	numberValueRe = regexp.MustCompile(`:\s*(-?[0-9][0-9.eE+-]*)`)
)

type example struct {
	name   string
	body   []byte
	valid  bool
	reason message.Reason
}

func contractsFS() fs.FS {
	return os.DirFS(filepath.Join("..", "..", "..", "contracts"))
}

func newValidator(t testing.TB, maxSize int) *message.Validator {
	t.Helper()
	validator, err := message.NewValidator(contractsFS(), maxSize)
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	return validator
}

func loadExamples(t testing.TB) []example {
	t.Helper()
	root := contractsFS()
	var examples []example
	err := fs.WalkDir(root, examplesRoot, func(p string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || path.Ext(p) != ".json" || !strings.Contains(p, "/exemples/") {
			return err
		}
		body, readErr := fs.ReadFile(root, p)
		if readErr != nil {
			return readErr
		}
		examples = append(examples, classify(t, p, body))
		return nil
	})
	if err != nil {
		t.Fatalf("lecture des exemples: %v", err)
	}
	sort.Slice(examples, func(i, j int) bool { return examples[i].name < examples[j].name })
	return examples
}

func classify(t testing.TB, p string, body []byte) example {
	t.Helper()
	parent := path.Base(path.Dir(p))
	grandParent := path.Base(path.Dir(path.Dir(p)))
	switch {
	case parent == validDir:
		return example{name: p, body: body, valid: true}
	case grandParent == invalidDir:
		return example{name: p, body: body, reason: message.Reason(parent)}
	default:
		t.Fatalf("exemple hors convention valides/ ou invalides/<raison>/: %s", p)
		return example{}
	}
}

func readExample(t testing.TB, messageType, kind, name string) []byte {
	t.Helper()
	body, err := fs.ReadFile(contractsFS(), path.Join(examplesRoot, messageType, "exemples", "v1", kind, name))
	if err != nil {
		t.Fatalf("lecture de %s: %v", name, err)
	}
	return body
}

func asMap(t testing.TB, body []byte) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("exemple de base illisible: %v", err)
	}
	return decoded
}

func marshal(t testing.TB, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return body
}

func dataOf(document map[string]any) map[string]any {
	return document["data"].(map[string]any)
}

func rejection(t testing.TB, err error) *message.RejectionError {
	t.Helper()
	var rejected *message.RejectionError
	if !errors.As(err, &rejected) {
		t.Fatalf("refus attendu sous forme de RejectionError, obtenu %v", err)
	}
	return rejected
}

func expectAccepted(t testing.TB, validator *message.Validator, body []byte) {
	t.Helper()
	if err := validator.Validate(body); err != nil {
		t.Fatalf("message attendu accepté, refusé: %v", err)
	}
}

func rejectionOf(t testing.TB, validator *message.Validator, body []byte) *message.RejectionError {
	t.Helper()
	err := validator.Validate(body)
	if err == nil {
		t.Fatalf("message attendu refusé, accepté: %s", body)
	}
	return rejection(t, err)
}

func expectKnownReason(t testing.TB, rejected *message.RejectionError) {
	t.Helper()
	if !knownReasons[rejected.Reason] {
		t.Fatalf("raison hors liste: %q", rejected.Reason)
	}
}

func expectRejected(t testing.TB, validator *message.Validator, body []byte) {
	t.Helper()
	expectKnownReason(t, rejectionOf(t, validator, body))
}

func expectRejectedWith(t testing.TB, validator *message.Validator, body []byte, want message.Reason) {
	t.Helper()
	rejected := rejectionOf(t, validator, body)
	expectKnownReason(t, rejected)
	if rejected.Reason != want {
		t.Fatalf("raison attendue %q, obtenue %q (détail: %s)", want, rejected.Reason, rejected.Detail)
	}
	reason, ok := message.ReasonOf(validator.Validate(body))
	if !ok || reason != want {
		t.Fatalf("ReasonOf attendu %q, obtenu %q (%v)", want, reason, ok)
	}
}

func expectRejectedWithoutLeak(t testing.TB, validator *message.Validator, body []byte) {
	t.Helper()
	expectRejected(t, validator, body)
	expectDetailWithoutReceivedValues(t, rejectionOf(t, validator, body), body)
}

func expectRejectedWithReasonWithoutLeak(t testing.TB, validator *message.Validator, body []byte, want message.Reason) {
	t.Helper()
	expectRejectedWith(t, validator, body, want)
	expectDetailWithoutReceivedValues(t, rejectionOf(t, validator, body), body)
}

func receivedValues(body []byte) []string {
	var values []string
	for _, match := range stringValueRe.FindAllSubmatch(body, -1) {
		values = append(values, string(match[1]))
	}
	for _, match := range numberValueRe.FindAllSubmatch(body, -1) {
		values = append(values, string(match[1]))
	}
	var significant []string
	for _, value := range values {
		if len(value) >= minLeakedValue {
			significant = append(significant, value)
		}
	}
	return significant
}

func expectDetailWithoutReceivedValues(t testing.TB, rejected *message.RejectionError, body []byte) {
	t.Helper()
	if strings.TrimSpace(rejected.Detail) == "" {
		t.Fatalf("détail du refus vide")
	}
	for _, value := range receivedValues(body) {
		if strings.Contains(rejected.Detail, value) || strings.Contains(rejected.Error(), value) {
			t.Fatalf("le refus recopie la valeur reçue %q: %s", value, rejected.Error())
		}
	}
}
