package httpjson_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"collector/libs/platform/httpjson"
)

type saisie struct {
	Titre string `json:"titre"`
	Prix  *struct {
		MontantCentimes int64  `json:"montantCentimes"`
		Devise          string `json:"devise"`
	} `json:"prix"`
}

const limite = 64

func decoder(t *testing.T, typeContenu string, corps []byte) (saisie, error) {
	t.Helper()
	requete := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader(corps))
	if typeContenu != "" {
		requete.Header.Set("Content-Type", typeContenu)
	}
	var cible saisie
	err := httpjson.Decode(httptest.NewRecorder(), requete, limite, &cible)
	return cible, err
}

func TestCorpsValideDecode(t *testing.T) {
	cible, err := decoder(t, "application/json; charset=utf-8", []byte(`{"titre":"a","prix":{"montantCentimes":3,"devise":"EUR"}}`))
	if err != nil || cible.Titre != "a" || cible.Prix == nil || cible.Prix.MontantCentimes != 3 {
		t.Fatalf("décodage inattendu : %+v, %v", cible, err)
	}
}

func TestCorpsALaLimiteAccepte(t *testing.T) {
	corps := append([]byte(`{"titre":"a"}`), bytes.Repeat([]byte(" "), limite-len(`{"titre":"a"}`))...)
	if _, err := decoder(t, "application/json", corps); err != nil {
		t.Fatalf("corps de %d octets refusé : %v", len(corps), err)
	}
}

func TestCorpsRefuses(t *testing.T) {
	trop := append([]byte(`{"titre":"a"}`), bytes.Repeat([]byte(" "), limite)...)
	cas := map[string]struct {
		typeContenu string
		corps       []byte
		raisons     []httpjson.Reason
		statut      int
	}{
		"sans type":               {"", []byte(`{}`), []httpjson.Reason{httpjson.ReasonUnsupportedMediaType}, http.StatusUnsupportedMediaType},
		"texte":                   {"text/plain", []byte(`{}`), []httpjson.Reason{httpjson.ReasonUnsupportedMediaType}, http.StatusUnsupportedMediaType},
		"trop volumineux":         {"application/json", trop, []httpjson.Reason{httpjson.ReasonTooLarge}, http.StatusRequestEntityTooLarge},
		"mal formé":               {"application/json", []byte(`{"titre":`), []httpjson.Reason{httpjson.ReasonMalformed}, http.StatusBadRequest},
		"clé dupliquée":           {"application/json", []byte(`{"titre":"a","titre":"b"}`), []httpjson.Reason{httpjson.ReasonDuplicateKey}, http.StatusBadRequest},
		"clé dupliquée casse":     {"application/json", []byte(`{"titre":"a","TITRE":"b"}`), []httpjson.Reason{httpjson.ReasonDuplicateKey, httpjson.ReasonUnknownField}, http.StatusBadRequest},
		"clé dupliquée imbriquée": {"application/json", []byte(`{"prix":{"devise":"EUR","devise":"USD"}}`), []httpjson.Reason{httpjson.ReasonDuplicateKey}, http.StatusBadRequest},
		"tableau":                 {"application/json", []byte(`[]`), []httpjson.Reason{httpjson.ReasonNotAnObject}, http.StatusBadRequest},
		"champ inconnu":           {"application/json", []byte(`{"statut":"publie"}`), []httpjson.Reason{httpjson.ReasonUnknownField}, http.StatusBadRequest},
		"type invalide":           {"application/json", []byte(`{"titre":3}`), []httpjson.Reason{httpjson.ReasonInvalidType}, http.StatusBadRequest},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			_, err := decoder(t, c.typeContenu, c.corps)
			var rejet *httpjson.RejectionError
			if !errors.As(err, &rejet) {
				t.Fatalf("RejectionError attendue, obtenu %v", err)
			}
			if !slices.Contains(c.raisons, rejet.Reason) || rejet.Problem().Status != c.statut {
				t.Fatalf("raison %q statut %d, attendus %v %d", rejet.Reason, rejet.Problem().Status, c.raisons, c.statut)
			}
		})
	}
}
