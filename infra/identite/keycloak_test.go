package identite

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	portKeycloak        = "8080/tcp"
	dossierDImport      = "/opt/keycloak/data/import/"
	compteDAmorce       = "amorce-test"
	clientDAdmin        = "admin-cli"
	delaiDeDemarrage    = 4 * time.Minute
	delaiDuTest         = 6 * time.Minute
	cheminDeDecouverte  = "/realms/" + Realm + "/.well-known/openid-configuration"
	cheminDuJetonMaster = "/realms/master/protocol/openid-connect/token"
	cheminAdminDuRealm  = "/admin/realms/" + Realm
)

type keycloakDeTest struct {
	base       string
	ctx        context.Context
	jetonAdmin string
}

type utilisateurDeTest struct {
	nom    string
	secret string
}

func valeurAleatoire() string {
	return rand.Text()
}

func demarrerKeycloak(t *testing.T) *keycloakDeTest {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx, annuler := context.WithTimeout(context.Background(), delaiDuTest)
	t.Cleanup(annuler)
	realm, err := filepath.Abs(FichierDuRealm)
	if err != nil {
		t.Fatalf("chemin du realm : %v", err)
	}
	secretDAmorce := valeurAleatoire()
	conteneur, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        ImageKeycloak,
			ExposedPorts: []string{portKeycloak},
			Cmd:          []string{"start-dev", "--import-realm"},
			Env: map[string]string{
				"KC_BOOTSTRAP_ADMIN_USERNAME": compteDAmorce,
				"KC_BOOTSTRAP_ADMIN_PASSWORD": secretDAmorce,
			},
			Files: []testcontainers.ContainerFile{
				{HostFilePath: realm, ContainerFilePath: dossierDImport + FichierDuRealm, FileMode: 0o644},
			},
			WaitingFor: wait.ForHTTP(cheminDeDecouverte).WithPort(portKeycloak).WithStartupTimeout(delaiDeDemarrage),
		},
		Started: true,
	})
	testcontainers.CleanupContainer(t, conteneur)
	if err != nil {
		t.Fatalf("démarrage de Keycloak : %v", err)
	}
	point, err := conteneur.PortEndpoint(ctx, portKeycloak, "http")
	if err != nil {
		t.Fatalf("adresse de Keycloak : %v", err)
	}
	kc := &keycloakDeTest{base: point, ctx: ctx}
	kc.jetonAdmin = kc.obtenirJetonAdmin(t, secretDAmorce)
	return kc
}

func (kc *keycloakDeTest) obtenirJetonAdmin(t *testing.T, secretDAmorce string) string {
	t.Helper()
	formulaire := url.Values{
		"grant_type": {"password"},
		"client_id":  {clientDAdmin},
		"username":   {compteDAmorce},
		"password":   {secretDAmorce},
	}
	var reponse struct {
		AccessToken string `json:"access_token"`
	}
	kc.envoyer(t, http.MethodPost, cheminDuJetonMaster, "application/x-www-form-urlencoded", strings.NewReader(formulaire.Encode()), "", http.StatusOK, &reponse)
	return reponse.AccessToken
}

func (kc *keycloakDeTest) creerUtilisateur(t *testing.T, nom, role string) utilisateurDeTest {
	t.Helper()
	secret := valeurAleatoire()
	corps, err := json.Marshal(map[string]any{
		"username":      nom,
		"enabled":       true,
		"email":         nom + "@collector.test",
		"emailVerified": true,
		"firstName":     "Compte",
		"lastName":      nom,
		"credentials":   []map[string]any{{"type": "password", "value": secret, "temporary": false}},
	})
	if err != nil {
		t.Fatalf("utilisateur %s : %v", nom, err)
	}
	reponse := kc.envoyer(t, http.MethodPost, cheminAdminDuRealm+"/users", "application/json", bytes.NewReader(corps), kc.jetonAdmin, http.StatusCreated, nil)
	identifiant := path.Base(reponse.Get("Location"))
	var representation json.RawMessage
	kc.envoyer(t, http.MethodGet, cheminAdminDuRealm+"/roles/"+role, "", nil, kc.jetonAdmin, http.StatusOK, &representation)
	attribution := append(append([]byte("["), representation...), ']')
	kc.envoyer(t, http.MethodPost, cheminAdminDuRealm+"/users/"+identifiant+"/role-mappings/realm", "application/json", bytes.NewReader(attribution), kc.jetonAdmin, http.StatusNoContent, nil)
	return utilisateurDeTest{nom: nom, secret: secret}
}

func (kc *keycloakDeTest) envoyer(t *testing.T, methode, chemin, typeDeContenu string, corps io.Reader, jeton string, attendu int, sortie any) http.Header {
	t.Helper()
	requete, err := http.NewRequestWithContext(kc.ctx, methode, kc.base+chemin, corps)
	if err != nil {
		t.Fatalf("%s %s : %v", methode, chemin, err)
	}
	if typeDeContenu != "" {
		requete.Header.Set("Content-Type", typeDeContenu)
	}
	if jeton != "" {
		requete.Header.Set("Authorization", "Bearer "+jeton)
	}
	reponse, err := http.DefaultClient.Do(requete)
	if err != nil {
		t.Fatalf("%s %s : %v", methode, chemin, err)
	}
	defer func() { _ = reponse.Body.Close() }()
	contenu, err := io.ReadAll(reponse.Body)
	if err != nil {
		t.Fatalf("%s %s : lecture : %v", methode, chemin, err)
	}
	if reponse.StatusCode != attendu {
		t.Fatalf("%s %s : statut %d, attendu %d", methode, chemin, reponse.StatusCode, attendu)
	}
	if sortie != nil {
		if err := json.Unmarshal(contenu, sortie); err != nil {
			t.Fatalf("%s %s : %v", methode, chemin, fmt.Errorf("réponse illisible : %w", err))
		}
	}
	return reponse.Header
}
