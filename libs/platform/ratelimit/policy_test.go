package ratelimit_test

import (
	"net/netip"
	"testing"

	"collector/libs/platform/ratelimit"
)

const (
	varParMinute    = "LIMITE_DEBIT_PAR_MINUTE"
	varRafale       = "LIMITE_DEBIT_RAFALE"
	varTirAdresse   = "LIMITE_DEBIT_TIR_ADRESSE"
	varTirParMinute = "LIMITE_DEBIT_TIR_PAR_MINUTE"
	varTirRafale    = "LIMITE_DEBIT_TIR_RAFALE"
)

func netipAddr(adresse string) netip.Addr {
	return netip.MustParseAddr(adresse)
}

func variables(valeurs map[string]string) func(string) string {
	return func(nom string) string {
		return valeurs[nom]
	}
}

func charger(t *testing.T, valeurs map[string]string) ratelimit.Policy {
	t.Helper()
	politique, err := ratelimit.LoadPolicy(variables(valeurs))
	if err != nil {
		t.Fatalf("configuration %v refusée : %v", valeurs, err)
	}
	return politique
}

func TestUS027AC3ValeursParDefautSansAdresseDeTir(t *testing.T) {
	politique := charger(t, nil)

	if politique.Nominal != ratelimit.PerMinute(30, 10) {
		t.Fatalf("limite nominale %+v, 30/min rafale 10 attendue", politique.Nominal)
	}
	if politique.LoadTest != ratelimit.PerMinute(6000, 200) {
		t.Fatalf("limite de tir %+v, 6000/min rafale 200 attendue", politique.LoadTest)
	}
	if politique.LoadTestAddress.IsValid() {
		t.Fatalf("aucune adresse de tir attendue, %s chargée", politique.LoadTestAddress)
	}
}

func TestUS027AC3AdresseDeTirInjecteeALExecution(t *testing.T) {
	cas := map[string]string{
		"IPv4": "198.51.100.42",
		"IPv6": "2001:db8::42",
	}
	for nom, adresse := range cas {
		t.Run(nom, func(t *testing.T) {
			politique := charger(t, map[string]string{
				varParMinute:    "12",
				varRafale:       "3",
				varTirAdresse:   adresse,
				varTirParMinute: "9000",
				varTirRafale:    "500",
			})

			if politique.LoadTestAddress != netipAddr(adresse) {
				t.Fatalf("adresse de tir %s, %s attendue", politique.LoadTestAddress, adresse)
			}
			if politique.Nominal != ratelimit.PerMinute(12, 3) || politique.LoadTest != ratelimit.PerMinute(9000, 500) {
				t.Fatalf("limites %+v / %+v, 12/3 et 9000/500 attendues", politique.Nominal, politique.LoadTest)
			}
		})
	}
}

func TestUS027AC3ValeursInvalidesRefusees(t *testing.T) {
	cas := map[string]map[string]string{
		"débit nul":                   {varParMinute: "0"},
		"débit négatif":               {varParMinute: "-5"},
		"débit non numérique":         {varParMinute: "trente"},
		"débit décimal":               {varParMinute: "1.5"},
		"rafale nulle":                {varRafale: "0"},
		"rafale négative":             {varRafale: "-1"},
		"rafale non numérique":        {varRafale: "dix"},
		"débit de tir nul":            {varTirParMinute: "0"},
		"débit de tir non numérique":  {varTirParMinute: "beaucoup"},
		"rafale de tir négative":      {varTirRafale: "-200"},
		"rafale de tir non numérique": {varTirRafale: "x"},
		"adresse de tir illisible":    {varTirAdresse: "poste-de-martin"},
		"adresse de tir hors plage":   {varTirAdresse: "999.1.1.1"},
	}
	for nom, valeurs := range cas {
		t.Run(nom, func(t *testing.T) {
			if politique, err := ratelimit.LoadPolicy(variables(valeurs)); err == nil {
				t.Fatalf("configuration %v acceptée : %+v", valeurs, politique)
			}
		})
	}
}

func TestUS027AC3PlusPetitesValeursAcceptees(t *testing.T) {
	politique := charger(t, map[string]string{varParMinute: "1", varRafale: "1", varTirParMinute: "1", varTirRafale: "1"})

	if politique.Nominal != ratelimit.PerMinute(1, 1) || politique.LoadTest != ratelimit.PerMinute(1, 1) {
		t.Fatalf("limites %+v / %+v, 1/1 attendues", politique.Nominal, politique.LoadTest)
	}
}
