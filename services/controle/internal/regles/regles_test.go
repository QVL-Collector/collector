package regles_test

import (
	"testing"

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

func exigerVerdict(t *testing.T, recu regles.Verdict, issue regles.Issue, motif regles.Reason) {
	t.Helper()
	if recu.Issue != issue || recu.Reason != motif {
		t.Fatalf("verdict %+v, attendu issue %q motif %q", recu, issue, motif)
	}
	if recu.HasReason() != (motif != "") {
		t.Fatalf("présence du motif incohérente : %+v", recu)
	}
}

func TestUS013AC5US018AC5VersionDesReglesEgaleA2(t *testing.T) {
	if regles.RulesVersion != 2 {
		t.Fatalf("version des règles %d, attendue 2", regles.RulesVersion)
	}
}

func TestUS013AC1AnnonceCompleteDansUneCategorieDeReferencePubliee(t *testing.T) {
	for _, categorie := range []string{"bandes-dessinees", "cartes-a-collectionner", "figurines", "jouets-anciens", "monnaies", "montres", "vinyles"} {
		t.Run(categorie, func(t *testing.T) {
			annonce := annonceComplete()
			annonce.CategoryCode = code(categorie)
			exigerVerdict(t, regles.Reference().Evaluate(annonce), regles.IssuePublished, "")
		})
	}
}

func TestUS013AC1TexteMinimalSuffit(t *testing.T) {
	annonce := regles.Listing{Title: "a", Description: "b", CategoryCode: code("figurines"), PriceCents: prix(1)}
	exigerVerdict(t, regles.Reference().Evaluate(annonce), regles.IssuePublished, "")
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
			exigerVerdict(t, regles.Reference().Evaluate(annonce), regles.IssueRefused, regles.ReasonIncompleteListing)
		})
	}
}

func TestUS013AC4CategorieInconnueAVerifierJamaisPubliee(t *testing.T) {
	for _, categorie := range []string{"timbres", "Figurines", "figurines-anciennes", "x"} {
		t.Run(categorie, func(t *testing.T) {
			annonce := annonceComplete()
			annonce.CategoryCode = code(categorie)
			exigerVerdict(t, regles.Reference().Evaluate(annonce), regles.IssueToVerify, regles.ReasonUnknownCategory)
		})
	}
}

func TestUS013AC4ReferentielVideNePublieRien(t *testing.T) {
	exigerVerdict(t, regles.New(nil).Evaluate(annonceComplete()), regles.IssueToVerify, regles.ReasonUnknownCategory)
}

func TestUS013AC4ReferentielFourniEstLeSeulConnu(t *testing.T) {
	regle := regles.New([]regles.CategoryCode{"timbres"})
	annonce := annonceComplete()
	exigerVerdict(t, regle.Evaluate(annonce), regles.IssueToVerify, regles.ReasonUnknownCategory)
	annonce.CategoryCode = code("timbres")
	exigerVerdict(t, regle.Evaluate(annonce), regles.IssuePublished, "")
}

func TestUS013AC4AnnonceIncompleteEtCategorieInconnueJamaisPubliee(t *testing.T) {
	annonce := annonceComplete()
	annonce.Description = ""
	annonce.CategoryCode = code("timbres")
	if verdict := regles.Reference().Evaluate(annonce); verdict.Issue == regles.IssuePublished || !verdict.HasReason() {
		t.Fatalf("verdict %+v : publié ou sans motif", verdict)
	}
}
