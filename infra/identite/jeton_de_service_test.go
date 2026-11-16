package identite

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
)

const audiencePlaceDeMarche = "place-de-marche"

type reponseDeJeton struct {
	statut       int
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Erreur       string `json:"error"`
}

func (kc *keycloakDeTest) demanderUnJeton(t *testing.T, formulaire url.Values, client, secret string) reponseDeJeton {
	t.Helper()
	requete, err := http.NewRequestWithContext(kc.ctx, http.MethodPost, kc.base+cheminDuJetonDuRealm, strings.NewReader(formulaire.Encode()))
	if err != nil {
		t.Fatalf("requête de jeton : %v", err)
	}
	requete.Header.Set("Content-Type", typeFormulaireEncodeEn)
	if client != "" {
		requete.SetBasicAuth(client, secret)
	}
	brute, err := http.DefaultClient.Do(requete)
	if err != nil {
		t.Fatalf("point de jeton : %v", err)
	}
	defer func() { _ = brute.Body.Close() }()
	contenu, err := io.ReadAll(brute.Body)
	if err != nil {
		t.Fatalf("lecture du jeton : %v", err)
	}
	reponse := reponseDeJeton{statut: brute.StatusCode}
	if err := json.Unmarshal(contenu, &reponse); err != nil {
		t.Fatalf("réponse du point de jeton illisible (%d) : %s", brute.StatusCode, contenu)
	}
	return reponse
}

func (kc *keycloakDeTest) jetonDeServiceTransactions(t *testing.T, formulaire url.Values) reponseDeJeton {
	t.Helper()
	return kc.demanderUnJeton(t, formulaire, ClientTransactions, kc.secretTransactions)
}

func identifiantsDuClient() url.Values {
	return url.Values{"grant_type": {"client_credentials"}}
}

func audiences(charge map[string]any) []string {
	switch valeur := charge["aud"].(type) {
	case string:
		return []string{valeur}
	case []any:
		liste := make([]string, 0, len(valeur))
		for _, a := range valeur {
			if texte, ok := a.(string); ok {
				liste = append(liste, texte)
			}
		}
		return liste
	}
	return nil
}

func portees(charge map[string]any) []string {
	texte, _ := charge["scope"].(string)
	return strings.Fields(texte)
}

func exigerJetonDeServiceRestreint(t *testing.T, reponse reponseDeJeton) {
	t.Helper()
	if reponse.statut != http.StatusOK || reponse.AccessToken == "" {
		t.Fatalf("jeton de service refusé : %d %s", reponse.statut, reponse.Erreur)
	}
	charge := revendications(t, reponse.AccessToken)
	if !slices.Contains(portees(charge), PorteeReserver) {
		t.Errorf("scope %q sans %s", charge["scope"], PorteeReserver)
	}
	if aud := audiences(charge); !slices.Contains(aud, audiencePlaceDeMarche) || slices.Contains(aud, ClientTransactions) {
		t.Errorf("aud %v : %s attendue, %s exclue", aud, audiencePlaceDeMarche, ClientTransactions)
	}
	if _, present := charge["realm_access"]; present {
		t.Errorf("realm_access présent dans le jeton de service : %v", charge["realm_access"])
	}
	if charge["azp"] != ClientTransactions || charge["typ"] != "Bearer" {
		t.Errorf("azp %v, typ %v ; %s et Bearer attendus", charge["azp"], charge["typ"], ClientTransactions)
	}
	duree, _ := charge["exp"].(float64)
	emission, _ := charge["iat"].(float64)
	if vie := duree - emission; vie <= 0 || vie > 300 {
		t.Errorf("durée de vie %v s, courte (5 min au plus) attendue", vie)
	}
	if reponse.RefreshToken != "" {
		t.Errorf("jeton de rafraîchissement délivré au client de service")
	}
}

func TestUS030JetonDeServiceDeTransactions(t *testing.T) {
	kc := demarrerKeycloak(t)
	t.Run("AC1 jeton de service : portée reserver, audience place-de-marche seule, aucun rôle de realm, durée courte", kc.jetonDeServiceRestreint)
	t.Run("AC1 portée reserver demandée explicitement : même jeton restreint", kc.porteeReserverDemandee)
	t.Run("AC4 secret du client faux : aucun jeton", kc.secretFauxRefuse)
	t.Run("AC4 client du site sans compte de service : aucun jeton client_credentials", kc.clientDuSiteSansCompteDeService)
	t.Run("AC4 client transactions : flux mot de passe refusé", kc.fluxMotDePasseRefuse)
	t.Run("AC4 AC9 jeton d'un utilisateur du site : jamais la portée reserver", kc.jetonDuSiteSansReserver)
}

func (kc *keycloakDeTest) jetonDeServiceRestreint(t *testing.T) {
	exigerJetonDeServiceRestreint(t, kc.jetonDeServiceTransactions(t, identifiantsDuClient()))
}

func (kc *keycloakDeTest) porteeReserverDemandee(t *testing.T) {
	formulaire := identifiantsDuClient()
	formulaire.Set("scope", PorteeReserver)
	exigerJetonDeServiceRestreint(t, kc.jetonDeServiceTransactions(t, formulaire))
}

func (kc *keycloakDeTest) secretFauxRefuse(t *testing.T) {
	reponse := kc.demanderUnJeton(t, identifiantsDuClient(), ClientTransactions, valeurAleatoire())
	if reponse.statut != http.StatusUnauthorized || reponse.AccessToken != "" {
		t.Fatalf("secret faux : %d, jeton délivré %t", reponse.statut, reponse.AccessToken != "")
	}
}

func (kc *keycloakDeTest) clientDuSiteSansCompteDeService(t *testing.T) {
	formulaire := identifiantsDuClient()
	formulaire.Set("client_id", ClientDuSite)
	reponse := kc.demanderUnJeton(t, formulaire, "", "")
	if reponse.statut < 400 || reponse.AccessToken != "" {
		t.Fatalf("client_credentials pour %s : %d, jeton délivré %t", ClientDuSite, reponse.statut, reponse.AccessToken != "")
	}
}

func (kc *keycloakDeTest) fluxMotDePasseRefuse(t *testing.T) {
	vendeur := kc.creerUtilisateur(t, "vendeur-"+strings.ToLower(valeurAleatoire()), RoleVendeur)
	formulaire := url.Values{"grant_type": {"password"}, "username": {vendeur.nom}, "password": {vendeur.secret}}
	reponse := kc.jetonDeServiceTransactions(t, formulaire)
	if reponse.statut < 400 || reponse.AccessToken != "" {
		t.Fatalf("flux mot de passe pour %s : %d, jeton délivré %t", ClientTransactions, reponse.statut, reponse.AccessToken != "")
	}
}

func (kc *keycloakDeTest) jetonDuSiteSansReserver(t *testing.T) {
	acheteur := kc.creerUtilisateur(t, "acheteur-"+strings.ToLower(valeurAleatoire()))
	for _, portee := range []string{"openid profile", "openid profile " + PorteeReserver} {
		charge, delivre := kc.connexionAvecPortee(t, acheteur, portee)
		if !delivre {
			t.Logf("portée %q : demande refusée par Keycloak", portee)
			continue
		}
		if slices.Contains(portees(charge), PorteeReserver) {
			t.Fatalf("portée %q : jeton du site avec %s : %v", portee, PorteeReserver, charge["scope"])
		}
		if charge["azp"] != ClientDuSite {
			t.Fatalf("azp %v, %s attendu", charge["azp"], ClientDuSite)
		}
		t.Logf("portée %q : jeton délivré, scope %q", portee, charge["scope"])
	}
}

func (kc *keycloakDeTest) connexionAvecPortee(t *testing.T, compte utilisateurDeTest, portee string) (map[string]any, bool) {
	t.Helper()
	n := nouveauNavigateur(t, kc)
	parametres := n.parametresDAutorisation()
	parametres.Set("scope", portee)
	page := n.ouvrir(t, parametres)
	if page.retour != nil && page.retour.Query().Get("error") != "" {
		return nil, false
	}
	arrivee := n.soumettre(t, page, formulaireDeConnexion, url.Values{"username": {compte.nom}, "password": {compte.secret}, "credentialId": {""}})
	if arrivee.code() == "" {
		t.Fatalf("portée %q : aucun code délivré", portee)
	}
	return n.echangerLeCode(t, arrivee.code()), true
}
