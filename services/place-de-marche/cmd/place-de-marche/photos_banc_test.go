package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	imageSeaweedFS      = "chrislusf/seaweedfs:3.80@sha256:1055999e08eed1789b0ae45d235126e4495e23d3fb9d6396293fd42539b1ae6a"
	portS3              = "8333/tcp"
	identifiantS3Local  = "local"
	regionS3Locale      = "us-east-1"
	typePhotoJPEG       = "image/jpeg"
	prefixeDesPhotosURL = "/photos"
)

type stockageS3DeTest struct {
	adresse string
	erreur  error
}

var (
	s3Partage     stockageS3DeTest
	s3PartageUnit sync.Once
)

type bucketDeTest struct {
	nom     string
	adresse string
	client  *s3.Client
}

type photoRecue struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	ExpireLe string `json:"expireLe"`
	Largeur  int    `json:"largeur"`
	Hauteur  int    `json:"hauteur"`
}

type listePhotos struct {
	Elements []photoRecue `json:"elements"`
}

func demarrerSeaweedFS(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("intégration S3 ignorée en mode -short")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)
	s3PartageUnit.Do(func() {
		ctx, annuler := context.WithTimeout(context.Background(), 3*time.Minute)
		defer annuler()
		conteneur, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        imageSeaweedFS,
				Cmd:          []string{"server", "-dir=/data", "-ip.bind=0.0.0.0", "-s3", "-s3.port=8333"},
				ExposedPorts: []string{portS3},
				WaitingFor:   wait.ForHTTP("/status").WithPort(portS3).WithStartupTimeout(2 * time.Minute),
			},
			Started: true,
		})
		if err != nil {
			s3Partage.erreur = err
			return
		}
		point, err := conteneur.PortEndpoint(ctx, portS3, "http")
		s3Partage = stockageS3DeTest{adresse: point, erreur: err}
	})
	if s3Partage.erreur != nil {
		t.Fatalf("démarrage de SeaweedFS : %v", s3Partage.erreur)
	}
	return s3Partage.adresse
}

func clientS3(adresse string) *s3.Client {
	return s3.New(s3.Options{
		Region:                     regionS3Locale,
		BaseEndpoint:               aws.String(adresse),
		UsePathStyle:               true,
		Credentials:                aws.NewCredentialsCache(aws.CredentialsProviderFunc(identifiantsLocaux)),
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
}

func identifiantsLocaux(context.Context) (aws.Credentials, error) {
	return aws.Credentials{AccessKeyID: identifiantS3Local, SecretAccessKey: identifiantS3Local}, nil
}

func nouveauBucket(t *testing.T) bucketDeTest {
	t.Helper()
	adresse := demarrerSeaweedFS(t)
	b := bucketDeTest{nom: "photos-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:20], adresse: adresse, client: clientS3(adresse)}
	attendre(t, time.Minute, "création du bucket "+b.nom, func() bool {
		_, err := b.client.CreateBucket(t.Context(), &s3.CreateBucketInput{Bucket: aws.String(b.nom)})
		return err == nil
	})
	return b
}

func (b bucketDeTest) variables() map[string]string {
	return map[string]string{
		"PHOTOS_BUCKET":             b.nom,
		"PHOTOS_S3_ADRESSE":         b.adresse,
		"PHOTOS_S3_STYLE_CHEMIN":    "true",
		"AWS_ACCESS_KEY_ID":         identifiantS3Local,
		"AWS_SECRET_ACCESS_KEY":     identifiantS3Local,
		"AWS_EC2_METADATA_DISABLED": "true",
	}
}

func (b bucketDeTest) cles(t *testing.T) []string {
	t.Helper()
	sortie, err := b.client.ListObjectsV2(t.Context(), &s3.ListObjectsV2Input{Bucket: aws.String(b.nom)})
	if err != nil {
		t.Fatalf("inventaire du bucket %s : %v", b.nom, err)
	}
	var cles []string
	for _, objet := range sortie.Contents {
		cles = append(cles, aws.ToString(objet.Key))
	}
	return cles
}

func (b bucketDeTest) objet(t *testing.T, cle string) ([]byte, string, bool) {
	t.Helper()
	sortie, err := b.client.GetObject(t.Context(), &s3.GetObjectInput{Bucket: aws.String(b.nom), Key: aws.String(cle)})
	if err != nil {
		return nil, "", false
	}
	defer func() { _ = sortie.Body.Close() }()
	var contenu bytes.Buffer
	if _, err := contenu.ReadFrom(sortie.Body); err != nil {
		t.Fatalf("lecture de l'objet %s : %v", cle, err)
	}
	return contenu.Bytes(), aws.ToString(sortie.ContentType), true
}

func photoJPEGDeBanc() []byte {
	toile := image.NewRGBA(image.Rect(0, 0, 32, 24))
	for x := range 32 {
		for y := range 24 {
			toile.Set(x, y, color.RGBA{R: uint8(x * 8), G: uint8(y * 10), B: 90, A: 255})
		}
	}
	var tampon bytes.Buffer
	if err := jpeg.Encode(&tampon, toile, nil); err != nil {
		panic(err)
	}
	return tampon.Bytes()
}

func cheminPhotosVendeur(id string) string {
	return cheminArticles + "/" + id + prefixeDesPhotosURL
}

func cheminPhotosPubliques(id string) string {
	return "/marche/v1/articles/" + id + prefixeDesPhotosURL
}

func corpsPhoto(contenu []byte, typ string) option {
	return func(e *envoi) {
		e.corps = contenu
		e.typeContenu = typ
	}
}

func (e *environnement) deposerPhoto(t *testing.T, jeton, id string, contenu []byte) reponse {
	t.Helper()
	return e.envoyer(t, http.MethodPost, cheminPhotosVendeur(id), jeton, corpsPhoto(contenu, typePhotoJPEG))
}

func (e *environnement) photosDuVendeur(t *testing.T, jeton, id string) listePhotos {
	t.Helper()
	var liste listePhotos
	e.envoyer(t, http.MethodGet, cheminPhotosVendeur(id), jeton).exiger(t, http.StatusOK).lire(t, &liste)
	return liste
}

func (e *environnement) assurerUnePhoto(t *testing.T, jeton, id string) {
	t.Helper()
	if !e.avecPhotos || len(e.photosDuVendeur(t, jeton, id).Elements) > 0 {
		return
	}
	e.deposerPhoto(t, jeton, id, photoJPEGDeBanc()).exiger(t, http.StatusCreated)
}

func avecLesVariables(t *testing.T, variables map[string]string) {
	t.Helper()
	for nom, valeur := range variables {
		t.Setenv(nom, valeur)
	}
}
