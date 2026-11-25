package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

const delaiDeRefusAuDemarrage = 5 * time.Second

func environnementDeDemarrageSansDependance(t *testing.T) map[string]string {
	t.Helper()
	return map[string]string{
		"BASE_DE_DONNEES_URL":       "postgres://marche_execution:secret@" + adresseLibre(t) + "/marche?sslmode=disable&connect_timeout=1",
		"BASE_DE_DONNEES_TLS_EXIGE": "false",
		"OIDC_EMETTEUR":             "http://" + adresseLibre(t) + "/realms/collector",
		"CANAL_HOTE":                "127.0.0.1",
		"CANAL_PORT":                strings.Split(adresseLibre(t), ":")[1],
		"CANAL_UTILISATEUR":         identiteMarche,
		"CANAL_MOT_DE_PASSE":        "secret-du-canal",
		"ADRESSE_API":               adresseLibre(t),
		"ADRESSE_INTERNE":           adresseLibre(t),
		"ADRESSE_GESTION":           adresseLibre(t),
		"ADRESSE_METRIQUES":         adresseLibre(t),
		"RESERVATION_DUREE":         "10m",
		"RESERVATION_DUREE_MAX":     "30m",
	}
}

func demarrerAvecLaConfiguration(t *testing.T, variables map[string]string) (bool, error) {
	t.Helper()
	for nom, valeur := range variables {
		t.Setenv(nom, valeur)
	}
	ctx, arreter := context.WithTimeout(context.Background(), 3*delaiDeRefusAuDemarrage)
	defer arreter()
	termine := make(chan error, 1)
	go func() { termine <- serve(ctx, journalMuet()) }()
	select {
	case err := <-termine:
		return true, err
	case <-time.After(delaiDeRefusAuDemarrage):
		arreter()
		return false, <-termine
	}
}

func TestUS031DureeDeReservationHorsBornesRefuseeAuDemarrage(t *testing.T) {
	cas := []struct {
		variable string
		valeur   string
	}{
		{"RESERVATION_DUREE", "999ms"},
		{"RESERVATION_DUREE_MAX", "999ms"},
	}
	for _, c := range cas {
		t.Run(c.variable+"="+c.valeur, func(t *testing.T) {
			variables := environnementDeDemarrageSansDependance(t)
			variables[c.variable] = c.valeur
			avant := time.Now()
			arreteSeul, err := demarrerAvecLaConfiguration(t, variables)
			if !arreteSeul || err == nil {
				t.Fatalf("%s=%s : démarrage non refusé (arrêt seul %t, erreur %v)", c.variable, c.valeur, arreteSeul, err)
			}
			if !strings.Contains(err.Error(), c.variable) {
				t.Errorf("%s=%s refusé sans nommer la variable : %v", c.variable, c.valeur, err)
			}
			t.Logf("%s=%s refusé en %s : %v", c.variable, c.valeur, time.Since(avant).Round(time.Millisecond), err)
		})
	}
}

func TestUS031DureeDeReservationDUneSecondeNonRefuseeAuDemarrage(t *testing.T) {
	variables := environnementDeDemarrageSansDependance(t)
	variables["RESERVATION_DUREE"] = "1s"
	variables["RESERVATION_DUREE_MAX"] = "1s"
	arreteSeul, err := demarrerAvecLaConfiguration(t, variables)
	if arreteSeul && err != nil && strings.Contains(err.Error(), "RESERVATION_DUREE") {
		t.Fatalf("RESERVATION_DUREE=1s refusée au démarrage : %v", err)
	}
}
