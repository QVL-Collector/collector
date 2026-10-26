package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	imageRabbitMQ         = "rabbitmq:4.3.7-management@sha256:79fb281ef7e0a15e3c5e0e12e66e944cebbf14a129493ef464369abd2c48d01a"
	vhostDuCanal          = "collector"
	identiteMarche        = "marche"
	identiteAdmin         = "admin-canal"
	fileDemandes          = "controle.demandes"
	fileResultats         = "pdm.resultats-controle"
	fileDemandesEcartees  = "controle.demandes.ecartes"
	fileResultatsEcartes  = "pdm.resultats-controle.ecartes"
	fileEspionDeResultats = "test.espion-resultats-controle"
	refusParLeBroker      = 406
	dossierDuBroker       = "/etc/rabbitmq"
)

var identitesDuCanal = []string{identiteMarche, identiteControle, "interactions", "keda", identiteAdmin}

type utilisateurDuCanal struct {
	Name             string   `json:"name"`
	PasswordHash     string   `json:"password_hash"`
	HashingAlgorithm string   `json:"hashing_algorithm"`
	Tags             []string `json:"tags"`
	Limits           struct{} `json:"limits"`
}

type canalDeTest struct {
	hote    string
	port    string
	secrets map[string]string
	admin   *amqp.Channel
}

type publication struct {
	echange  string
	cle      string
	id       string
	emetteur string
	corps    []byte
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
	var utilisateurs []utilisateurDuCanal
	for _, identite := range identitesDuCanal {
		secrets[identite] = secretAleatoire(t)
		utilisateurs = append(utilisateurs, utilisateurDuCanal{
			Name:             identite,
			PasswordHash:     empreinteRabbitMQ(t, secrets[identite]),
			HashingAlgorithm: "rabbit_password_hashing_sha256",
			Tags:             etiquettesDe(identite),
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
	contenu, err := fs.ReadFile(os.DirFS(filepath.Join("testdata", "canal")), relatif)
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
	ctx := contexteBorne(t, 3*time.Minute)
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
			).WithDeadline(2 * time.Minute),
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
	canal := &canalDeTest{hote: hote, port: port.Port(), secrets: secrets}
	canal.admin = canal.ouvrir(t, identiteAdmin)
	return canal
}

func (c *canalDeTest) adresse(identite string) string {
	return (&url.URL{Scheme: "amqp", User: url.UserPassword(identite, c.secrets[identite]), Host: c.hote + ":" + c.port, Path: "/" + vhostDuCanal}).String()
}

func (c *canalDeTest) ouvrir(t *testing.T, identite string) *amqp.Channel {
	t.Helper()
	connexion, err := amqp.Dial(c.adresse(identite))
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

func (c *canalDeTest) environnementDeService(identite string) map[string]string {
	return map[string]string{
		"CANAL_HOTE":         c.hote,
		"CANAL_PORT":         c.port,
		"CANAL_VHOST":        vhostDuCanal,
		"CANAL_UTILISATEUR":  identite,
		"CANAL_MOT_DE_PASSE": c.secrets[identite],
	}
}

func (c *canalDeTest) compter(t *testing.T, file string) int {
	t.Helper()
	etat, err := c.admin.QueueDeclarePassive(file, true, false, false, false, nil)
	if err != nil {
		t.Fatalf("lecture de la file %s : %v", file, err)
	}
	return etat.Messages
}

func (c *canalDeTest) poserUnEspion(t *testing.T) {
	t.Helper()
	if _, err := c.admin.QueueDeclare(fileEspionDeResultats, true, false, false, false, amqp.Table{"x-queue-type": "classic"}); err != nil {
		t.Fatalf("file espion : %v", err)
	}
	if err := c.admin.QueueBind(fileEspionDeResultats, cleDeRoutageV1, echangeResultats, false, nil); err != nil {
		t.Fatalf("liaison de l'espion : %v", err)
	}
}

func (c *canalDeTest) releverLEspion(t *testing.T) []amqp.Delivery {
	t.Helper()
	var relevees []amqp.Delivery
	for {
		livraison, presente, err := c.admin.Get(fileEspionDeResultats, true)
		if err != nil {
			t.Fatalf("relevé de l'espion : %v", err)
		}
		if !presente {
			return relevees
		}
		relevees = append(relevees, livraison)
	}
}

func (c *canalDeTest) publier(t *testing.T, identite string, p publication) error {
	t.Helper()
	canal := c.ouvrir(t, identite)
	fermeture := canal.NotifyClose(make(chan *amqp.Error, 1))
	if err := canal.Confirm(false); err != nil {
		t.Fatalf("confirmations en %s : %v", identite, err)
	}
	ctx := contexteBorne(t, 10*time.Second)
	confirmation, err := canal.PublishWithDeferredConfirmWithContext(ctx, p.echange, p.cle, true, false, amqp.Publishing{
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

func codeDeRefus(err error) int {
	var amqpErr *amqp.Error
	if errors.As(err, &amqpErr) {
		return amqpErr.Code
	}
	return 0
}
