package regles_test

import (
	"fmt"
	"testing"

	"collector/services/controle/internal/regles"
)

func annonceAvecPhotos(nombre int) regles.Listing {
	annonce := annonceComplete()
	annonce.PhotoCount = nombre
	return annonce
}

func TestUS020AC2AnnonceSansPhotoRefuseeIncompleteDansChaqueCategorie(t *testing.T) {
	for _, s := range seuilsDuReadme {
		t.Run(s.categorie, func(t *testing.T) {
			annonce := annonceAuPrix(s.categorie, (s.minimum+s.maximum)/2)
			annonce.PhotoCount = 0
			exigerVerdict(t, evaluer(annonce), regles.IssueRefused, regles.ReasonIncompleteListing)
		})
	}
}

func TestUS020AC2AnnonceAvecPhotosPubliee(t *testing.T) {
	for _, nombre := range []int{1, 2, 10} {
		t.Run(fmt.Sprint(nombre), func(t *testing.T) {
			exigerVerdict(t, evaluer(annonceAvecPhotos(nombre)), regles.IssuePublished, "")
		})
	}
}

func TestUS020AC2AbsenceDePhotoPrimeSurLesReglesDeVerification(t *testing.T) {
	cas := map[string]regles.Listing{
		"catégorie inconnue":             annonceAuPrix("timbres", 4500),
		"prix hors plage":                annonceAuPrix("figurines", 300_001),
		"prix au-dessus de la vigilance": annonceAuPrix("figurines", 30_001),
	}
	for nom, annonce := range cas {
		t.Run(nom, func(t *testing.T) {
			annonce.PhotoCount = 0
			exigerVerdict(t, evaluerPour(vendeurInscritIlYA(2), annonce), regles.IssueRefused, regles.ReasonIncompleteListing)
		})
	}
}

func TestUS020AC2VendeurSuspenduResteRefuseCompteNonAutoriseSansPhoto(t *testing.T) {
	exigerVerdict(t, evaluerPour(vendeurSuspendu(), annonceAvecPhotos(0)), regles.IssueRefused, regles.ReasonUnauthorizedAccount)
}
