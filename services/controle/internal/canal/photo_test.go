package canal

import "testing"

func TestUS020AC2DemandeDuContratSansPhotoRefuseeIncomplete(t *testing.T) {
	corps := demandeAvec(t, exempleDeDemande(t, "annonce-complete.json"), func(d map[string]any) {
		d["listing"].(map[string]any)["photo_count"] = 0
	})
	resultat, _ := controler(t, corps)
	exigerResultat(t, resultat, "refuse", "annonce_incomplete")
}

func TestUS020AC2DemandeDuContratAvecUnePhotoPubliee(t *testing.T) {
	corps := demandeAvec(t, exempleDeDemande(t, "annonce-complete.json"), func(d map[string]any) {
		d["listing"].(map[string]any)["photo_count"] = 1
	})
	resultat, _ := controler(t, corps)
	exigerResultat(t, resultat, "publie", "")
}
