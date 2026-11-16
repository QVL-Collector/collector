package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	imageKeycloakDuRealm   = "quay.io/keycloak/keycloak:26.8.1@sha256:a78d6b79572168cb25886b80a8b5b8d4ed4390743e3eecb810538683e209dd6b"
	portDeKeycloak         = "8080/tcp"
	realmCollector         = "collector"
	clientTransactions     = "transactions"
	porteeReserver         = "reserver"
	variableSecretService  = "SECRET_CLIENT_TRANSACTIONS"
	compteDAmorceKeycloak  = "amorce-us030"
	redirectionDuFront     = "http://localhost:5173/rappel"
	formulaireDeConnexionK = "kc-form-login"
	typeFormulaire         = "application/x-www-form-urlencoded"
	cheminAdminCollector   = "/admin/realms/" + realmCollector
	toleranceDHorlogeD06   = 30 * time.Second
)

var (
	baliseDeFormulaire = regexp.MustCompile(`(?is)<form\b[^>]*>`)
	attributDeBalise   = regexp.MustCompile(`(?is)\b([a-z][a-z0-9_-]*)\s*=\s*"([^"]*)"`)
)

type fournisseurKeycloak struct {
	base               string
	ctx                context.Context
	secretDAmorce      string
	secretTransactions string
}

type compteKeycloak struct {
	nom    string
	secret string
	sujet  string
}

type chargeDeJeton map[string]any

func texteAleatoire() string {
	return strings.ToLower(rand.Text())
}

func (kc *fournisseurKeycloak) emetteur() string {
	return kc.base + "/realms/" + realmCollector
}

func (kc *fournisseurKeycloak) pointDeJeton() string {
	return kc.emetteur() + "/protocol/openid-connect/token"
}

func demarrerKeycloakDuRealm(t *testing.T) *fournisseurKeycloak {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx, annuler := context.WithTimeout(context.Background(), 40*time.Minute)
	t.Cleanup(annuler)
	realm, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "infra", "identite", "realm-collector.json"))
	if err != nil {
		t.Fatalf("chemin du realm : %v", err)
	}
	kc := &fournisseurKeycloak{ctx: ctx, secretDAmorce: texteAleatoire(), secretTransactions: texteAleatoire()}
	conteneur, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        imageKeycloakDuRealm,
			ExposedPorts: []string{portDeKeycloak},
			Cmd:          []string{"start-dev", "--import-realm"},
			Env: map[string]string{
				"KC_BOOTSTRAP_ADMIN_USERNAME": compteDAmorceKeycloak,
				"KC_BOOTSTRAP_ADMIN_PASSWORD": kc.secretDAmorce,
				variableSecretService:         kc.secretTransactions,
			},
			Files: []testcontainers.ContainerFile{
				{HostFilePath: realm, ContainerFilePath: "/opt/keycloak/data/import/realm-collector.json", FileMode: 0o644},
			},
			WaitingFor: wait.ForHTTP("/realms/" + realmCollector + "/.well-known/openid-configuration").WithPort(portDeKeycloak).WithStartupTimeout(4 * time.Minute),
		},
		Started: true,
	})
	testcontainers.CleanupContainer(t, conteneur)
	if err != nil {
		t.Fatalf("démarrage de Keycloak : %v", err)
	}
	kc.base, err = conteneur.PortEndpoint(ctx, portDeKeycloak, "http")
	if err != nil {
		t.Fatalf("adresse de Keycloak : %v", err)
	}
	return kc
}

func (kc *fournisseurKeycloak) poster(t *testing.T, adresse string, formulaire url.Values, client, secret string) (int, []byte) {
	t.Helper()
	requete, err := http.NewRequestWithContext(kc.ctx, http.MethodPost, adresse, strings.NewReader(formulaire.Encode()))
	if err != nil {
		t.Fatalf("requête vers Keycloak : %v", err)
	}
	requete.Header.Set("Content-Type", typeFormulaire)
	if client != "" {
		requete.SetBasicAuth(client, secret)
	}
	brute, err := http.DefaultClient.Do(requete)
	if err != nil {
		t.Fatalf("appel de Keycloak : %v", err)
	}
	defer func() { _ = brute.Body.Close() }()
	contenu, err := io.ReadAll(brute.Body)
	if err != nil {
		t.Fatalf("lecture de Keycloak : %v", err)
	}
	return brute.StatusCode, contenu
}

func jetonDAcces(t *testing.T, statut int, contenu []byte) string {
	t.Helper()
	var reponse struct {
		AccessToken string `json:"access_token"`
	}
	if statut != http.StatusOK || json.Unmarshal(contenu, &reponse) != nil || reponse.AccessToken == "" {
		t.Fatalf("jeton refusé par Keycloak : %d %s", statut, contenu)
	}
	return reponse.AccessToken
}

func (kc *fournisseurKeycloak) jetonAdmin(t *testing.T) string {
	t.Helper()
	statut, contenu := kc.poster(t, kc.base+"/realms/master/protocol/openid-connect/token", url.Values{
		"grant_type": {"password"},
		"client_id":  {"admin-cli"},
		"username":   {compteDAmorceKeycloak},
		"password":   {kc.secretDAmorce},
	}, "", "")
	return jetonDAcces(t, statut, contenu)
}

func (kc *fournisseurKeycloak) jetonDeService(t *testing.T, client, secret string) string {
	t.Helper()
	statut, contenu := kc.poster(t, kc.pointDeJeton(), url.Values{"grant_type": {"client_credentials"}}, client, secret)
	return jetonDAcces(t, statut, contenu)
}

func (kc *fournisseurKeycloak) jetonDeTransactions(t *testing.T) string {
	t.Helper()
	return kc.jetonDeService(t, clientTransactions, kc.secretTransactions)
}

func (kc *fournisseurKeycloak) administrer(t *testing.T, methode, chemin string, corps any, attendu int, sortie any) http.Header {
	t.Helper()
	var lecteur io.Reader
	if corps != nil {
		octets, err := json.Marshal(corps)
		if err != nil {
			t.Fatalf("corps d'administration : %v", err)
		}
		lecteur = bytes.NewReader(octets)
	}
	requete, err := http.NewRequestWithContext(kc.ctx, methode, kc.base+cheminAdminCollector+chemin, lecteur)
	if err != nil {
		t.Fatalf("%s %s : %v", methode, chemin, err)
	}
	if corps != nil {
		requete.Header.Set("Content-Type", typeJSON)
	}
	requete.Header.Set("Authorization", "Bearer "+kc.jetonAdmin(t))
	brute, err := http.DefaultClient.Do(requete)
	if err != nil {
		t.Fatalf("%s %s : %v", methode, chemin, err)
	}
	defer func() { _ = brute.Body.Close() }()
	contenu, err := io.ReadAll(brute.Body)
	if err != nil {
		t.Fatalf("%s %s : lecture : %v", methode, chemin, err)
	}
	if brute.StatusCode != attendu {
		t.Fatalf("%s %s : statut %d, attendu %d : %s", methode, chemin, brute.StatusCode, attendu, contenu)
	}
	if sortie != nil {
		if err := json.Unmarshal(contenu, sortie); err != nil {
			t.Fatalf("%s %s : réponse illisible : %v", methode, chemin, err)
		}
	}
	return brute.Header
}

func (kc *fournisseurKeycloak) creerCompte(t *testing.T, roles ...string) compteKeycloak {
	t.Helper()
	compte := compteKeycloak{nom: "compte-" + texteAleatoire()[:12], secret: texteAleatoire()}
	entetes := kc.administrer(t, http.MethodPost, "/users", map[string]any{
		"username":      compte.nom,
		"enabled":       true,
		"email":         compte.nom + "@collector.test",
		"emailVerified": true,
		"firstName":     "Compte",
		"lastName":      compte.nom,
		"credentials":   []map[string]any{{"type": "password", "value": compte.secret, "temporary": false}},
	}, http.StatusCreated, nil)
	compte.sujet = path.Base(entetes.Get("Location"))
	representations := make([]json.RawMessage, 0, len(roles))
	for _, role := range roles {
		var representation json.RawMessage
		kc.administrer(t, http.MethodGet, "/roles/"+role, nil, http.StatusOK, &representation)
		representations = append(representations, representation)
	}
	if len(representations) > 0 {
		kc.administrer(t, http.MethodPost, "/users/"+compte.sujet+"/role-mappings/realm", representations, http.StatusNoContent, nil)
	}
	return compte
}

func (kc *fournisseurKeycloak) creerAcheteur(t *testing.T) compteKeycloak {
	t.Helper()
	return kc.creerCompte(t, roleAcheteur)
}

func (kc *fournisseurKeycloak) creerVendeur(t *testing.T) compteKeycloak {
	t.Helper()
	return kc.creerCompte(t, roleAcheteur, roleVendeur)
}

func (kc *fournisseurKeycloak) seConnecter(t *testing.T, compte compteKeycloak) string {
	t.Helper()
	pot, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("pot à cookies : %v", err)
	}
	client := &http.Client{Jar: pot, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	verificateur := texteAleatoire() + texteAleatoire()
	empreinte := sha256.Sum256([]byte(verificateur))
	autorisation := url.Values{
		"client_id":             {clientAutorise},
		"response_type":         {"code"},
		"redirect_uri":          {redirectionDuFront},
		"scope":                 {"openid profile"},
		"state":                 {texteAleatoire()},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(empreinte[:])},
		"code_challenge_method": {"S256"},
	}
	adresse, page := kc.naviguer(t, client, http.MethodGet, kc.emetteur()+"/protocol/openid-connect/auth?"+autorisation.Encode(), nil)
	action := actionDeConnexion(t, page)
	cible, err := adresse.Parse(action)
	if err != nil {
		t.Fatalf("action de connexion : %v", err)
	}
	retour, _ := kc.naviguer(t, client, http.MethodPost, cible.String(), url.Values{"username": {compte.nom}, "password": {compte.secret}, "credentialId": {""}})
	code := retour.Query().Get("code")
	if !strings.HasPrefix(retour.String(), redirectionDuFront) || code == "" {
		t.Fatalf("connexion de %s : aucun code délivré (%s)", compte.nom, retour.Redacted())
	}
	statut, contenu := kc.poster(t, kc.pointDeJeton(), url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientAutorise},
		"redirect_uri":  {redirectionDuFront},
		"code":          {code},
		"code_verifier": {verificateur},
	}, "", "")
	return jetonDAcces(t, statut, contenu)
}

func (kc *fournisseurKeycloak) naviguer(t *testing.T, client *http.Client, methode, adresse string, champs url.Values) (*url.URL, string) {
	t.Helper()
	for range 10 {
		cible, err := url.Parse(adresse)
		if err != nil {
			t.Fatalf("adresse %q : %v", adresse, err)
		}
		if strings.HasPrefix(adresse, redirectionDuFront) {
			return cible, ""
		}
		page, suivante := kc.etapeDeNavigation(t, client, methode, cible, champs)
		if suivante == nil {
			return cible, page
		}
		methode, adresse, champs = http.MethodGet, suivante.String(), nil
	}
	t.Fatalf("plus de 10 redirections")
	return nil, ""
}

func (kc *fournisseurKeycloak) etapeDeNavigation(t *testing.T, client *http.Client, methode string, cible *url.URL, champs url.Values) (string, *url.URL) {
	t.Helper()
	var corps io.Reader
	if champs != nil {
		corps = strings.NewReader(champs.Encode())
	}
	requete, err := http.NewRequestWithContext(kc.ctx, methode, cible.String(), corps)
	if err != nil {
		t.Fatalf("%s %s : %v", methode, cible.Path, err)
	}
	if champs != nil {
		requete.Header.Set("Content-Type", typeFormulaire)
	}
	brute, err := client.Do(requete)
	if err != nil {
		t.Fatalf("%s %s : %v", methode, cible.Path, err)
	}
	defer func() { _ = brute.Body.Close() }()
	contenu, err := io.ReadAll(brute.Body)
	if err != nil {
		t.Fatalf("%s %s : lecture : %v", methode, cible.Path, err)
	}
	emplacement := brute.Header.Get("Location")
	if brute.StatusCode < 300 || brute.StatusCode > 399 || emplacement == "" {
		return string(contenu), nil
	}
	suivante, err := cible.Parse(emplacement)
	if err != nil {
		t.Fatalf("redirection illisible : %v", err)
	}
	return "", suivante
}

func actionDeConnexion(t *testing.T, page string) string {
	t.Helper()
	for _, balise := range baliseDeFormulaire.FindAllString(page, -1) {
		attributs := map[string]string{}
		for _, correspondance := range attributDeBalise.FindAllStringSubmatch(balise, -1) {
			attributs[strings.ToLower(correspondance[1])] = html.UnescapeString(correspondance[2])
		}
		if attributs["id"] == formulaireDeConnexionK && attributs["action"] != "" {
			return attributs["action"]
		}
	}
	t.Fatalf("formulaire de connexion absent de la page")
	return ""
}

func lireLaCharge(t *testing.T, jeton string) chargeDeJeton {
	t.Helper()
	parties := strings.Split(jeton, ".")
	if len(parties) != 3 {
		t.Fatalf("jeton mal formé")
	}
	brut, err := base64.RawURLEncoding.DecodeString(parties[1])
	if err != nil {
		t.Fatalf("charge du jeton : %v", err)
	}
	var charge chargeDeJeton
	if err := json.Unmarshal(brut, &charge); err != nil {
		t.Fatalf("charge du jeton : %v", err)
	}
	return charge
}

func (c chargeDeJeton) audiences() []string {
	switch valeur := c["aud"].(type) {
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

func (c chargeDeJeton) portees() []string {
	texte, _ := c["scope"].(string)
	return strings.Fields(texte)
}

func (c chargeDeJeton) dureeDeVie() time.Duration {
	expiration, _ := c["exp"].(float64)
	emission, _ := c["iat"].(float64)
	return time.Duration(expiration-emission) * time.Second
}

func jetonAvecChargeAlteree(t *testing.T, jeton string, alterer func(chargeDeJeton)) string {
	t.Helper()
	parties := strings.Split(jeton, ".")
	charge := lireLaCharge(t, jeton)
	alterer(charge)
	octets, err := json.Marshal(charge)
	if err != nil {
		t.Fatalf("charge altérée : %v", err)
	}
	return parties[0] + "." + base64.RawURLEncoding.EncodeToString(octets) + "." + parties[2]
}

func jetonNonSigne(t *testing.T, jeton string) string {
	t.Helper()
	parties := strings.Split(jeton, ".")
	entete := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	return entete + "." + parties[1] + "."
}

type clientDeServiceIntrus struct {
	identifiant string
	secret      string
}

func (kc *fournisseurKeycloak) creerClientDeService(t *testing.T, portees []string) clientDeServiceIntrus {
	t.Helper()
	intrus := clientDeServiceIntrus{identifiant: "intrus-" + texteAleatoire()[:8], secret: texteAleatoire()}
	kc.administrer(t, http.MethodPost, "/clients", map[string]any{
		"clientId":                  intrus.identifiant,
		"enabled":                   true,
		"publicClient":              false,
		"secret":                    intrus.secret,
		"serviceAccountsEnabled":    true,
		"standardFlowEnabled":       false,
		"directAccessGrantsEnabled": false,
		"fullScopeAllowed":          false,
		"defaultClientScopes":       portees,
	}, http.StatusCreated, nil)
	return intrus
}

func (kc *fournisseurKeycloak) identifiantInterne(t *testing.T, chemin, champ, valeur string) string {
	t.Helper()
	var elements []map[string]any
	kc.administrer(t, http.MethodGet, chemin, nil, http.StatusOK, &elements)
	for _, e := range elements {
		if e[champ] == valeur {
			id, _ := e["id"].(string)
			return id
		}
	}
	t.Fatalf("%s %s introuvable dans %s", champ, valeur, chemin)
	return ""
}

func (kc *fournisseurKeycloak) modifierTemporairement(t *testing.T, chemin string, modifier func(map[string]any)) {
	t.Helper()
	var origine map[string]any
	kc.administrer(t, http.MethodGet, chemin, nil, http.StatusOK, &origine)
	var copie map[string]any
	octets, _ := json.Marshal(origine)
	_ = json.Unmarshal(octets, &copie)
	modifier(copie)
	kc.administrer(t, http.MethodPut, chemin, copie, http.StatusNoContent, nil)
	t.Cleanup(func() { kc.administrer(t, http.MethodPut, chemin, origine, http.StatusNoContent, nil) })
}

func (kc *fournisseurKeycloak) jetonDeTransactionsExpire(t *testing.T) string {
	t.Helper()
	client := kc.identifiantInterne(t, "/clients?clientId="+clientTransactions, "clientId", clientTransactions)
	var jeton string
	t.Run("durée de vie réduite le temps d'émettre un jeton", func(t *testing.T) {
		kc.modifierTemporairement(t, "/clients/"+client, func(r map[string]any) {
			attributs, _ := r["attributes"].(map[string]any)
			attributs["access.token.lifespan"] = "1"
		})
		jeton = kc.jetonDeTransactions(t)
	})
	charge := lireLaCharge(t, jeton)
	expiration, _ := charge["exp"].(float64)
	attendre(t, time.Minute, "expiration du jeton de service au-delà de la tolérance d'horloge de 30 s (D06)", func() bool {
		return time.Now().After(time.Unix(int64(expiration), 0).Add(toleranceDHorlogeD06 + 2*time.Second))
	})
	return jeton
}

func (kc *fournisseurKeycloak) jetonDeTransactionsPourUneAutreAudience(t *testing.T, audience string) string {
	t.Helper()
	portee := kc.identifiantInterne(t, "/client-scopes", "name", porteeReserver)
	var mappeurs []map[string]any
	kc.administrer(t, http.MethodGet, "/client-scopes/"+portee+"/protocol-mappers/models", nil, http.StatusOK, &mappeurs)
	var jeton string
	t.Run("audience de la portée reserver détournée le temps d'émettre un jeton", func(t *testing.T) {
		for _, m := range mappeurs {
			if m["protocolMapper"] == "oidc-audience-mapper" {
				id, _ := m["id"].(string)
				kc.modifierTemporairement(t, "/client-scopes/"+portee+"/protocol-mappers/models/"+id, func(r map[string]any) {
					configuration, _ := r["config"].(map[string]any)
					configuration["included.custom.audience"] = audience
				})
			}
		}
		jeton = kc.jetonDeTransactions(t)
	})
	return jeton
}
