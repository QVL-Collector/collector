package canal

import (
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"collector/contracts"
	"collector/libs/platform/broker"
	"collector/libs/platform/message"
	"collector/services/controle/internal/regles"
)

var exemplesValides = os.DirFS(filepath.Join("..", "..", "..", "..", "contracts", "messages", "controle.demande", "exemples", "v1", "valides"))

type resultatLu struct {
	RulesVersion int     `json:"rules_version"`
	Issue        string  `json:"issue"`
	Reason       *string `json:"reason"`
}

func validateur(t *testing.T) *message.Validator {
	t.Helper()
	v, err := message.NewValidator(contracts.Messages, message.DefaultMaxSize)
	if err != nil {
		t.Fatalf("validateur : %v", err)
	}
	return v
}

func exempleDeDemande(t *testing.T, nom string) []byte {
	t.Helper()
	corps, err := fs.ReadFile(exemplesValides, nom)
	if err != nil {
		t.Fatalf("exemple %s : %v", nom, err)
	}
	return corps
}

func demandeAvec(t *testing.T, corps []byte, modifier func(map[string]any)) []byte {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(corps, &document); err != nil {
		t.Fatalf("exemple : %v", err)
	}
	modifier(document["data"].(map[string]any))
	modifie, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("demande : %v", err)
	}
	return modifie
}

func controler(t *testing.T, corps []byte) (resultatLu, []byte) {
	t.Helper()
	v := validateur(t)
	responder := NewResponder(v, regles.Reference(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	var demande requestData
	enveloppe, err := v.Decode(corps, &demande)
	if err != nil {
		t.Fatalf("demande non conforme au contrat : %v", err)
	}
	verdict := responder.rules.Evaluate(demande.submission())
	publie, err := responder.resultMessage(demande, verdict, enveloppe.CorrelationID)
	if err != nil {
		t.Fatalf("résultat : %v", err)
	}
	var resultat resultatLu
	if _, err := v.Decode(publie.Body, &resultat); err != nil {
		t.Fatalf("résultat non conforme au contrat : %v", err)
	}
	var enveloppeBrute struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(publie.Body, &enveloppeBrute); err != nil {
		t.Fatalf("résultat : %v", err)
	}
	return resultat, enveloppeBrute.Data
}

func exigerResultat(t *testing.T, recu resultatLu, issue, motif string) {
	t.Helper()
	recuMotif := ""
	if recu.Reason != nil {
		recuMotif = *recu.Reason
	}
	if recu.Issue != issue || recuMotif != motif || recu.RulesVersion != 3 {
		t.Fatalf("résultat %s/%s v%d, attendu %s/%s v3", recu.Issue, recuMotif, recu.RulesVersion, issue, motif)
	}
}

func TestUS019AC2ExempleVendeurSuspenduConformeEtRefuse(t *testing.T) {
	resultat, _ := controler(t, exempleDeDemande(t, "vendeur-suspendu.json"))
	exigerResultat(t, resultat, "refuse", "compte_non_autorise")
}

func TestUS019AC5ExempleAnnonceCompletePubliee(t *testing.T) {
	resultat, _ := controler(t, exempleDeDemande(t, "annonce-complete.json"))
	exigerResultat(t, resultat, "publie", "")
}

func TestUS019AC1PrixHorsPlageViaLeContratSansDivulguerLaPlage(t *testing.T) {
	corps := demandeAvec(t, exempleDeDemande(t, "annonce-complete.json"), func(d map[string]any) {
		d["listing"].(map[string]any)["price_cents"] = 300_001
	})
	resultat, donnees := controler(t, corps)
	exigerResultat(t, resultat, "a_verifier", "prix_a_confirmer")
	for _, revele := range []string{"300000", "300001", "3000", "30000", "plage", "seuil", "min", "max"} {
		if strings.Contains(strings.ToLower(string(donnees)), revele) {
			t.Errorf("le résultat révèle %q : %s", revele, donnees)
		}
	}
}

func TestUS019AC3AncienneteLueDansLaDemandeRelativeASaDate(t *testing.T) {
	cas := []struct {
		nom         string
		inscription string
		issue       string
		motif       string
	}{
		{"29 jours", "2026-10-04", "a_verifier", "vendeur_a_verifier"},
		{"30 jours", "2026-10-03", "publie", ""},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			corps := demandeAvec(t, exempleDeDemande(t, "annonce-complete.json"), func(d map[string]any) {
				d["submitted_at"] = "2026-11-02T00:00:00Z"
				d["seller"].(map[string]any)["member_since"] = c.inscription
				d["listing"].(map[string]any)["price_cents"] = 30_001
			})
			resultat, donnees := controler(t, corps)
			exigerResultat(t, resultat, c.issue, c.motif)
			for _, revele := range []string{"30000", "30001", "anciennete", "jours"} {
				if strings.Contains(string(donnees), revele) {
					t.Errorf("le résultat révèle %q : %s", revele, donnees)
				}
			}
		})
	}
}

func TestUS019AC4DemandeAvecNomOuEmailDuVendeurRejeteeParLeContrat(t *testing.T) {
	v := validateur(t)
	for champ, valeur := range map[string]string{"email": "vendeur.fictif@example.com", "display_name": "Vendeur Fictif", "name": "Jean Dupont"} {
		t.Run(champ, func(t *testing.T) {
			corps := demandeAvec(t, exempleDeDemande(t, "annonce-complete.json"), func(d map[string]any) {
				d["seller"].(map[string]any)[champ] = valeur
			})
			if err := v.Validate(corps); err == nil {
				t.Fatalf("demande portant %s acceptée", champ)
			}
		})
	}
}

func TestUS019AC4DateDInscriptionMalFormeeEcarteeSansResultat(t *testing.T) {
	for nom, date := range map[string]string{"format français": "14/03/2025", "jour inexistant": "2025-02-30", "horodatage": "2025-03-14T00:00:00Z", "vide": ""} {
		t.Run(nom, func(t *testing.T) {
			corps := demandeAvec(t, exempleDeDemande(t, "annonce-complete.json"), func(d map[string]any) {
				d["seller"].(map[string]any)["member_since"] = date
			})
			var journal strings.Builder
			responder := NewResponder(validateur(t), regles.Reference(), slog.New(slog.NewJSONHandler(&journal, nil)))
			disposition, err := responder.handle(t.Context(), broker.Delivery{MessageID: "demande-mal-formee", Body: corps}, nil)
			if err != nil || disposition != broker.Reject {
				t.Fatalf("disposition %v, erreur %v : demande non écartée", disposition, err)
			}
			if !strings.Contains(journal.String(), "controle.demande_ecartee") || strings.Contains(journal.String(), "controle.demande_controlee") {
				t.Fatalf("journal inattendu : %s", journal.String())
			}
		})
	}
}
