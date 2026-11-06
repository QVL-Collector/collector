package token_test

import (
	"testing"

	"collector/libs/platform/token"
)

const (
	emetteurDuRealm = "https://auth.collector.test/realms/collector"
	audienceDuBanc  = "interactions"
)

func lecture(valeurs map[string]string) func(string) string {
	return func(nom string) string { return valeurs[nom] }
}

func TestUS028LoadConfigAppliqueLesValeursParDefaut(t *testing.T) {
	config, err := token.LoadConfig(lecture(map[string]string{"OIDC_EMETTEUR": emetteurDuRealm}), audienceDuBanc)
	if err != nil {
		t.Fatalf("configuration refusée : %v", err)
	}
	attendue := token.Config{
		Issuer:          emetteurDuRealm,
		Audience:        audienceDuBanc,
		AuthorizedParty: "collector-web",
		KeySetURL:       emetteurDuRealm + "/protocol/openid-connect/certs",
	}
	if config != attendue {
		t.Fatalf("configuration %+v, attendue %+v", config, attendue)
	}
}

func TestUS028LoadConfigRespecteLesValeursFournies(t *testing.T) {
	config, err := token.LoadConfig(lecture(map[string]string{
		"OIDC_EMETTEUR":        emetteurDuRealm,
		"OIDC_JEU_DE_CLES_URL": "http://keycloak.auth.svc:8080/realms/collector/protocol/openid-connect/certs",
		"OIDC_AUDIENCE":        "place-de-marche",
		"OIDC_CLIENT_AUTORISE": "collector-admin",
	}), audienceDuBanc)
	if err != nil {
		t.Fatalf("configuration refusée : %v", err)
	}
	attendue := token.Config{
		Issuer:          emetteurDuRealm,
		Audience:        "place-de-marche",
		AuthorizedParty: "collector-admin",
		KeySetURL:       "http://keycloak.auth.svc:8080/realms/collector/protocol/openid-connect/certs",
	}
	if config != attendue {
		t.Fatalf("configuration %+v, attendue %+v", config, attendue)
	}
}

func TestUS028LoadConfigRefuseUnEmetteurAbsent(t *testing.T) {
	for nom, valeur := range map[string]string{"absent": "", "espaces": "   "} {
		t.Run(nom, func(t *testing.T) {
			if config, err := token.LoadConfig(lecture(map[string]string{"OIDC_EMETTEUR": valeur}), audienceDuBanc); err == nil {
				t.Fatalf("configuration acceptée sans émetteur : %+v", config)
			}
		})
	}
}

func TestUS028ConfigurationSansAudienceNeDonneAucunVerificateur(t *testing.T) {
	config, err := token.LoadConfig(lecture(map[string]string{"OIDC_EMETTEUR": emetteurDuRealm}), "")
	if err != nil {
		return
	}
	if verificateur, err := token.NewVerifier(t.Context(), config); err == nil {
		t.Fatalf("vérificateur créé sans audience : %+v", verificateur)
	}
}
