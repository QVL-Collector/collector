package httpjson_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"collector/libs/platform/httpjson"
)

func decoderEnIgnorantLesInconnus(t *testing.T, typeContenu string, corps []byte) (saisie, error) {
	t.Helper()
	requete := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader(corps))
	requete.Header.Set("Content-Type", typeContenu)
	var cible saisie
	err := httpjson.DecodeIgnoringUnknownFields(httptest.NewRecorder(), requete, limite, &cible)
	return cible, err
}

func TestChampsInconnusIgnores(t *testing.T) {
	cas := map[string]string{
		"champ inconnu simple":    `{"titre":"a","statut":"publie"}`,
		"champ inconnu imbriqué":  `{"titre":"a","montant":{"valeur":1}}`,
		"champ connu, casse":      `{"TITRE":"a"}`,
		"seulement des inconnus":  `{"acheteurId":"x"}`,
		"inconnu de type tableau": `{"titre":"a","liste":[1,2]}`,
	}
	for nom, corps := range cas {
		t.Run(nom, func(t *testing.T) {
			if _, err := decoderEnIgnorantLesInconnus(t, "application/json", []byte(corps)); err != nil {
				t.Fatalf("corps %s refusé : %v", corps, err)
			}
		})
	}
	cible, err := decoderEnIgnorantLesInconnus(t, "application/json", []byte(`{"titre":"a","prix":{"montantCentimes":3},"commission":9}`))
	if err != nil || cible.Titre != "a" || cible.Prix == nil || cible.Prix.MontantCentimes != 3 {
		t.Fatalf("champs connus mal décodés : %+v, %v", cible, err)
	}
}

func TestCorpsRefusesMemeEnIgnorantLesInconnus(t *testing.T) {
	trop := append([]byte(`{"titre":"a"}`), bytes.Repeat([]byte(" "), limite)...)
	cas := map[string]struct {
		typeContenu string
		corps       []byte
		raison      httpjson.Reason
	}{
		"clé connue dupliquée":   {"application/json", []byte(`{"titre":"a","titre":"b"}`), httpjson.ReasonDuplicateKey},
		"clé dupliquée en casse": {"application/json", []byte(`{"titre":"a","Titre":"b"}`), httpjson.ReasonDuplicateKey},
		"clé inconnue dupliquée": {"application/json", []byte(`{"x":1,"x":2}`), httpjson.ReasonDuplicateKey},
		"type invalide":          {"application/json", []byte(`{"titre":3}`), httpjson.ReasonInvalidType},
		"type invalide imbriqué": {"application/json", []byte(`{"prix":{"montantCentimes":"3"}}`), httpjson.ReasonInvalidType},
		"tableau":                {"application/json", []byte(`[]`), httpjson.ReasonNotAnObject},
		"mal formé":              {"application/json", []byte(`{"titre":`), httpjson.ReasonMalformed},
		"trop volumineux":        {"application/json", trop, httpjson.ReasonTooLarge},
		"texte":                  {"text/plain", []byte(`{}`), httpjson.ReasonUnsupportedMediaType},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			_, err := decoderEnIgnorantLesInconnus(t, c.typeContenu, c.corps)
			var rejet *httpjson.RejectionError
			if !errors.As(err, &rejet) || rejet.Reason != c.raison {
				t.Fatalf("refus %s attendu, obtenu %v", c.raison, err)
			}
		})
	}
}
