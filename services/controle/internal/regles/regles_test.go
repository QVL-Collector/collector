package regles_test

import (
	"testing"
	"time"

	"collector/services/controle/internal/regles"
)

func code(valeur string) *regles.CategoryCode {
	c := regles.CategoryCode(valeur)
	return &c
}

func prix(centimes int64) *int64 {
	return &centimes
}

func annonceComplete() regles.Listing {
	return regles.Listing{
		Title:        "Figurine Tintin",
		Description:  "Belle pièce en bon état",
		CategoryCode: code("figurines"),
		PriceCents:   prix(4500),
	}
}

var politiqueDeReference = regles.SeniorityPolicy{MinimumDays: 30}

var dateDeSoumission = time.Date(2026, 11, 2, 9, 14, 59, 0, time.UTC)

func vendeurEnRegle() regles.Seller {
	return regles.Seller{MemberSince: time.Date(2025, 3, 14, 0, 0, 0, 0, time.UTC)}
}

func soumissionDe(annonce regles.Listing) regles.Submission {
	return regles.Submission{SubmittedAt: dateDeSoumission, Seller: vendeurEnRegle(), Listing: annonce}
}

func evaluer(annonce regles.Listing) regles.Verdict {
	return regles.Reference().Evaluate(soumissionDe(annonce))
}

func exigerVerdict(t *testing.T, recu regles.Verdict, issue regles.Issue, motif regles.Reason) {
	t.Helper()
	if recu.Issue != issue || recu.Reason != motif {
		t.Fatalf("verdict %+v, attendu issue %q motif %q", recu, issue, motif)
	}
	if recu.HasReason() != (motif != "") {
		t.Fatalf("présence du motif incohérente : %+v", recu)
	}
}

func TestUS013AC5US018AC5US019VersionDesReglesEgaleA3(t *testing.T) {
	if regles.RulesVersion != 3 {
		t.Fatalf("version des règles %d, attendue 3", regles.RulesVersion)
	}
}

func TestUS013AC1AnnonceCompleteDansUneCategorieDeReferencePubliee(t *testing.T) {
	for _, categorie := range []string{"bandes-dessinees", "cartes-a-collectionner", "figurines", "jouets-anciens", "monnaies", "montres", "vinyles"} {
		t.Run(categorie, func(t *testing.T) {
			annonce := annonceComplete()
			annonce.CategoryCode = code(categorie)
			exigerVerdict(t, evaluer(annonce), regles.IssuePublished, "")
		})
	}
}

func TestUS013AC1TexteMinimalSuffit(t *testing.T) {
	annonce := regles.Listing{Title: "a", Description: "b", CategoryCode: code("figurines"), PriceCents: prix(4500)}
	exigerVerdict(t, evaluer(annonce), regles.IssuePublished, "")
}

func TestUS013AC2AnnonceIncompleteRefusee(t *testing.T) {
	cas := map[string]func(*regles.Listing){
		"sans description":       func(a *regles.Listing) { a.Description = "" },
		"description en espaces": func(a *regles.Listing) { a.Description = " \t\n " },
		"sans titre":             func(a *regles.Listing) { a.Title = "" },
		"titre en espaces":       func(a *regles.Listing) { a.Title = "   " },
		"sans catégorie":         func(a *regles.Listing) { a.CategoryCode = nil },
		"catégorie vide":         func(a *regles.Listing) { a.CategoryCode = code("") },
		"sans prix":              func(a *regles.Listing) { a.PriceCents = nil },
		"tout absent":            func(a *regles.Listing) { *a = regles.Listing{} },
	}
	for nom, alterer := range cas {
		t.Run(nom, func(t *testing.T) {
			annonce := annonceComplete()
			alterer(&annonce)
			exigerVerdict(t, evaluer(annonce), regles.IssueRefused, regles.ReasonIncompleteListing)
		})
	}
}

func TestUS013AC4CategorieInconnueAVerifierJamaisPubliee(t *testing.T) {
	for _, categorie := range []string{"timbres", "Figurines", "figurines-anciennes", "x"} {
		t.Run(categorie, func(t *testing.T) {
			annonce := annonceComplete()
			annonce.CategoryCode = code(categorie)
			exigerVerdict(t, evaluer(annonce), regles.IssueToVerify, regles.ReasonUnknownCategory)
		})
	}
}

func TestUS013AC4ReferentielVideNePublieRien(t *testing.T) {
	exigerVerdict(t, regles.New(nil, politiqueDeReference).Evaluate(soumissionDe(annonceComplete())), regles.IssueToVerify, regles.ReasonUnknownCategory)
}

func TestUS013AC4ReferentielFourniEstLeSeulConnu(t *testing.T) {
	regle := regles.New(regles.Catalogue{"timbres": {MinCents: 100, MaxCents: 100_000, VigilanceCents: 10_000}}, politiqueDeReference)
	annonce := annonceComplete()
	exigerVerdict(t, regle.Evaluate(soumissionDe(annonce)), regles.IssueToVerify, regles.ReasonUnknownCategory)
	annonce.CategoryCode = code("timbres")
	exigerVerdict(t, regle.Evaluate(soumissionDe(annonce)), regles.IssuePublished, "")
}

func TestUS013AC4AnnonceIncompleteEtCategorieInconnueJamaisPubliee(t *testing.T) {
	annonce := annonceComplete()
	annonce.Description = ""
	annonce.CategoryCode = code("timbres")
	if verdict := evaluer(annonce); verdict.Issue == regles.IssuePublished || !verdict.HasReason() {
		t.Fatalf("verdict %+v : publié ou sans motif", verdict)
	}
}
