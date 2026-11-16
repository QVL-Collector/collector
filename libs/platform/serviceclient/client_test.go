package serviceclient_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"collector/libs/platform/serviceclient"
)

const (
	emetteur      = "https://auth.collector.test/realms/collector"
	clientParDef  = "transactions"
	secretService = "secret-de-test-tres-long"
	jetonEmis     = "jeton-de-service-emis"
)

func environnement(valeurs map[string]string) func(string) string {
	return func(nom string) string { return valeurs[nom] }
}

func TestChargementAvecLesValeursParDefaut(t *testing.T) {
	identifiants, err := serviceclient.LoadCredentials(environnement(map[string]string{"OIDC_CLIENT_SECRET": secretService}), emetteur+"/", clientParDef, "reserver")
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	if identifiants.TokenURL != emetteur+"/protocol/openid-connect/token" || identifiants.ClientID != clientParDef || identifiants.ClientSecret != secretService {
		t.Fatalf("identifiants inattendus : URL %q, client %q", identifiants.TokenURL, identifiants.ClientID)
	}
	if len(identifiants.Scopes) != 1 || identifiants.Scopes[0] != "reserver" {
		t.Fatalf("portées %v, [reserver] attendu", identifiants.Scopes)
	}
}

func TestChargementAvecLesValeursFournies(t *testing.T) {
	identifiants, err := serviceclient.LoadCredentials(environnement(map[string]string{
		"OIDC_CLIENT_SECRET": "  " + secretService + "  ",
		"OIDC_CLIENT_ID":     "autre-client",
		"OIDC_JETON_URL":     "http://keycloak.interne:8080/realms/collector/protocol/openid-connect/token",
	}), emetteur, clientParDef)
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	if identifiants.TokenURL != "http://keycloak.interne:8080/realms/collector/protocol/openid-connect/token" || identifiants.ClientID != "autre-client" || identifiants.ClientSecret != secretService {
		t.Fatalf("identifiants inattendus : URL %q, client %q", identifiants.TokenURL, identifiants.ClientID)
	}
}

func TestChargementRefuse(t *testing.T) {
	cas := map[string]struct {
		valeurs map[string]string
		emet    string
	}{
		"secret absent":             {map[string]string{}, emetteur},
		"secret fait d'espaces":     {map[string]string{"OIDC_CLIENT_SECRET": "   "}, emetteur},
		"URL du jeton illisible":    {map[string]string{"OIDC_CLIENT_SECRET": secretService, "OIDC_JETON_URL": "pas une url"}, emetteur},
		"émetteur relatif sans URL": {map[string]string{"OIDC_CLIENT_SECRET": secretService}, "realms/collector"},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			_, err := serviceclient.LoadCredentials(environnement(c.valeurs), c.emet, clientParDef)
			if err == nil {
				t.Fatalf("configuration acceptée")
			}
			if strings.Contains(err.Error(), secretService) {
				t.Fatalf("l'erreur révèle le secret : %v", err)
			}
		})
	}
}

type pointDeJeton struct {
	appels   atomic.Int32
	serveur  *httptest.Server
	statut   int
	formules atomic.Value
}

func demarrerPointDeJeton(t *testing.T, statut int) *pointDeJeton {
	t.Helper()
	p := &pointDeJeton{statut: statut}
	p.serveur = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.appels.Add(1)
		client, secret, basic := r.BasicAuth()
		if err := r.ParseForm(); err != nil || !basic || client != clientParDef || secret != secretService || r.PostForm.Get("grant_type") != "client_credentials" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p.formules.Store(r.PostForm.Get("scope"))
		if p.statut != http.StatusOK {
			w.WriteHeader(p.statut)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": jetonEmis, "token_type": "Bearer", "expires_in": 300})
	}))
	t.Cleanup(p.serveur.Close)
	return p
}

func (p *pointDeJeton) client(t *testing.T) *http.Client {
	t.Helper()
	return serviceclient.New(t.Context(), serviceclient.Credentials{TokenURL: p.serveur.URL, ClientID: clientParDef, ClientSecret: secretService, Scopes: []string{"reserver"}}, 5*time.Second)
}

func appelerAvec(t *testing.T, client *http.Client, adresse string) (int, error) {
	t.Helper()
	requete, err := http.NewRequestWithContext(t.Context(), http.MethodPost, adresse, nil)
	if err != nil {
		t.Fatalf("requête : %v", err)
	}
	reponse, err := client.Do(requete)
	if err != nil {
		return 0, err
	}
	_ = reponse.Body.Close()
	return reponse.StatusCode, nil
}

func TestJetonObtenuUneFoisPuisReutilise(t *testing.T) {
	jeton := demarrerPointDeJeton(t, http.StatusOK)
	var autorisations []string
	cible := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		autorisations = append(autorisations, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(cible.Close)
	client := jeton.client(t)
	for range 3 {
		statut, err := appelerAvec(t, client, cible.URL)
		if err != nil || statut != http.StatusCreated {
			t.Fatalf("appel : %v", err)
		}
	}
	if n := jeton.appels.Load(); n != 1 {
		t.Fatalf("%d demandes de jeton pour 3 appels, 1 attendue", n)
	}
	if portee, _ := jeton.formules.Load().(string); portee != "reserver" {
		t.Fatalf("portée demandée %q, reserver attendue", portee)
	}
	for _, a := range autorisations {
		if a != "Bearer "+jetonEmis {
			t.Fatalf("Authorization %q, jeton de service attendu", a)
		}
	}
}

func TestRedirectionRefuseeSansTransmettreLeJeton(t *testing.T) {
	jeton := demarrerPointDeJeton(t, http.StatusOK)
	var atteinte atomic.Int32
	ailleurs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atteinte.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ailleurs.Close)
	redirige := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, ailleurs.URL+"/vol", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirige.Close)
	if _, err := appelerAvec(t, jeton.client(t), redirige.URL); err == nil {
		t.Fatalf("redirection suivie sans erreur")
	}
	if atteinte.Load() != 0 {
		t.Fatalf("la cible de la redirection a été appelée")
	}
}

func TestJetonRefuseParKeycloakAucunAppel(t *testing.T) {
	jeton := demarrerPointDeJeton(t, http.StatusUnauthorized)
	var atteinte atomic.Int32
	cible := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atteinte.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(cible.Close)
	if _, err := appelerAvec(t, jeton.client(t), cible.URL); err == nil {
		t.Fatalf("appel réussi sans jeton de service")
	}
	if atteinte.Load() != 0 {
		t.Fatalf("service appelé sans jeton de service")
	}
}

func TestDelaiDuClientApplique(t *testing.T) {
	jeton := demarrerPointDeJeton(t, http.StatusOK)
	lent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(lent.Close)
	client := serviceclient.New(t.Context(), serviceclient.Credentials{TokenURL: jeton.serveur.URL, ClientID: clientParDef, ClientSecret: secretService}, 200*time.Millisecond)
	debut := time.Now()
	if _, err := appelerAvec(t, client, lent.URL); err == nil {
		t.Fatalf("appel lent réussi malgré le délai")
	}
	if duree := time.Since(debut); duree > 2*time.Second {
		t.Fatalf("délai non appliqué : %s", duree)
	}
}
