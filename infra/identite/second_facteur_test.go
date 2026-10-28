package identite

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
)

const (
	audienceDeLaPlaceDeMarche = "place-de-marche"
	dureeDuJetonDAcces        = 300
	erreurDeRequete           = "invalid_request"
)

type scenarioDuRealm struct {
	kc        *keycloakDeTest
	vendeur   utilisateurDeTest
	admin     utilisateurDeTest
	secretOTP string
}

func TestSecondFacteurDuRealmCollector(t *testing.T) {
	kc := demarrerKeycloak(t)
	s := &scenarioDuRealm{
		kc:      kc,
		vendeur: kc.creerUtilisateur(t, "vendeur-test", RoleVendeur),
		admin:   kc.creerUtilisateur(t, "admin-test", RoleAdmin),
	}
	t.Run("AC6 le vendeur obtient le code d'autorisation sans second facteur", s.vendeurSansSecondFacteur)
	t.Run("AC6 l'admin sans OTP doit le configurer et n'obtient aucun code avant", s.adminConfigureLOTP)
	t.Run("AC6 l'admin avec OTP doit saisir un code valide pour obtenir le code d'autorisation", s.adminSaisitLOTP)
	t.Run("PKCE S256 exigé par collector-web", s.pkceS256Exige)
	t.Run("redirection hors liste refusée", s.redirectionHorsListeRefusee)
}

func (s *scenarioDuRealm) vendeurSansSecondFacteur(t *testing.T) {
	n := nouveauNavigateur(t, s.kc)
	resultat := n.seConnecter(t, s.vendeur)
	if resultat.code() == "" {
		t.Fatalf("code d'autorisation attendu pour le vendeur, page %d reçue", resultat.statut)
	}
	if resultat.retour.Query().Get("state") != n.etat {
		t.Fatalf("state non restitué")
	}
	charge := n.echangerLeCode(t, resultat.code())
	exigerRole(t, charge, RoleVendeur)
	exigerAbsenceDuRole(t, charge, RoleAdmin)
	exigerJetonDuSite(t, charge)
}

func (s *scenarioDuRealm) adminConfigureLOTP(t *testing.T) {
	n := nouveauNavigateur(t, s.kc)
	configuration := n.seConnecter(t, s.admin)
	if configuration.retour != nil {
		t.Fatalf("aucun code ne doit être délivré à l'admin avant l'OTP")
	}
	if !contientLeFormulaire(configuration.page, formulaireDeConfigOTP) {
		t.Fatalf("page de configuration de l'OTP attendue")
	}
	if !strings.Contains(actionDuFormulaire(t, configuration.page, formulaireDeConfigOTP), "execution="+actionDeConfigOTP) {
		t.Fatalf("l'action requise %s est attendue", actionDeConfigOTP)
	}
	secret := valeurDuChamp(t, configuration.page, "totpSecret")
	resultat := n.soumettre(t, configuration, formulaireDeConfigOTP, url.Values{
		"totp":       {codeOTP(secret, 0)},
		"totpSecret": {secret},
		"userLabel":  {"test"},
	})
	if resultat.code() == "" {
		t.Fatalf("code attendu après configuration de l'OTP, page %d reçue", resultat.statut)
	}
	exigerRole(t, n.echangerLeCode(t, resultat.code()), RoleAdmin)
	s.secretOTP = secret
}

func (s *scenarioDuRealm) adminSaisitLOTP(t *testing.T) {
	if s.secretOTP == "" {
		t.Skip("OTP de l'admin non configuré par le cas précédent")
	}
	n := nouveauNavigateur(t, s.kc)
	saisie := n.seConnecter(t, s.admin)
	exigerFormulaireOTPSansCode(t, saisie, "après le mot de passe")
	refus := n.soumettre(t, saisie, formulaireDeSaisieOTP, url.Values{"otp": {codeOTPFaux(s.secretOTP)}})
	exigerFormulaireOTPSansCode(t, refus, "après un OTP faux")
	resultat := n.soumettre(t, refus, formulaireDeSaisieOTP, url.Values{"otp": {codeOTP(s.secretOTP, 1)}})
	if resultat.code() == "" {
		t.Fatalf("code attendu après un OTP valide, page %d reçue", resultat.statut)
	}
	exigerRole(t, n.echangerLeCode(t, resultat.code()), RoleAdmin)
}

func (s *scenarioDuRealm) pkceS256Exige(t *testing.T) {
	n := nouveauNavigateur(t, s.kc)
	sansDefi := n.parametresDAutorisation()
	sansDefi.Del("code_challenge")
	sansDefi.Del("code_challenge_method")
	exigerRefusDeRequete(t, n.ouvrir(t, sansDefi), "demande sans PKCE")
	enClair := n.parametresDAutorisation()
	enClair.Set("code_challenge_method", "plain")
	exigerRefusDeRequete(t, n.ouvrir(t, enClair), "PKCE plain")
}

func (s *scenarioDuRealm) redirectionHorsListeRefusee(t *testing.T) {
	n := nouveauNavigateur(t, s.kc)
	parametres := n.parametresDAutorisation()
	parametres.Set("redirect_uri", "http://localhost:5173/autre")
	resultat := n.ouvrir(t, parametres)
	if resultat.retour != nil || resultat.statut != http.StatusBadRequest {
		t.Fatalf("redirection hors liste : 400 attendu, statut %d", resultat.statut)
	}
}

func exigerFormulaireOTPSansCode(t *testing.T, resultat etape, moment string) {
	t.Helper()
	if resultat.retour != nil || !contientLeFormulaire(resultat.page, formulaireDeSaisieOTP) {
		t.Fatalf("%s : formulaire de saisie de l'OTP attendu, sans code", moment)
	}
}

func exigerRefusDeRequete(t *testing.T, resultat etape, cas string) {
	t.Helper()
	if resultat.retour == nil || resultat.retour.Query().Get("error") != erreurDeRequete {
		t.Fatalf("%s : refus %s attendu, statut %d", cas, erreurDeRequete, resultat.statut)
	}
}

func rolesDuRealm(charge map[string]any) []string {
	acces, _ := charge["realm_access"].(map[string]any)
	brut, _ := acces["roles"].([]any)
	roles := make([]string, 0, len(brut))
	for _, r := range brut {
		if role, ok := r.(string); ok {
			roles = append(roles, role)
		}
	}
	return roles
}

func exigerRole(t *testing.T, charge map[string]any, role string) {
	t.Helper()
	if !slices.Contains(rolesDuRealm(charge), role) {
		t.Fatalf("rôle %s absent du jeton : %v", role, rolesDuRealm(charge))
	}
}

func exigerAbsenceDuRole(t *testing.T, charge map[string]any, role string) {
	t.Helper()
	if slices.Contains(rolesDuRealm(charge), role) {
		t.Fatalf("rôle %s présent à tort dans le jeton", role)
	}
}

func exigerJetonDuSite(t *testing.T, charge map[string]any) {
	t.Helper()
	if charge["azp"] != ClientDuSite {
		t.Fatalf("azp %v, attendu %s", charge["azp"], ClientDuSite)
	}
	audiences, _ := charge["aud"].([]any)
	if !slices.Contains(audiences, any(audienceDeLaPlaceDeMarche)) {
		t.Fatalf("audience %s absente : %v", audienceDeLaPlaceDeMarche, charge["aud"])
	}
	expiration, _ := charge["exp"].(float64)
	emission, _ := charge["iat"].(float64)
	if expiration-emission != dureeDuJetonDAcces {
		t.Fatalf("durée du jeton d'accès %v s, attendu %d s", expiration-emission, dureeDuJetonDAcces)
	}
}
