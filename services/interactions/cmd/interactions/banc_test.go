package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"collector/libs/platform/logging"
)

const (
	imagePostgreSQL     = "postgres:18.3@sha256:7e32e9833a6fb1c92c32552794cb6ed569d51b445a54907d35fc112ef39684db"
	imageRabbitMQ       = "rabbitmq:4.3.7-management@sha256:79fb281ef7e0a15e3c5e0e12e66e944cebbf14a129493ef464369abd2c48d01a"
	baseInteractions    = "interactions"
	baseMarche          = "marche"
	vhostDuCanal        = "collector"
	identiteMarche      = "marche"
	identiteControle    = "controle"
	identiteInteraction = "interactions"
	identiteAdmin       = "admin-canal"
	echangeEvenements   = "pdm.evenements"
	echangeResultats    = "controle.resultats"
	echangeDemandes     = "controle.demandes"
	cleDecision         = "publication.decidee.v1"
	fileDecisions       = "interactions.publication-decidee"
	fileEcartes         = "interactions.publication-decidee.ecartes"
	dossierDuBroker     = "/etc/rabbitmq"
	droitInsuffisant    = "42501"
)

var identitesDuCanal = []string{identiteMarche, identiteControle, identiteInteraction, "keda", identiteAdmin}

type roles struct {
	migration string
	execution string
}

type basesPostgreSQL struct {
	maitre       *pgx.Conn
	interactions roles
	marche       roles
	adresses     map[string]string
}

type canalDeTest struct {
	hote      string
	port      string
	secrets   map[string]string
	admin     *amqp.Channel
	conteneur testcontainers.Container
}

type publication struct {
	echange  string
	cle      string
	id       string
	emetteur string
	corps    []byte
}

type journalPartage struct {
	mu     sync.Mutex
	tampon bytes.Buffer
}

func (j *journalPartage) Write(octets []byte) (int, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.tampon.Write(octets)
}

func (j *journalPartage) compter(fragment string) int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return strings.Count(j.tampon.String(), fragment)
}

func contexteBorne(t *testing.T, duree time.Duration) context.Context {
	t.Helper()
	ctx, annuler := context.WithTimeout(t.Context(), duree)
	t.Cleanup(annuler)
	return ctx
}

func secretAleatoire(t *testing.T) string {
	t.Helper()
	octets := make([]byte, 24)
	if _, err := rand.Read(octets); err != nil {
		t.Fatalf("secret : %v", err)
	}
	return hex.EncodeToString(octets)
}

func attendre(t *testing.T, delai time.Duration, quoi string, condition func() bool) time.Duration {
	t.Helper()
	debut := time.Now()
	for time.Since(debut) < delai {
		if condition() {
			return time.Since(debut)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("%s : non atteint en %s", quoi, delai)
	return delai
}

func adresseLibre(t *testing.T) string {
	t.Helper()
	ecoute, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("port libre : %v", err)
	}
	adresse := ecoute.Addr().String()
	_ = ecoute.Close()
	return adresse
}

func connecter(ctx context.Context, t *testing.T, adresse string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(ctx, adresse)
	if err != nil {
		t.Fatalf("connexion : %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func executerFormate(ctx context.Context, t *testing.T, conn *pgx.Conn, gabarit string, arguments ...string) {
	t.Helper()
	var instruction string
	if err := conn.QueryRow(ctx, "SELECT format($1, VARIADIC $2::text[])", gabarit, arguments).Scan(&instruction); err != nil {
		t.Fatalf("préparation de %q : %v", gabarit, err)
	}
	if _, err := conn.Exec(ctx, instruction); err != nil {
		t.Fatalf("amorçage %q : %v", gabarit, err)
	}
}

func adresseEnTantQue(t *testing.T, adresseMaitre, role, secret, base string) string {
	t.Helper()
	adresse, err := url.Parse(adresseMaitre)
	if err != nil {
		t.Fatalf("adresse du conteneur : %v", err)
	}
	if role != "" {
		adresse.User = url.UserPassword(role, secret)
	}
	adresse.Path = "/" + base
	return adresse.String()
}

func amorcerUneBase(ctx context.Context, t *testing.T, maitre *pgx.Conn, adresseMaitre, base string, adresses map[string]string) roles {
	t.Helper()
	r := roles{migration: base + "_migration", execution: base + "_execution"}
	secretMigration, secretExecution := secretAleatoire(t), secretAleatoire(t)
	executerFormate(ctx, t, maitre, "CREATE ROLE %I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD %L", r.migration, secretMigration)
	executerFormate(ctx, t, maitre, "CREATE ROLE %I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD %L", r.execution, secretExecution)
	executerFormate(ctx, t, maitre, "CREATE DATABASE %I OWNER %I", base, r.migration)
	executerFormate(ctx, t, maitre, "REVOKE CONNECT ON DATABASE %I FROM PUBLIC", base)
	executerFormate(ctx, t, maitre, "GRANT CONNECT ON DATABASE %I TO %I, %I", base, r.migration, r.execution)
	for _, cible := range []string{baseInteractions, baseMarche} {
		adresses[r.migration+"@"+cible] = adresseEnTantQue(t, adresseMaitre, r.migration, secretMigration, cible)
		adresses[r.execution+"@"+cible] = adresseEnTantQue(t, adresseMaitre, r.execution, secretExecution, cible)
	}
	if _, err := connecter(ctx, t, adresses[r.migration+"@"+base]).Exec(ctx, "REVOKE CREATE ON SCHEMA public FROM PUBLIC"); err != nil {
		t.Fatalf("révocation de CREATE sur public de %s : %v", base, err)
	}
	return r
}

func demarrerPostgreSQL(t *testing.T) basesPostgreSQL {
	t.Helper()
	if testing.Short() {
		t.Skip("intégration PostgreSQL ignorée en mode -short")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := contexteBorne(t, 4*time.Minute)
	conteneur, err := postgres.Run(ctx, imagePostgreSQL,
		postgres.WithDatabase("postgres"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword(secretAleatoire(t)),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, conteneur)
	if err != nil {
		t.Fatalf("démarrage de PostgreSQL : %v", err)
	}
	adresseMaitre, err := conteneur.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("adresse de PostgreSQL : %v", err)
	}
	maitre := connecter(ctx, t, adresseMaitre)
	adresses := map[string]string{}
	bases := basesPostgreSQL{
		interactions: amorcerUneBase(ctx, t, maitre, adresseMaitre, baseInteractions, adresses),
		marche:       amorcerUneBase(ctx, t, maitre, adresseMaitre, baseMarche, adresses),
		adresses:     adresses,
	}
	bases.maitre = connecter(ctx, t, adresseEnTantQue(t, adresseMaitre, "", "", baseInteractions))
	return bases
}

func (b basesPostgreSQL) adresse(role, base string) string {
	return b.adresses[role+"@"+base]
}

func empreinteRabbitMQ(t *testing.T, secret string) string {
	t.Helper()
	sel := make([]byte, 4)
	if _, err := rand.Read(sel); err != nil {
		t.Fatalf("sel : %v", err)
	}
	condense := sha256.Sum256(append(append([]byte{}, sel...), secret...))
	return base64.StdEncoding.EncodeToString(append(sel, condense[:]...))
}

func etiquettesDe(identite string) []string {
	switch identite {
	case "keda":
		return []string{"monitoring"}
	case identiteAdmin:
		return []string{"administrator"}
	}
	return []string{}
}

func genererIdentites(t *testing.T) (map[string]string, []byte) {
	t.Helper()
	secrets := map[string]string{}
	var utilisateurs []map[string]any
	for _, identite := range identitesDuCanal {
		secrets[identite] = secretAleatoire(t)
		utilisateurs = append(utilisateurs, map[string]any{
			"name":              identite,
			"password_hash":     empreinteRabbitMQ(t, secrets[identite]),
			"hashing_algorithm": "rabbit_password_hashing_sha256",
			"tags":              etiquettesDe(identite),
			"limits":            map[string]any{},
		})
	}
	contenu, err := json.Marshal(map[string]any{"users": utilisateurs})
	if err != nil {
		t.Fatalf("identités du canal : %v", err)
	}
	return secrets, contenu
}

func fichierDuBroker(t *testing.T, relatif string) []byte {
	t.Helper()
	contenu, err := fs.ReadFile(os.DirFS(filepath.Join("..", "..", "..", "..", "infra", "canal", "broker")), relatif)
	if err != nil {
		t.Fatalf("configuration du broker %s : %v", relatif, err)
	}
	return contenu
}

func copie(contenu []byte, destination string) testcontainers.ContainerFile {
	return testcontainers.ContainerFile{Reader: bytes.NewReader(contenu), ContainerFilePath: destination, FileMode: 0o644}
}

func demarrerRabbitMQ(t *testing.T) *canalDeTest {
	t.Helper()
	if testing.Short() {
		t.Skip("intégration RabbitMQ ignorée en mode -short")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := contexteBorne(t, 4*time.Minute)
	secrets, utilisateurs := genererIdentites(t)
	conteneur, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        imageRabbitMQ,
			ExposedPorts: []string{"5672/tcp"},
			Files: []testcontainers.ContainerFile{
				copie(nil, dossierDuBroker+"/conf.d/10-defaults.conf"),
				copie(fichierDuBroker(t, "conf.d/10-collector.conf"), dossierDuBroker+"/conf.d/10-collector.conf"),
				copie(fichierDuBroker(t, "enabled_plugins"), dossierDuBroker+"/enabled_plugins"),
				copie(fichierDuBroker(t, "definitions/10-topologie.json"), dossierDuBroker+"/definitions/10-topologie.json"),
				copie(utilisateurs, dossierDuBroker+"/definitions/00-utilisateurs.json"),
			},
			WaitingFor: wait.ForAll(
				wait.ForLog("Server startup complete"),
				wait.ForListeningPort("5672/tcp"),
			).WithDeadline(3 * time.Minute),
		},
		Started: true,
	})
	testcontainers.CleanupContainer(t, conteneur)
	if err != nil {
		t.Fatalf("démarrage de RabbitMQ : %v", err)
	}
	hote, err := conteneur.Host(ctx)
	if err != nil {
		t.Fatalf("hôte de RabbitMQ : %v", err)
	}
	port, err := conteneur.MappedPort(ctx, "5672/tcp")
	if err != nil {
		t.Fatalf("port de RabbitMQ : %v", err)
	}
	canal := &canalDeTest{hote: hote, port: port.Port(), secrets: secrets, conteneur: conteneur}
	canal.admin = canal.ouvrir(t, identiteAdmin)
	return canal
}

func (c *canalDeTest) ouvrir(t *testing.T, identite string) *amqp.Channel {
	t.Helper()
	adresse := (&url.URL{Scheme: "amqp", User: url.UserPassword(identite, c.secrets[identite]), Host: c.hote + ":" + c.port, Path: "/" + vhostDuCanal}).String()
	connexion, err := amqp.Dial(adresse)
	if err != nil {
		t.Fatalf("connexion au canal en %s : %v", identite, err)
	}
	t.Cleanup(func() { _ = connexion.Close() })
	canal, err := connexion.Channel()
	if err != nil {
		t.Fatalf("canal AMQP en %s : %v", identite, err)
	}
	return canal
}

func (c *canalDeTest) compter(t *testing.T, file string) int {
	t.Helper()
	etat, err := c.admin.QueueDeclarePassive(file, true, false, false, false, nil)
	if err != nil {
		t.Fatalf("lecture de la file %s : %v", file, err)
	}
	return etat.Messages
}

func (c *canalDeTest) publier(t *testing.T, identite string, p publication) error {
	t.Helper()
	canal := c.ouvrir(t, identite)
	fermeture := canal.NotifyClose(make(chan *amqp.Error, 1))
	if err := canal.Confirm(false); err != nil {
		t.Fatalf("confirmations en %s : %v", identite, err)
	}
	confirmation, err := canal.PublishWithDeferredConfirmWithContext(contexteBorne(t, 10*time.Second), p.echange, p.cle, true, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    p.id,
		UserId:       p.emetteur,
		Body:         p.corps,
	})
	if err != nil {
		return err
	}
	select {
	case raison := <-fermeture:
		return raison
	case <-confirmation.Done():
	}
	select {
	case raison := <-fermeture:
		return raison
	default:
	}
	if !confirmation.Acked() {
		return errors.New("publication non confirmée par le broker")
	}
	return nil
}

func (c *canalDeTest) relever(t *testing.T, file string) []amqp.Delivery {
	t.Helper()
	var relevees []amqp.Delivery
	for {
		livraison, presente, err := c.admin.Get(file, true)
		if err != nil {
			t.Fatalf("relevé de %s : %v", file, err)
		}
		if !presente {
			return relevees
		}
		relevees = append(relevees, livraison)
	}
}

func (c *canalDeTest) rabbitmqctl(t *testing.T, arguments ...string) []byte {
	t.Helper()
	code, sortie, err := c.conteneur.Exec(contexteBorne(t, time.Minute), append([]string{"rabbitmqctl", "-q"}, arguments...), tcexec.Multiplexed())
	if err != nil {
		t.Fatalf("rabbitmqctl %v : %v", arguments, err)
	}
	contenu, err := io.ReadAll(sortie)
	if err != nil || code != 0 {
		t.Fatalf("rabbitmqctl %v : code %d, %v : %s", arguments, code, err, contenu)
	}
	return contenu
}

func codeDeRefus(err error) int {
	var amqpErr *amqp.Error
	if errors.As(err, &amqpErr) {
		return amqpErr.Code
	}
	return 0
}

type serviceInteractions struct {
	adresse string
	client  *http.Client
	journal *journalPartage
	jetons  *fabriqueDeJetons
}

func migrer(t *testing.T, bases basesPostgreSQL) {
	t.Helper()
	t.Setenv("BASE_DE_DONNEES_URL", bases.adresse(bases.interactions.migration, baseInteractions))
	t.Setenv("BASE_DE_DONNEES_TLS_EXIGE", "false")
	for range 2 {
		if err := dispatch(contexteBorne(t, 2*time.Minute), journalMuet(), migrateCommand); err != nil {
			t.Fatalf("migrations en %s : %v", bases.interactions.migration, err)
		}
	}
}

func lancerInteractions(t *testing.T, bases basesPostgreSQL, canal *canalDeTest) *serviceInteractions {
	t.Helper()
	jetons := nouvelleFabriqueDeJetons(t)
	api, gestion := adresseLibre(t), adresseLibre(t)
	variables := map[string]string{
		"BASE_DE_DONNEES_URL":       bases.adresse(bases.interactions.execution, baseInteractions),
		"BASE_DE_DONNEES_TLS_EXIGE": "false",
		"OIDC_EMETTEUR":             emetteurAttendu,
		"OIDC_JEU_DE_CLES_URL":      jetons.jwks.URL,
		"CANAL_HOTE":                canal.hote,
		"CANAL_PORT":                canal.port,
		"CANAL_VHOST":               vhostDuCanal,
		"CANAL_UTILISATEUR":         identiteInteraction,
		"CANAL_MOT_DE_PASSE":        canal.secrets[identiteInteraction],
		"ADRESSE_API":               api,
		"ADRESSE_GESTION":           gestion,
	}
	for nom, valeur := range variables {
		t.Setenv(nom, valeur)
	}
	journal := &journalPartage{}
	ctx, arreter := context.WithCancel(context.Background())
	termine := make(chan error, 1)
	go func() { termine <- serve(ctx, logging.New(serviceName, journal, slog.LevelInfo)) }()
	t.Cleanup(func() {
		arreter()
		select {
		case erreur := <-termine:
			t.Logf("arrêt d'Interactions : %v", erreur)
		case <-time.After(15 * time.Second):
			t.Error("Interactions ne s'est pas arrêté")
		}
	})
	client := &http.Client{Timeout: 5 * time.Second}
	attendre(t, time.Minute, "démarrage d'Interactions", func() bool {
		select {
		case err := <-termine:
			t.Fatalf("Interactions s'est arrêté : %v", err)
		default:
		}
		reponse, err := client.Get("http://" + gestion + "/disponibilite")
		if err != nil {
			return false
		}
		_ = reponse.Body.Close()
		return reponse.StatusCode == http.StatusOK
	})
	return &serviceInteractions{adresse: "http://" + api, client: client, journal: journal, jetons: jetons}
}
