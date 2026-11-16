package config_test

import (
	"strings"
	"testing"
	"time"

	"collector/services/place-de-marche/internal/config"
)

func environnementComplet(duree, plafond string) config.Lookup {
	valeurs := map[string]string{
		"BASE_DE_DONNEES_URL":       "postgres://marche_execution@127.0.0.1:5432/marche?sslmode=disable",
		"BASE_DE_DONNEES_TLS_EXIGE": "false",
		"OIDC_EMETTEUR":             "http://127.0.0.1:8180/realms/collector",
		"CANAL_HOTE":                "127.0.0.1",
		"CANAL_UTILISATEUR":         "place-de-marche",
		"CANAL_MOT_DE_PASSE":        "secret-du-canal",
		"RESERVATION_DUREE":         duree,
		"RESERVATION_DUREE_MAX":     plafond,
	}
	return func(nom string) string { return valeurs[nom] }
}

func TestUS031DureesDeReservationAcceptees(t *testing.T) {
	cas := []struct {
		duree, plafond      string
		attendue            time.Duration
		demandee, plafondLu time.Duration
	}{
		{"1s", "1s", time.Second, time.Second, time.Second},
		{"1.5s", "30m", 1500 * time.Millisecond, 1500 * time.Millisecond, 30 * time.Minute},
		{"6s", "6s", 6 * time.Second, 6 * time.Second, 6 * time.Second},
		{"45m", "20m", 20 * time.Minute, 45 * time.Minute, 20 * time.Minute},
		{"", "", 10 * time.Minute, 10 * time.Minute, 30 * time.Minute},
	}
	for _, c := range cas {
		t.Run(c.duree+"/"+c.plafond, func(t *testing.T) {
			serveur, err := config.LoadServer(environnementComplet(c.duree, c.plafond))
			if err != nil {
				t.Fatalf("RESERVATION_DUREE=%q RESERVATION_DUREE_MAX=%q refusées : %v", c.duree, c.plafond, err)
			}
			terme := serveur.Reservation.Term
			if terme.Requested != c.demandee || terme.Ceiling != c.plafondLu || terme.Duration() != c.attendue {
				t.Fatalf("terme %+v (durée %s), attendu %s / %s → %s", terme, terme.Duration(), c.demandee, c.plafondLu, c.attendue)
			}
		})
	}
}

func TestUS031DureesDeReservationRefusees(t *testing.T) {
	cas := []struct{ variable, duree, plafond string }{
		{"RESERVATION_DUREE", "999ms", "30m"},
		{"RESERVATION_DUREE", "500ms", "30m"},
		{"RESERVATION_DUREE", "1ns", "30m"},
		{"RESERVATION_DUREE", "0s", "30m"},
		{"RESERVATION_DUREE", "-5s", "30m"},
		{"RESERVATION_DUREE", "cinq", "30m"},
		{"RESERVATION_DUREE_MAX", "10m", "999ms"},
		{"RESERVATION_DUREE_MAX", "10m", "0s"},
		{"RESERVATION_DUREE_MAX", "10m", "trente"},
	}
	for _, c := range cas {
		t.Run(c.variable+"="+c.duree+"/"+c.plafond, func(t *testing.T) {
			_, err := config.LoadServer(environnementComplet(c.duree, c.plafond))
			if err == nil {
				t.Fatalf("RESERVATION_DUREE=%q RESERVATION_DUREE_MAX=%q acceptées", c.duree, c.plafond)
			}
			if !strings.Contains(err.Error(), c.variable) {
				t.Errorf("refus sans nommer %s : %v", c.variable, err)
			}
		})
	}
}
