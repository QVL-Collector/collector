package database_test

import (
	"strings"
	"testing"

	"collector/libs/platform/database"
)

const (
	secretDuRole  = "s3cr3t-du-role"
	adresseVerif  = "postgres://interactions_execution:" + secretDuRole + "@base.collector.test:5432/interactions?sslmode=verify-full&sslrootcert=/etc/rds/paquet.pem"
	adresseLocale = "postgres://interactions_execution:" + secretDuRole + "@localhost:5432/interactions?sslmode=disable"
)

func variables(valeurs map[string]string) func(string) string {
	return func(nom string) string { return valeurs[nom] }
}

func exigerRefus(t *testing.T, valeurs map[string]string) {
	t.Helper()
	config, err := database.LoadConfig(variables(valeurs))
	if err == nil {
		t.Fatalf("configuration acceptée : %+v", config)
	}
	if strings.Contains(err.Error(), secretDuRole) {
		t.Fatalf("le refus divulgue le mot de passe : %v", err)
	}
}

func TestUS028LoadConfigAccepteUneAdresseVerifyFullParDefaut(t *testing.T) {
	config, err := database.LoadConfig(variables(map[string]string{"BASE_DE_DONNEES_URL": adresseVerif}))
	if err != nil || config.URL != adresseVerif {
		t.Fatalf("configuration %+v, erreur %v ; adresse conservée attendue", config, err)
	}
}

func TestUS028LoadConfigExigeTLSQuandLaVariableEstAbsenteOuVide(t *testing.T) {
	for nom, exige := range map[string]map[string]string{
		"absente": {"BASE_DE_DONNEES_URL": adresseLocale},
		"vide":    {"BASE_DE_DONNEES_URL": adresseLocale, "BASE_DE_DONNEES_TLS_EXIGE": ""},
		"true":    {"BASE_DE_DONNEES_URL": adresseLocale, "BASE_DE_DONNEES_TLS_EXIGE": "true"},
	} {
		t.Run(nom, func(t *testing.T) { exigerRefus(t, exige) })
	}
}

func TestUS028LoadConfigRefuseUnSSLModeAutreQueVerifyFull(t *testing.T) {
	base := "postgres://interactions_execution:" + secretDuRole + "@base.collector.test:5432/interactions"
	for _, suffixe := range []string{"", "?sslmode=disable", "?sslmode=require", "?sslmode=verify-ca", "?sslmode=prefer", "?sslmode=VERIFY-FULL"} {
		t.Run("sslmode"+suffixe, func(t *testing.T) {
			exigerRefus(t, map[string]string{"BASE_DE_DONNEES_URL": base + suffixe})
		})
	}
}

func TestUS028LoadConfigRefuseUneAdresseQuiNEstPasPostgres(t *testing.T) {
	for _, adresse := range []string{
		"mysql://root:" + secretDuRole + "@base.collector.test:3306/interactions?sslmode=verify-full",
		"https://base.collector.test/interactions?sslmode=verify-full",
		"base.collector.test:5432/interactions?sslmode=verify-full",
		"pas une adresse",
		"://interactions?sslmode=verify-full",
	} {
		t.Run(adresse, func(t *testing.T) {
			exigerRefus(t, map[string]string{"BASE_DE_DONNEES_URL": adresse, "BASE_DE_DONNEES_TLS_EXIGE": "false"})
		})
	}
}

func TestUS028LoadConfigRefuseUneAdresseAbsente(t *testing.T) {
	for nom, valeur := range map[string]string{"absente": "", "espaces": "   "} {
		t.Run(nom, func(t *testing.T) {
			exigerRefus(t, map[string]string{"BASE_DE_DONNEES_URL": valeur})
		})
	}
}

func TestUS028LoadConfigSansTLSAccepteUnAutreSSLModeEnLocal(t *testing.T) {
	config, err := database.LoadConfig(variables(map[string]string{"BASE_DE_DONNEES_URL": adresseLocale, "BASE_DE_DONNEES_TLS_EXIGE": "false"}))
	if err != nil || config.URL != adresseLocale {
		t.Fatalf("configuration %+v, erreur %v ; adresse locale acceptée attendue", config, err)
	}
}

func TestUS028LoadConfigRefuseUneValeurDeTLSIllisible(t *testing.T) {
	for _, valeur := range []string{"peut-etre", "oui", "2"} {
		t.Run(valeur, func(t *testing.T) {
			exigerRefus(t, map[string]string{"BASE_DE_DONNEES_URL": adresseVerif, "BASE_DE_DONNEES_TLS_EXIGE": valeur})
		})
	}
}
