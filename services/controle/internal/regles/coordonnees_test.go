package regles_test

import (
	"testing"

	"collector/services/controle/internal/regles"
)

var textesAvecCoordonnees = map[string]string{
	"email en clair":          "Écrivez à jean.dupont@example.com",
	"email masqué":            "jean point dupont arobase example point com",
	"lien externe":            "Photos sur https://example.com/lot",
	"téléphone en clair":      "Appelez le 01 99 00 12 34",
	"téléphone international": "+33 1 99 00 12 34",
	"téléphone en lettres":    "zéro six trente-neuf quatre-vingt-dix-huit douze trente-quatre",
	"chiffres espacés":        "0 1 9 9 0 0 1 2 3 4",
}

var textesSansCoordonneesRealistes = map[string]string{
	"dimensions":      "Affiche 30 x 40 cm, édition 1987",
	"date":            "Achetée le 12/05/1987, conservée à l'abri",
	"référence":       "ISBN 978-2-203-00101-8, dépôt légal 1946",
	"numéro de série": "Numéro de série : 45781236",
	"prix":            "Valeur estimée 1 250,00 €",
	"tome":            "Tintin tome 12",
	"phrase collée":   "Bon état.De plus la boîte est d'origine",
}

func TestUS018AC1CoordonneesDansLeTitreRefusees(t *testing.T) {
	for nom, texte := range textesAvecCoordonnees {
		t.Run(nom, func(t *testing.T) {
			annonce := annonceComplete()
			annonce.Title = texte
			exigerVerdict(t, regles.Reference().Evaluate(annonce), regles.IssueRefused, regles.ReasonContactDetected)
		})
	}
}

func TestUS018AC1CoordonneesDansLaDescriptionRefusees(t *testing.T) {
	for nom, texte := range textesAvecCoordonnees {
		t.Run(nom, func(t *testing.T) {
			annonce := annonceComplete()
			annonce.Description = "Belle pièce en bon état.\n" + texte + "\nEnvoi soigné."
			exigerVerdict(t, regles.Reference().Evaluate(annonce), regles.IssueRefused, regles.ReasonContactDetected)
		})
	}
}

func TestUS018AC1MotifPublieEstLaCategorieFermee(t *testing.T) {
	if regles.ReasonContactDetected != "coordonnees_detectees" {
		t.Fatalf("motif %q, attendu coordonnees_detectees", regles.ReasonContactDetected)
	}
}

func TestUS018AC3TexteSansCoordonneesPublie(t *testing.T) {
	for nom, texte := range textesSansCoordonneesRealistes {
		t.Run(nom, func(t *testing.T) {
			annonce := annonceComplete()
			annonce.Title = texte
			annonce.Description = texte
			exigerVerdict(t, regles.Reference().Evaluate(annonce), regles.IssuePublished, "")
		})
	}
}

func TestUS018AC4CoordonneesRetireesPuisResoumisPublie(t *testing.T) {
	annonce := annonceComplete()
	annonce.Description = "Contact : 06 39 98 12 34"
	exigerVerdict(t, regles.Reference().Evaluate(annonce), regles.IssueRefused, regles.ReasonContactDetected)
	annonce.Description = "Belle pièce en bon état"
	exigerVerdict(t, regles.Reference().Evaluate(annonce), regles.IssuePublished, "")
}

func TestUS018AnnonceIncompletePrimeSurCoordonnees(t *testing.T) {
	annonce := annonceComplete()
	annonce.Title = "jean.dupont@example.com"
	annonce.Description = ""
	exigerVerdict(t, regles.Reference().Evaluate(annonce), regles.IssueRefused, regles.ReasonIncompleteListing)
}

func TestUS018CoordonneesPrimentSurCategorieInconnue(t *testing.T) {
	annonce := annonceComplete()
	annonce.Description = "Appelez le 01 99 00 12 34"
	annonce.CategoryCode = code("timbres")
	exigerVerdict(t, regles.Reference().Evaluate(annonce), regles.IssueRefused, regles.ReasonContactDetected)
}
