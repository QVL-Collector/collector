//go:build !race

package coordonnees_test

import (
	"slices"
	"testing"
	"time"

	"collector/services/controle/internal/coordonnees"
)

const (
	budgetParAnnonce = 50 * time.Millisecond
	mesuresParTexte  = 7
)

func meilleureDuree(texte string) time.Duration {
	durees := make([]time.Duration, mesuresParTexte)
	for i := range durees {
		debut := time.Now()
		coordonnees.Detected(texte)
		durees[i] = time.Since(debut)
	}
	return slices.Min(durees)
}

func TestUS018AC6TexteMaximalPiegeAnalyseEnMoinsDe50ms(t *testing.T) {
	titres := textesPiegesDeLongueur(longueurMaxTitre)
	for nom, description := range textesPiegesDeLongueur(longueurMaxDescription) {
		t.Run(nom, func(t *testing.T) {
			duree := meilleureDuree(titres[nom]) + meilleureDuree(description)
			t.Logf("annonce piégée %q : %s", nom, duree)
			if duree >= budgetParAnnonce {
				t.Errorf("détection en %s, budget %s", duree, budgetParAnnonce)
			}
		})
	}
}
