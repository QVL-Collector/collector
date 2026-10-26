package main

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/annonces/stockage"
	"collector/services/place-de-marche/internal/migrations"
)

const imagePostgreSQL = "postgres:18.3@sha256:7e32e9833a6fb1c92c32552794cb6ed569d51b445a54907d35fc112ef39684db"

func demarrerPostgreSQL(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("intégration PostgreSQL ignorée en mode -short")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := contexteBorne(t, 3*time.Minute)
	conteneur, err := postgres.Run(ctx, imagePostgreSQL,
		postgres.WithDatabase("marche"),
		postgres.WithUsername("marche_test"),
		postgres.WithPassword(strings.ReplaceAll(uuid.NewString(), "-", "")),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, conteneur)
	if err != nil {
		t.Fatalf("démarrage de PostgreSQL : %v", err)
	}
	adresse, err := conteneur.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("adresse de PostgreSQL : %v", err)
	}
	return adresse
}

func demarrerSurPostgreSQL(t *testing.T) *environnement {
	t.Helper()
	adresse := demarrerPostgreSQL(t)
	ctx := contexteBorne(t, time.Minute)
	for range 2 {
		if err := migrations.Apply(ctx, journalMuet(), adresse); err != nil {
			t.Fatalf("migrations : %v", err)
		}
	}
	pool, err := stockage.Connect(ctx, adresse)
	if err != nil {
		t.Fatalf("connexion : %v", err)
	}
	t.Cleanup(pool.Close)
	depot := stockage.NewStore(pool)
	return demarrer(t, annonces.NewService(depot, depot))
}

func TestUS009IntegrationPostgreSQL(t *testing.T) {
	e := demarrerSurPostgreSQL(t)
	for _, s := range scenariosMetier() {
		t.Run(s.nom, func(t *testing.T) {
			s.jouer(t, e)
		})
	}
	t.Run("réponses conformes au contrat OpenAPI sur PostgreSQL", func(t *testing.T) {
		routeur := chargerContrat(t, e.serveur.URL)
		for _, ech := range e.historique() {
			verifierEchange(t, routeur, ech)
		}
	})
}
