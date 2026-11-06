package message_test

import (
	"slices"
	"strings"
	"testing"

	"collector/libs/platform/message"
)

var (
	adminRefusalReasons = []string{"prix_abusif", "article_interdit", "vendeur_non_fiable"}
	decisionMembers     = []string{"article_id", "revision", "seller_id", "issue", "reason", "decided_at"}
)

func decisionWith(t *testing.T, mutate func(document map[string]any)) []byte {
	t.Helper()
	document := asMap(t, readExample(t, decisionType, validDir, "refuse-par-un-administrateur.json"))
	mutate(document)
	return marshal(t, document)
}

func TestUS028AC9_DecisionDecritArticleRevisionVendeurOpaqueIssueEtDate(t *testing.T) {
	schema := readSchema(t, decisionType+"/v1.schema.json")
	defs := node(t, schema, "$defs")
	decision := node(t, defs, "decision")
	expectRequired(t, decision, "article_id", "revision", "seller_id", "issue", "decided_at")
	if format := node(t, decision, "properties", "seller_id")["format"]; format != "uuid" {
		t.Errorf("vendeur attendu opaque (uuid), format %v", format)
	}
	expectSameSet(t, stringsOf(t, node(t, defs, "issue")["enum"]), closedIssues)
}

func TestUS028AC9_MotifsDeLaDecisionCouvrentLeControleEtLAdministrateur(t *testing.T) {
	reasons := stringsOf(t, node(t, readSchema(t, decisionType+"/v1.schema.json"), "$defs", "motif")["enum"])
	expectSameSet(t, reasons, append(slices.Clone(closedReasons), adminRefusalReasons...))
}

func TestUS028AC9_ChaqueIssueEtChaqueMotifSontAcceptes(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	for _, issue := range []string{"refuse", "a_verifier"} {
		for _, reason := range append(slices.Clone(closedReasons), adminRefusalReasons...) {
			t.Run(issue+"/"+reason, func(t *testing.T) {
				expectAccepted(t, validator, decisionWith(t, func(d map[string]any) {
					dataOf(d)["issue"] = issue
					dataOf(d)["reason"] = reason
				}))
			})
		}
	}
	t.Run("publie sans motif", func(t *testing.T) {
		expectAccepted(t, validator, decisionWith(t, func(d map[string]any) {
			dataOf(d)["issue"] = "publie"
			delete(dataOf(d), "reason")
		}))
	})
}

func TestUS028AC5_IssueOuMotifHorsListeFermeeEstRefuse(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	cases := map[string]func(map[string]any){
		"issue en majuscules":      func(d map[string]any) { dataOf(d)["issue"] = "REFUSE" },
		"issue inconnue":           func(d map[string]any) { dataOf(d)["issue"] = "suspendu" },
		"issue vide":               func(d map[string]any) { dataOf(d)["issue"] = "" },
		"issue nulle":              func(d map[string]any) { dataOf(d)["issue"] = nil },
		"motif inconnu":            func(d map[string]any) { dataOf(d)["reason"] = "fraude_suspectee" },
		"motif vide":               func(d map[string]any) { dataOf(d)["reason"] = "" },
		"motif nul":                func(d map[string]any) { dataOf(d)["reason"] = nil },
		"motif en texte libre":     func(d map[string]any) { dataOf(d)["reason"] = "Votre figurine contient un numero" },
		"motif structuré":          func(d map[string]any) { dataOf(d)["reason"] = map[string]any{"code": "article_interdit"} },
		"refusé sans motif":        func(d map[string]any) { delete(dataOf(d), "reason") },
		"à vérifier sans motif":    func(d map[string]any) { dataOf(d)["issue"] = "a_verifier"; delete(dataOf(d), "reason") },
		"publié avec motif":        func(d map[string]any) { dataOf(d)["issue"] = "publie" },
		"révision nulle":           func(d map[string]any) { dataOf(d)["revision"] = 0 },
		"révision au-delà d'int32": func(d map[string]any) { dataOf(d)["revision"] = 2147483648 },
		"vendeur non opaque":       func(d map[string]any) { dataOf(d)["seller_id"] = "vendeur.fictif@example.com" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			expectRejectedWith(t, validator, decisionWith(t, mutate), message.ReasonSchemaViolation)
		})
	}
}

func TestUS028AC5_VersionInconnueDeLaDecisionEstRefusee(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	for name, version := range map[string]any{"zero": 0, "suivante": 2, "texte": "v1", "nulle": nil} {
		t.Run(name, func(t *testing.T) {
			expectRejected(t, validator, decisionWith(t, func(d map[string]any) { d["version"] = version }))
		})
	}
}

func TestUS028AC4_AucunContenuSaisiParUnUtilisateurNEstAccepte(t *testing.T) {
	validator := newValidator(t, message.DefaultMaxSize)
	for _, member := range []string{"title", "titre", "description", "seller_email", "comment", "message", "text"} {
		t.Run(member, func(t *testing.T) {
			body := decisionWith(t, func(d map[string]any) { dataOf(d)[member] = "Figurine Tintin rare, appelez le 01 99 00 12 34" })
			expectRejectedWithReasonWithoutLeak(t, validator, body, message.ReasonSchemaViolation)
		})
	}
	t.Run("dans l'enveloppe", func(t *testing.T) {
		body := decisionWith(t, func(d map[string]any) { d["title"] = "Figurine Tintin rare" })
		expectRejectedWithReasonWithoutLeak(t, validator, body, message.ReasonSchemaViolation)
	})
}

func TestUS028AC4_ExemplesValidesNePortentQueDesCodesEtDesIdentifiants(t *testing.T) {
	for _, ex := range loadExamples(t) {
		if !ex.valid || !strings.Contains(ex.name, "/"+decisionType+"/") {
			continue
		}
		t.Run(ex.name, func(t *testing.T) {
			for key := range dataOf(asMap(t, ex.body)) {
				if !slices.Contains(decisionMembers, key) {
					t.Fatalf("membre %q hors des codes et identifiants de la décision", key)
				}
			}
		})
	}
}
