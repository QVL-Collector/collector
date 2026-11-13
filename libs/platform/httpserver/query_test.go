package httpserver_test

import (
	"net/url"
	"slices"
	"testing"

	"collector/libs/platform/httpserver"
	"collector/libs/platform/problem"
)

var limitesDesNotifications = httpserver.PageLimits{DefaultSize: 20, MaxSize: 50, MaxNumber: 10_000}

func requete(t *testing.T, brute string) url.Values {
	t.Helper()
	valeurs, err := url.ParseQuery(brute)
	if err != nil {
		t.Fatalf("requête %q : %v", brute, err)
	}
	return valeurs
}

func exigerErreurDeChamp(t *testing.T, erreurs []problem.FieldError, champ string, codes ...string) {
	t.Helper()
	if len(erreurs) != 1 || erreurs[0].Field != champ || !slices.Contains(codes, erreurs[0].Code) {
		t.Fatalf("erreurs %+v, une erreur sur %q avec un code parmi %v attendue", erreurs, champ, codes)
	}
}

func TestUS028PaginationParDefaut(t *testing.T) {
	pagination, erreurs := httpserver.PaginationOf(requete(t, ""), limitesDesNotifications)
	if len(erreurs) != 0 || pagination != (httpserver.Pagination{Number: 1, Size: 20}) {
		t.Fatalf("pagination %+v, erreurs %+v ; page 1 de 20 attendue", pagination, erreurs)
	}
}

func TestUS028PaginationAuxBornes(t *testing.T) {
	cas := map[string]httpserver.Pagination{
		"page=1&taille=1":      {Number: 1, Size: 1},
		"page=10000&taille=50": {Number: 10_000, Size: 50},
		"page=3":               {Number: 3, Size: 20},
		"taille=7":             {Number: 1, Size: 7},
		"page=2&autre=x":       {Number: 2, Size: 20},
	}
	for brute, attendue := range cas {
		t.Run(brute, func(t *testing.T) {
			pagination, erreurs := httpserver.PaginationOf(requete(t, brute), limitesDesNotifications)
			if len(erreurs) != 0 || pagination != attendue {
				t.Fatalf("pagination %+v, erreurs %+v ; %+v attendue", pagination, erreurs, attendue)
			}
		})
	}
}

func TestUS028PaginationHorsBornesRefusee(t *testing.T) {
	cas := map[string]string{
		"page=0":       "page",
		"page=-1":      "page",
		"page=10001":   "page",
		"taille=0":     "taille",
		"taille=51":    "taille",
		"taille=-5":    "taille",
		"taille=99999": "taille",
	}
	for brute, champ := range cas {
		t.Run(brute, func(t *testing.T) {
			_, erreurs := httpserver.PaginationOf(requete(t, brute), limitesDesNotifications)
			exigerErreurDeChamp(t, erreurs, champ, problem.CodeOutOfBounds)
		})
	}
}

func TestUS028PaginationIllisibleRefusee(t *testing.T) {
	cas := map[string]string{
		"page=un":                  "page",
		"page=1.5":                 "page",
		"page=":                    "page",
		"page=1&page=2":            "page",
		"taille=vingt":             "taille",
		"taille=":                  "taille",
		"taille=10&taille=20":      "taille",
		"taille=99999999999999999": "taille",
	}
	for brute, champ := range cas {
		t.Run(brute, func(t *testing.T) {
			_, erreurs := httpserver.PaginationOf(requete(t, brute), limitesDesNotifications)
			exigerErreurDeChamp(t, erreurs, champ, problem.CodeInvalid, problem.CodeOutOfBounds)
		})
	}
}

func TestUS028PaginationSignaleLesDeuxParametres(t *testing.T) {
	_, erreurs := httpserver.PaginationOf(requete(t, "page=0&taille=abc"), limitesDesNotifications)
	var champs []string
	for _, e := range erreurs {
		champs = append(champs, e.Field)
	}
	slices.Sort(champs)
	if !slices.Equal(champs, []string{"page", "taille"}) {
		t.Fatalf("erreurs %+v, une par paramètre attendue", erreurs)
	}
}

func TestUS028ParametreBorneAbsentPrendLaValeurParDefaut(t *testing.T) {
	valeur, erreurs := httpserver.BoundedQueryInt(requete(t, "autre=3"), "limite", 7, 9)
	if valeur != 7 || len(erreurs) != 0 {
		t.Fatalf("valeur %d, erreurs %+v ; 7 sans erreur attendu", valeur, erreurs)
	}
}

func TestUS028ParametreBorneAccepteDeUnAuMaximum(t *testing.T) {
	for brute, attendue := range map[string]int{"limite=1": 1, "limite=9": 9, "limite=5": 5} {
		t.Run(brute, func(t *testing.T) {
			valeur, erreurs := httpserver.BoundedQueryInt(requete(t, brute), "limite", 7, 9)
			if valeur != attendue || len(erreurs) != 0 {
				t.Fatalf("valeur %d, erreurs %+v ; %d attendu", valeur, erreurs, attendue)
			}
		})
	}
}

func TestUS028ParametreBorneHorsBornesRefuse(t *testing.T) {
	for _, brute := range []string{"limite=0", "limite=10"} {
		t.Run(brute, func(t *testing.T) {
			_, erreurs := httpserver.BoundedQueryInt(requete(t, brute), "limite", 7, 9)
			exigerErreurDeChamp(t, erreurs, "limite", problem.CodeOutOfBounds)
		})
	}
}

func TestUS028ParametreBorneIllisibleRefuse(t *testing.T) {
	for _, brute := range []string{"limite=x", "limite=", "limite=1&limite=2", "limite=0x5", "limite=3.0"} {
		t.Run(brute, func(t *testing.T) {
			_, erreurs := httpserver.BoundedQueryInt(requete(t, brute), "limite", 7, 9)
			exigerErreurDeChamp(t, erreurs, "limite", problem.CodeInvalid, problem.CodeOutOfBounds)
		})
	}
}

func TestUS028ParametreUniqueAbsentPresentOuRepete(t *testing.T) {
	valeur, present, erreurs := httpserver.SingleQueryParameter(requete(t, "autre=1"), "categorie")
	if valeur != "" || present || len(erreurs) != 0 {
		t.Fatalf("absent : %q %v %+v", valeur, present, erreurs)
	}
	valeur, present, erreurs = httpserver.SingleQueryParameter(requete(t, "categorie=figurines"), "categorie")
	if valeur != "figurines" || !present || len(erreurs) != 0 {
		t.Fatalf("présent : %q %v %+v", valeur, present, erreurs)
	}
	_, _, erreurs = httpserver.SingleQueryParameter(requete(t, "categorie=figurines&categorie=timbres"), "categorie")
	exigerErreurDeChamp(t, erreurs, "categorie", problem.CodeInvalid)
}
