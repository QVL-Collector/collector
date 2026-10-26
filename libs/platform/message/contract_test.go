package message_test

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"collector/libs/platform/message"
)

type schemaNode = map[string]any

func readSchema(t *testing.T, relative string) schemaNode {
	t.Helper()
	body, err := fs.ReadFile(contractsFS(), path.Join(examplesRoot, relative))
	if err != nil {
		t.Fatalf("schéma %s absent: %v", relative, err)
	}
	return asMap(t, body)
}

func node(t *testing.T, root schemaNode, keys ...string) schemaNode {
	t.Helper()
	current := root
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			t.Fatalf("membre %s absent du schéma", strings.Join(keys, "/"))
		}
		current = next
	}
	return current
}

func stringsOf(t *testing.T, raw any) []string {
	t.Helper()
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("liste attendue, obtenu %v", raw)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.(string))
	}
	return out
}

func expectRequired(t *testing.T, definition schemaNode, names ...string) {
	t.Helper()
	required := stringsOf(t, definition["required"])
	properties := definition["properties"].(map[string]any)
	for _, name := range names {
		if !slices.Contains(required, name) {
			t.Errorf("membre %q non requis", name)
		}
		if _, ok := properties[name]; !ok {
			t.Errorf("membre %q non décrit", name)
		}
	}
	if definition["additionalProperties"] != false {
		t.Errorf("additionalProperties doit être false")
	}
}

func expectSameSet(t *testing.T, got, want []string) {
	t.Helper()
	slices.Sort(got)
	sorted := slices.Clone(want)
	slices.Sort(sorted)
	if !slices.Equal(got, sorted) {
		t.Fatalf("liste fermée attendue %v, obtenue %v", sorted, got)
	}
}

func TestUS012AC1_EnveloppeCommuneDecritVersionEtCorrelation(t *testing.T) {
	envelope := readSchema(t, "enveloppe-v1.schema.json")
	expectRequired(t, envelope, "id", "type", "version", "emitted_at", "correlation_id", "data")
}

func TestUS012AC1_DemandeDecritArticleRevisionCompteOpaqueStatutEtPhotos(t *testing.T) {
	schema := readSchema(t, "controle.demande/v1.schema.json")
	defs := node(t, schema, "$defs")
	expectRequired(t, node(t, defs, "demande"), "article_id", "revision", "seller", "listing")
	seller := node(t, defs, "vendeur")
	expectRequired(t, seller, "account_id", "member_since", "suspended")
	if format := node(t, seller, "properties", "account_id")["format"]; format != "uuid" {
		t.Errorf("identifiant de compte attendu opaque (uuid), format %v", format)
	}
	expectRequired(t, node(t, defs, "annonce"), "photo_count")
}

func TestUS012AC1_ResultatDecritVersionDesReglesIssueEtCategorieDeMotif(t *testing.T) {
	schema := readSchema(t, "controle.resultat/v1.schema.json")
	defs := node(t, schema, "$defs")
	expectRequired(t, node(t, defs, "resultat"), "article_id", "revision", "rules_version", "issue")
	if _, ok := node(t, defs, "resultat", "properties")["reason"]; !ok {
		t.Errorf("catégorie de motif absente du résultat")
	}
	expectSameSet(t, stringsOf(t, node(t, defs, "issue")["enum"]), closedIssues)
	expectSameSet(t, stringsOf(t, node(t, defs, "motif")["enum"]), closedReasons)
}

func TestUS012AC1_ChaqueIssueEtChaqueCategorieDeMotifSontAcceptees(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	base := readExample(t, resultType, validDir, "refuse-annonce-incomplete.json")
	for _, issue := range []string{"refuse", "a_verifier"} {
		for _, reason := range closedReasons {
			t.Run(issue+"/"+reason, func(t *testing.T) {
				document := asMap(t, base)
				dataOf(document)["issue"] = issue
				dataOf(document)["reason"] = reason
				expectAccepted(t, validator, marshal(t, document))
			})
		}
	}
}

func TestUS012AC1_DecodeRestitueLEnveloppeEtLesDonnees(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	body := readExample(t, resultType, validDir, "refuse-annonce-incomplete.json")
	var data struct {
		ArticleID    string `json:"article_id"`
		Revision     int    `json:"revision"`
		RulesVersion int    `json:"rules_version"`
		Issue        string `json:"issue"`
		Reason       string `json:"reason"`
	}
	envelope, err := validator.Decode(body, &data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if envelope.Type != resultType || envelope.Version != 1 || envelope.CorrelationID != "2c8e4a61-9b7d-4f3e-a5c1-d0e2f4a6b8c9" {
		t.Fatalf("enveloppe inattendue: %+v", envelope)
	}
	if data.Revision != 3 || data.RulesVersion != 1 || data.Issue != "refuse" || data.Reason != "annonce_incomplete" {
		t.Fatalf("données inattendues: %+v", data)
	}
}

func TestUS012AC2_ChaqueExempleEstAccepteOuRefuseAvecSaRaison(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	examples := loadExamples(t)
	for _, messageType := range []string{requestType, resultType} {
		valid, invalid := 0, 0
		for _, ex := range examples {
			if strings.Contains(ex.name, "/"+messageType+"/") {
				if ex.valid {
					valid++
				} else {
					invalid++
				}
			}
		}
		if valid == 0 || invalid == 0 {
			t.Fatalf("%s: exemples valides (%d) et invalides (%d) attendus", messageType, valid, invalid)
		}
	}
	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			if ex.valid {
				expectAccepted(t, validator, ex.body)
				var data map[string]any
				if _, err := validator.Decode(ex.body, &data); err != nil {
					t.Fatalf("Decode refuse un exemple valide: %v", err)
				}
				return
			}
			if !knownReasons[ex.reason] {
				t.Fatalf("dossier de raison inconnu: %q", ex.reason)
			}
			rejected := expectRejectedWith(t, validator, ex.body, ex.reason)
			expectDetailWithoutReceivedValues(t, rejected, ex.body)
			var data map[string]any
			if reason, _ := message.ReasonOf(decodeErr(validator, ex.body, &data)); reason != ex.reason {
				t.Fatalf("Decode: raison attendue %q, obtenue %q", ex.reason, reason)
			}
		})
	}
}

func decodeErr(validator *message.Validator, body []byte, data any) error {
	_, err := validator.Decode(body, data)
	return err
}

func TestUS012AC2_MessageAuDelaDeLaTailleConfigureeEstRefuseTailleDepassee(t *testing.T) {
	base := readExample(t, requestType, validDir, "annonce-complete.json")
	limit := len(base) + 16
	validator := newValidator(t, limit)
	atLimit := append(bytes.Clone(base), bytes.Repeat([]byte(" "), limit-len(base))...)
	expectAccepted(t, validator, atLimit)
	overLimit := append(bytes.Clone(atLimit), ' ')
	rejected := expectRejectedWith(t, validator, overLimit, message.ReasonTooLarge)
	expectDetailWithoutReceivedValues(t, rejected, overLimit)
}

func TestUS012AC2_MessageAuDelaDeLaTailleParDefautEstRefuseTailleDepassee(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	document := asMap(t, readExample(t, requestType, validDir, "annonce-complete.json"))
	dataOf(document)["listing"].(map[string]any)["description"] = strings.Repeat("a", message.DefaultMaxSize)
	expectRejectedWith(t, validator, marshal(t, document), message.ReasonTooLarge)
}

func TestUS012AC2_CorpsQuiNEstPasUnMessageJSONEstRefuse(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	cases := map[string]string{
		"vide":          "",
		"espaces":       "   ",
		"null":          "null",
		"tableau":       "[]",
		"objet vide":    "{}",
		"texte":         `"controle.demande"`,
		"utf8 invalide": "{\"type\":\"\xff\xfe\"}",
		"deux messages": "{} {}",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			rejected := expectRejected(t, validator, []byte(body))
			expectDetailWithoutReceivedValues(t, rejected, []byte(body))
		})
	}
}

func resultWith(t *testing.T, mutate func(document map[string]any)) []byte {
	t.Helper()
	document := asMap(t, readExample(t, resultType, validDir, "refuse-annonce-incomplete.json"))
	mutate(document)
	return marshal(t, document)
}

func requestWith(t *testing.T, mutate func(document map[string]any)) []byte {
	t.Helper()
	document := asMap(t, readExample(t, requestType, validDir, "annonce-complete.json"))
	mutate(document)
	return marshal(t, document)
}

func TestUS012AC3_IssueHorsListeFermeeEstRefusee(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	for name, issue := range map[string]any{
		"majuscules":   "PUBLIE",
		"accentuee":    "publié",
		"vide":         "",
		"espace final": "refuse ",
		"null":         nil,
		"nombre":       1,
		"liste":        []any{"publie"},
	} {
		t.Run(name, func(t *testing.T) {
			body := resultWith(t, func(document map[string]any) { dataOf(document)["issue"] = issue })
			expectRejectedWith(t, validator, body, message.ReasonSchemaViolation)
		})
	}
}

func TestUS012AC3_CategorieDeMotifHorsListeFermeeEstRefusee(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	for name, reason := range map[string]any{
		"majuscules":     "ANNONCE_INCOMPLETE",
		"vide":           "",
		"null":           nil,
		"inconnue":       "fraude_suspectee",
		"liste":          []any{"annonce_incomplete"},
		"issue en motif": "publie",
	} {
		t.Run(name, func(t *testing.T) {
			body := resultWith(t, func(document map[string]any) { dataOf(document)["reason"] = reason })
			expectRejectedWith(t, validator, body, message.ReasonSchemaViolation)
		})
	}
}

func TestUS012AC3_VersionInconnueEstRefusee(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	for name, version := range map[string]any{
		"zero":      0,
		"negative":  -1,
		"suivante":  2,
		"lointaine": 99999,
		"decimale":  1.5,
		"texte":     "v1",
		"null":      nil,
		"booleen":   true,
	} {
		for _, build := range []func(*testing.T, func(map[string]any)) []byte{requestWith, resultWith} {
			body := build(t, func(document map[string]any) { document["version"] = version })
			t.Run(name, func(t *testing.T) {
				expectRejected(t, validator, body)
			})
		}
	}
}

func TestUS012AC3_TypeInconnuEstRefuse(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	for _, messageType := range []string{"", "Controle.Resultat", "controle.resultat ", "controle", "enveloppe-v1"} {
		t.Run(messageType, func(t *testing.T) {
			body := resultWith(t, func(document map[string]any) { document["type"] = messageType })
			expectRejected(t, validator, body)
		})
	}
}

func TestUS012AC3_EncodeRefuseDEmettreUneIssueHorsListe(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	contract := message.Contract{Type: resultType, Version: 1}
	data := map[string]any{"article_id": "3f1c2a8e-5b7d-4e2a-9c1f-6d8b0a4e7c21", "revision": 1, "rules_version": 1, "issue": "approuve"}
	envelope, err := message.NewEnvelope(contract, "7a9d2e14-c6b3-4f81-a0d5-e2c47b19f863", data, time.Date(2026, 11, 2, 9, 15, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	if _, err := validator.Encode(envelope); err == nil {
		t.Fatalf("Encode a émis un résultat dont l'issue est hors liste")
	}
	data["issue"] = "publie"
	envelope, err = message.NewEnvelope(contract, "7a9d2e14-c6b3-4f81-a0d5-e2c47b19f863", data, time.Date(2026, 11, 2, 9, 15, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	body, err := validator.Encode(envelope)
	if err != nil {
		t.Fatalf("Encode refuse un résultat conforme: %v", err)
	}
	expectAccepted(t, validator, body)
}

func TestUS012AC3_VersionInconnueRefuseeALEmission(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	data := map[string]any{"article_id": "3f1c2a8e-5b7d-4e2a-9c1f-6d8b0a4e7c21", "revision": 1, "rules_version": 1, "issue": "publie"}
	envelope, err := message.NewEnvelope(message.Contract{Type: resultType, Version: 2}, "7a9d2e14-c6b3-4f81-a0d5-e2c47b19f863", data, time.Now().UTC())
	if err != nil {
		return
	}
	if _, err := validator.Encode(envelope); err == nil {
		t.Fatalf("Encode a émis un message de version inconnue")
	}
}

func TestUS012AC4_CleDupliqueeEstRefusee(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	result := string(readExample(t, resultType, validDir, "refuse-annonce-incomplete.json"))
	request := string(readExample(t, requestType, validDir, "annonce-complete.json"))
	cases := map[string]string{
		"version en double dans l'enveloppe": strings.Replace(result, `"version": 1,`, `"version": 1, "version": 1,`, 1),
		"type en double":                     strings.Replace(result, `"version": 1,`, `"version": 1, "type": "controle.demande",`, 1),
		"data en double":                     strings.Replace(result, `"data": {`, `"data": {}, "data": {`, 1),
		"motif en double de même valeur":     strings.Replace(result, `"reason": "annonce_incomplete"`, `"reason": "annonce_incomplete", "reason": "annonce_incomplete"`, 1),
		"cle echappee en unicode":            strings.Replace(result, `"reason": "annonce_incomplete"`, `"reason": "annonce_incomplete", "reason": "prix_a_confirmer"`, 1),
		"compte en double":                   strings.Replace(request, `"suspended": false`, `"suspended": false, "account_id": "9b1d3f5a-7c9e-4b2d-a4f6-1c3e5a7b9d2f"`, 1),
		"photos en double":                   strings.Replace(request, `"photo_count": 2`, `"photo_count": 2, "photo_count": 0`, 1),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(body, "\n") {
				t.Fatalf("préparation du cas impossible")
			}
			rejected := expectRejectedWith(t, validator, []byte(body), message.ReasonDuplicateKey)
			expectDetailWithoutReceivedValues(t, rejected, []byte(body))
		})
	}
}

func TestUS012AC4_ChampInconnuEstRefuseAChaqueNiveau(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	cases := map[string][]byte{
		"enveloppe du résultat": resultWith(t, func(d map[string]any) { d["trace"] = "00-abc" }),
		"données du résultat":   resultWith(t, func(d map[string]any) { dataOf(d)["comment"] = "ok" }),
		"données de la demande": requestWith(t, func(d map[string]any) { dataOf(d)["priority"] = "haute" }),
		"vendeur":               requestWith(t, func(d map[string]any) { dataOf(d)["seller"].(map[string]any)["rating"] = 5 }),
		"annonce":               requestWith(t, func(d map[string]any) { dataOf(d)["listing"].(map[string]any)["condition"] = "neuf" }),
		"champ vide":            requestWith(t, func(d map[string]any) { dataOf(d)[""] = "x" }),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			rejected := expectRejectedWith(t, validator, body, message.ReasonSchemaViolation)
			expectDetailWithoutReceivedValues(t, rejected, body)
		})
	}
}

func TestUS012AC4_EmailDuVendeurEstRefuse(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	const email = "vendeur.fictif@example.com"
	cases := map[string][]byte{
		"email dans le vendeur":             requestWith(t, func(d map[string]any) { dataOf(d)["seller"].(map[string]any)["email"] = email }),
		"email dans les données":            requestWith(t, func(d map[string]any) { dataOf(d)["seller_email"] = email }),
		"email dans l'enveloppe":            requestWith(t, func(d map[string]any) { d["email"] = email }),
		"email dans l'annonce":              requestWith(t, func(d map[string]any) { dataOf(d)["listing"].(map[string]any)["contact_email"] = email }),
		"email comme identifiant de compte": requestWith(t, func(d map[string]any) { dataOf(d)["seller"].(map[string]any)["account_id"] = email }),
		"email dans le résultat":            resultWith(t, func(d map[string]any) { dataOf(d)["seller_email"] = email }),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			rejected := expectRejectedWith(t, validator, body, message.ReasonSchemaViolation)
			expectDetailWithoutReceivedValues(t, rejected, body)
		})
	}
}

func TestUS012AC5_MotifDesResultatsValidesEstUneCategorieSansRegleNiSeuil(t *testing.T) {
	for _, ex := range loadExamples(t) {
		if !ex.valid || !strings.Contains(ex.name, "/"+resultType+"/") {
			continue
		}
		t.Run(ex.name, func(t *testing.T) {
			data := dataOf(asMap(t, ex.body))
			for key := range data {
				if !slices.Contains([]string{"article_id", "revision", "rules_version", "issue", "reason"}, key) {
					t.Fatalf("membre %q en plus de la catégorie de motif", key)
				}
			}
			reason, present := data["reason"]
			if !present {
				return
			}
			if !slices.Contains(closedReasons, reason.(string)) {
				t.Fatalf("motif %v hors catégories", reason)
			}
		})
	}
}

func TestUS012AC5_CategoriesDeMotifNeCitentNiRegleNiSeuil(t *testing.T) {
	for _, reason := range closedReasons {
		if strings.ContainsAny(reason, "0123456789<>=%€$") {
			t.Errorf("catégorie %q porte un seuil", reason)
		}
	}
}

func TestUS012AC5_ResultatDivulguantRegleOuSeuilEstRefuse(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	cases := map[string][]byte{
		"règle exacte":         resultWith(t, func(d map[string]any) { dataOf(d)["rule"] = "titre-trop-court" }),
		"seuil appliqué":       resultWith(t, func(d map[string]any) { dataOf(d)["threshold"] = 20 }),
		"motif en texte libre": resultWith(t, func(d map[string]any) { dataOf(d)["reason"] = "titre de moins de 20 caracteres" }),
		"motif structuré": resultWith(t, func(d map[string]any) {
			dataOf(d)["reason"] = map[string]any{"category": "annonce_incomplete", "threshold": 20}
		}),
		"catégorie suffixée du seuil": resultWith(t, func(d map[string]any) { dataOf(d)["reason"] = "prix_a_confirmer:500000" }),
		"détail du motif":             resultWith(t, func(d map[string]any) { dataOf(d)["reason_detail"] = "prix > 5000" }),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			rejected := expectRejectedWith(t, validator, body, message.ReasonSchemaViolation)
			expectDetailWithoutReceivedValues(t, rejected, body)
		})
	}
}

func TestUS012AC5_IssueNonPublieeSansCategorieOuPublieeAvecCategorieEstRefusee(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	cases := map[string][]byte{
		"à vérifier sans motif": resultWith(t, func(d map[string]any) { dataOf(d)["issue"] = "a_verifier"; delete(dataOf(d), "reason") }),
		"refusé sans motif":     resultWith(t, func(d map[string]any) { delete(dataOf(d), "reason") }),
		"publié avec motif":     resultWith(t, func(d map[string]any) { dataOf(d)["issue"] = "publie" }),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			expectRejectedWith(t, validator, body, message.ReasonSchemaViolation)
		})
	}
}

func TestUS012AC2_RefusEstStableEtRaisonLisibleParReasonOf(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	if reason, ok := message.ReasonOf(nil); ok || reason != "" {
		t.Fatalf("ReasonOf(nil) doit indiquer l'absence de refus, obtenu %q %v", reason, ok)
	}
	body := resultWith(t, func(d map[string]any) { dataOf(d)["issue"] = "approuve" })
	first := expectRejected(t, validator, body)
	second := expectRejected(t, validator, body)
	if first.Reason != second.Reason || first.Detail != second.Detail {
		t.Fatalf("refus non déterministe: %v / %v", first, second)
	}
	if json.Valid([]byte(first.Detail)) && strings.HasPrefix(strings.TrimSpace(first.Detail), "{") {
		t.Fatalf("le détail recopie le message")
	}
}
