package regles_test

import (
	"reflect"
	"testing"
	"time"

	"collector/services/controle/internal/regles"
)

type seuilsDocumentes struct {
	categorie string
	minimum   int64
	maximum   int64
	vigilance int64
}

var seuilsDuReadme = []seuilsDocumentes{
	{"bandes-dessinees", 100, 500_000, 30_000},
	{"cartes-a-collectionner", 50, 1_000_000, 50_000},
	{"figurines", 100, 300_000, 30_000},
	{"jouets-anciens", 200, 500_000, 40_000},
	{"monnaies", 100, 2_000_000, 100_000},
	{"montres", 2_000, 5_000_000, 200_000},
	{"vinyles", 100, 200_000, 20_000},
}

var motifsDuContrat = map[regles.Reason]bool{
	"annonce_incomplete":    true,
	"coordonnees_detectees": true,
	"compte_non_autorise":   true,
	"prix_a_confirmer":      true,
	"vendeur_a_verifier":    true,
	"categorie_inconnue":    true,
}

func jour(annee int, mois time.Month, quantieme int) time.Time {
	return time.Date(annee, mois, quantieme, 0, 0, 0, 0, time.UTC)
}

func vendeurInscritIlYA(jours int) regles.Seller {
	return regles.Seller{MemberSince: dateDeSoumission.AddDate(0, 0, -jours).Truncate(24 * time.Hour)}
}

func vendeurSuspendu() regles.Seller {
	vendeur := vendeurEnRegle()
	vendeur.Suspended = true
	return vendeur
}

func annonceAuPrix(categorie string, centimes int64) regles.Listing {
	annonce := annonceComplete()
	annonce.CategoryCode = code(categorie)
	annonce.PriceCents = prix(centimes)
	return annonce
}

func evaluerPour(vendeur regles.Seller, annonce regles.Listing) regles.Verdict {
	return regles.Reference().Evaluate(regles.Submission{SubmittedAt: dateDeSoumission, Seller: vendeur, Listing: annonce})
}

func TestUS019AC1PrixHorsPlageAVerifierPrixAConfirmer(t *testing.T) {
	for _, s := range seuilsDuReadme {
		t.Run(s.categorie, func(t *testing.T) {
			for nom, centimes := range map[string]int64{"un centime sous le minimum": s.minimum - 1, "un centime au-dessus du maximum": s.maximum + 1, "prix nul": 0, "prix très élevé": 100_000_000} {
				t.Run(nom, func(t *testing.T) {
					exigerVerdict(t, evaluer(annonceAuPrix(s.categorie, centimes)), regles.IssueToVerify, regles.ReasonPriceToConfirm)
				})
			}
		})
	}
}

func TestUS019AC1AC5BornesDeLaPlageIncluses(t *testing.T) {
	for _, s := range seuilsDuReadme {
		t.Run(s.categorie, func(t *testing.T) {
			for nom, centimes := range map[string]int64{"minimum": s.minimum, "maximum": s.maximum} {
				t.Run(nom, func(t *testing.T) {
					exigerVerdict(t, evaluer(annonceAuPrix(s.categorie, centimes)), regles.IssuePublished, "")
				})
			}
		})
	}
}

func TestUS019AC1PrixHorsPlageAVerifierMemePourUnVendeurRecent(t *testing.T) {
	exigerVerdict(t, evaluerPour(vendeurInscritIlYA(3), annonceAuPrix("figurines", 99)), regles.IssueToVerify, regles.ReasonPriceToConfirm)
}

func TestUS019AC2VendeurSuspenduRefuseQuelQueSoitLeReste(t *testing.T) {
	cas := map[string]regles.Listing{
		"annonce irréprochable":           annonceComplete(),
		"annonce vide":                    {},
		"annonce sans description":        {Title: "Figurine", CategoryCode: code("figurines"), PriceCents: prix(4500)},
		"coordonnées dans la description": {Title: "Figurine", Description: "Appelez le 01 99 00 12 34", CategoryCode: code("figurines"), PriceCents: prix(4500)},
		"catégorie inconnue":              annonceAuPrix("timbres", 4500),
		"prix hors plage":                 annonceAuPrix("figurines", 300_001),
		"prix au-dessus de la vigilance":  annonceAuPrix("figurines", 30_001),
	}
	for nom, annonce := range cas {
		t.Run(nom, func(t *testing.T) {
			exigerVerdict(t, evaluerPour(vendeurSuspendu(), annonce), regles.IssueRefused, regles.ReasonUnauthorizedAccount)
		})
	}
}

func TestUS019AC2VendeurRecentEtSuspenduRefuse(t *testing.T) {
	vendeur := vendeurInscritIlYA(0)
	vendeur.Suspended = true
	exigerVerdict(t, evaluerPour(vendeur, annonceAuPrix("montres", 300_000)), regles.IssueRefused, regles.ReasonUnauthorizedAccount)
}

func TestUS019AC3VendeurRecentAuDessusDeLaVigilanceAVerifier(t *testing.T) {
	for _, s := range seuilsDuReadme {
		t.Run(s.categorie, func(t *testing.T) {
			exigerVerdict(t, evaluerPour(vendeurInscritIlYA(10), annonceAuPrix(s.categorie, s.vigilance+1)), regles.IssueToVerify, regles.ReasonSellerToVerify)
		})
	}
}

func TestUS019AC3VendeurRecentAuSeuilDeVigilanceOuEnDessousPublie(t *testing.T) {
	for _, s := range seuilsDuReadme {
		t.Run(s.categorie, func(t *testing.T) {
			for nom, centimes := range map[string]int64{"au seuil": s.vigilance, "un centime sous le seuil": s.vigilance - 1} {
				t.Run(nom, func(t *testing.T) {
					exigerVerdict(t, evaluerPour(vendeurInscritIlYA(10), annonceAuPrix(s.categorie, centimes)), regles.IssuePublished, "")
				})
			}
		})
	}
}

func TestUS019AC3VendeurAncienAuDessusDeLaVigilancePublie(t *testing.T) {
	for _, s := range seuilsDuReadme {
		t.Run(s.categorie, func(t *testing.T) {
			exigerVerdict(t, evaluerPour(vendeurInscritIlYA(30), annonceAuPrix(s.categorie, s.vigilance+1)), regles.IssuePublished, "")
		})
	}
}

func TestUS019AC3AncienneteEnJoursEntiersUTCAutourDe30Jours(t *testing.T) {
	inscription := jour(2026, 10, 3)
	paris := time.FixedZone("Europe/Paris", 2*3600)
	cas := []struct {
		nom    string
		soumis time.Time
		issue  regles.Issue
		motif  regles.Reason
	}{
		{"29 jours à la dernière seconde", time.Date(2026, 11, 1, 23, 59, 59, 0, time.UTC), regles.IssueToVerify, regles.ReasonSellerToVerify},
		{"30 jours à minuit", time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), regles.IssuePublished, ""},
		{"31 jours", time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC), regles.IssuePublished, ""},
		{"30 jours en heure de Paris mais 29 en UTC", time.Date(2026, 11, 2, 1, 30, 0, 0, paris), regles.IssueToVerify, regles.ReasonSellerToVerify},
		{"inscrit le jour même", time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC), regles.IssueToVerify, regles.ReasonSellerToVerify},
		{"inscription postérieure à la soumission", time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC), regles.IssueToVerify, regles.ReasonSellerToVerify},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			soumission := regles.Submission{SubmittedAt: c.soumis, Seller: regles.Seller{MemberSince: inscription}, Listing: annonceAuPrix("figurines", 30_001)}
			exigerVerdict(t, regles.Reference().Evaluate(soumission), c.issue, c.motif)
		})
	}
}

func TestUS019AC3SeuilDAncienneteConfigure(t *testing.T) {
	regle := regles.New(regles.Catalogue{"figurines": {MinCents: 100, MaxCents: 300_000, VigilanceCents: 30_000}}, regles.SeniorityPolicy{MinimumDays: 7})
	for jours, attendu := range map[int]regles.Issue{6: regles.IssueToVerify, 7: regles.IssuePublished} {
		soumission := regles.Submission{SubmittedAt: dateDeSoumission, Seller: vendeurInscritIlYA(jours), Listing: annonceAuPrix("figurines", 30_001)}
		if verdict := regle.Evaluate(soumission); verdict.Issue != attendu {
			t.Errorf("ancienneté %d jours : %+v, attendu %q", jours, verdict, attendu)
		}
	}
}

func TestUS019OrdreDesReglesQuandPlusieursSAppliquent(t *testing.T) {
	cas := []struct {
		nom     string
		vendeur regles.Seller
		annonce regles.Listing
		issue   regles.Issue
		motif   regles.Reason
	}{
		{"incomplète et prix hors plage", vendeurInscritIlYA(2), regles.Listing{Title: "Figurine", CategoryCode: code("figurines"), PriceCents: prix(1)}, regles.IssueRefused, regles.ReasonIncompleteListing},
		{"coordonnées et prix hors plage", vendeurInscritIlYA(2), regles.Listing{Title: "Figurine", Description: "Écrivez à jean.dupont@example.com", CategoryCode: code("figurines"), PriceCents: prix(1)}, regles.IssueRefused, regles.ReasonContactDetected},
		{"coordonnées et vendeur récent au-dessus de la vigilance", vendeurInscritIlYA(2), regles.Listing{Title: "Figurine", Description: "Écrivez à jean.dupont@example.com", CategoryCode: code("figurines"), PriceCents: prix(30_001)}, regles.IssueRefused, regles.ReasonContactDetected},
		{"catégorie inconnue et prix extrême", vendeurInscritIlYA(2), annonceAuPrix("timbres", 1), regles.IssueToVerify, regles.ReasonUnknownCategory},
		{"prix hors plage et vendeur récent au-dessus de la vigilance", vendeurInscritIlYA(2), annonceAuPrix("figurines", 300_001), regles.IssueToVerify, regles.ReasonPriceToConfirm},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			exigerVerdict(t, evaluerPour(c.vendeur, c.annonce), c.issue, c.motif)
		})
	}
}

func TestUS019AC5AnnonceEnRegleDansChaqueCategoriePubliee(t *testing.T) {
	for _, s := range seuilsDuReadme {
		t.Run(s.categorie, func(t *testing.T) {
			exigerVerdict(t, evaluer(annonceAuPrix(s.categorie, (s.minimum+s.maximum)/2)), regles.IssuePublished, "")
		})
	}
}

func TestUS019VerdictNeDivulgueAucunSeuil(t *testing.T) {
	champs := reflect.VisibleFields(reflect.TypeOf(regles.Verdict{}))
	if len(champs) != 2 || champs[0].Name != "Issue" || champs[1].Name != "Reason" {
		t.Fatalf("le verdict expose d'autres champs que l'issue et le motif : %v", champs)
	}
	for _, motif := range []regles.Reason{regles.ReasonUnauthorizedAccount, regles.ReasonPriceToConfirm, regles.ReasonSellerToVerify} {
		if !motifsDuContrat[motif] {
			t.Errorf("motif %q hors de la liste fermée du contrat", motif)
		}
	}
}

func TestUS019AC4EvaluationPureSansHorloge(t *testing.T) {
	historique := regles.Submission{SubmittedAt: time.Date(2020, 1, 10, 10, 0, 0, 0, time.UTC), Seller: regles.Seller{MemberSince: jour(2020, 1, 1)}, Listing: annonceAuPrix("figurines", 30_001)}
	exigerVerdict(t, regles.Reference().Evaluate(historique), regles.IssueToVerify, regles.ReasonSellerToVerify)
	futur := regles.Submission{SubmittedAt: time.Date(2040, 1, 10, 10, 0, 0, 0, time.UTC), Seller: regles.Seller{MemberSince: jour(2039, 11, 1)}, Listing: annonceAuPrix("figurines", 30_001)}
	exigerVerdict(t, regles.Reference().Evaluate(futur), regles.IssuePublished, "")
	premier := regles.Reference().Evaluate(historique)
	for range 1000 {
		if verdict := regles.Reference().Evaluate(historique); verdict != premier {
			t.Fatalf("même entrée, sorties différentes : %+v puis %+v", premier, verdict)
		}
	}
}

func TestUS019SeuilsFigesALaConstruction(t *testing.T) {
	catalogue := regles.Catalogue{"figurines": {MinCents: 100, MaxCents: 300_000, VigilanceCents: 30_000}}
	regle := regles.New(catalogue, politiqueDeReference)
	soumission := soumissionDe(annonceAuPrix("figurines", 4500))
	avant := regle.Evaluate(soumission)
	catalogue["figurines"] = regles.PriceRange{MinCents: 10_000, MaxCents: 20_000, VigilanceCents: 15_000}
	if apres := regle.Evaluate(soumission); apres != avant {
		t.Fatalf("modifier le catalogue après construction change l'issue : %+v puis %+v", avant, apres)
	}
}
