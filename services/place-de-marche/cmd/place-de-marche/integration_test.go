package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/annonces/controle"
	"collector/services/place-de-marche/internal/annonces/stockage"
	"collector/services/place-de-marche/internal/migrations"
)

const (
	imagePostgreSQL  = "postgres:18.3@sha256:7e32e9833a6fb1c92c32552794cb6ed569d51b445a54907d35fc112ef39684db"
	roleMigration    = "marche_migration"
	roleExecution    = "marche_execution"
	baseDuService    = "marche"
	droitInsuffisant = "42501"
)

type basePostgreSQL struct {
	maitre    *pgx.Conn
	migration string
	execution string
}

func secretAleatoire(t *testing.T) string {
	t.Helper()
	octets := make([]byte, 24)
	if _, err := rand.Read(octets); err != nil {
		t.Fatalf("secret : %v", err)
	}
	return hex.EncodeToString(octets)
}

func versLaBase(t *testing.T, adresseMaitre string, utilisateur func(*url.Userinfo) *url.Userinfo) string {
	t.Helper()
	adresse, err := url.Parse(adresseMaitre)
	if err != nil {
		t.Fatalf("adresse du conteneur : %v", err)
	}
	adresse.User = utilisateur(adresse.User)
	adresse.Path = "/" + baseDuService
	return adresse.String()
}

func enTantQue(role, secret string) func(*url.Userinfo) *url.Userinfo {
	return func(*url.Userinfo) *url.Userinfo { return url.UserPassword(role, secret) }
}

func enTantQueMaitre(maitre *url.Userinfo) *url.Userinfo {
	return maitre
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

func connecter(ctx context.Context, t *testing.T, adresse string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(ctx, adresse)
	if err != nil {
		t.Fatalf("connexion : %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func amorcer(ctx context.Context, t *testing.T, adresseMaitre string) basePostgreSQL {
	t.Helper()
	maitre := connecter(ctx, t, adresseMaitre)
	secretMigration, secretExecution := secretAleatoire(t), secretAleatoire(t)
	executerFormate(ctx, t, maitre, "CREATE ROLE %I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD %L", roleMigration, secretMigration)
	executerFormate(ctx, t, maitre, "CREATE ROLE %I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD %L", roleExecution, secretExecution)
	executerFormate(ctx, t, maitre, "CREATE DATABASE %I OWNER %I", baseDuService, roleMigration)
	executerFormate(ctx, t, maitre, "REVOKE CONNECT ON DATABASE %I FROM PUBLIC", baseDuService)
	executerFormate(ctx, t, maitre, "GRANT CONNECT ON DATABASE %I TO %I, %I", baseDuService, roleMigration, roleExecution)
	base := basePostgreSQL{
		maitre:    connecter(ctx, t, versLaBase(t, adresseMaitre, enTantQueMaitre)),
		migration: versLaBase(t, adresseMaitre, enTantQue(roleMigration, secretMigration)),
		execution: versLaBase(t, adresseMaitre, enTantQue(roleExecution, secretExecution)),
	}
	proprietaire := connecter(ctx, t, base.migration)
	if _, err := proprietaire.Exec(ctx, "REVOKE CREATE ON SCHEMA public FROM PUBLIC"); err != nil {
		t.Fatalf("révocation de CREATE sur public : %v", err)
	}
	return base
}

func demarrerPostgreSQL(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("intégration PostgreSQL ignorée en mode -short")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := contexteBorne(t, 3*time.Minute)
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
	adresse, err := conteneur.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("adresse de PostgreSQL : %v", err)
	}
	return adresse
}

func demarrerSurPostgreSQL(t *testing.T) (*environnement, basePostgreSQL, *pgxpool.Pool) {
	t.Helper()
	base, pool, service := servicePostgreSQL(t)
	return demarrer(t, service), base, pool
}

func servicePostgreSQL(t *testing.T) (basePostgreSQL, *pgxpool.Pool, *annonces.Service) {
	t.Helper()
	ctx := contexteBorne(t, 2*time.Minute)
	base := amorcer(ctx, t, demarrerPostgreSQL(t))
	for range 2 {
		if err := migrations.Apply(ctx, journalMuet(), base.migration); err != nil {
			t.Fatalf("migrations en %s : %v", roleMigration, err)
		}
	}
	pool, err := stockage.Connect(ctx, base.execution)
	if err != nil {
		t.Fatalf("connexion en %s : %v", roleExecution, err)
	}
	t.Cleanup(pool.Close)
	depot := stockage.NewStore(pool, controle.NewRequestEncoder(validateurDeContrats(t)))
	return base, pool, annonces.NewService(depot, depot)
}

func exigerRefusDeDroit(t *testing.T, pool *pgxpool.Pool, instruction string) {
	t.Helper()
	_, err := pool.Exec(t.Context(), instruction)
	var erreur *pgconn.PgError
	if !errors.As(err, &erreur) || erreur.Code != droitInsuffisant {
		t.Errorf("%q devait être refusé au rôle d'exécution (42501), obtenu %v", instruction, err)
	}
}

func verifierDroitsMinimaux(t *testing.T, base basePostgreSQL, pool *pgxpool.Pool) {
	exigerRefusDeDroit(t, pool, "CREATE TABLE intrus (id integer)")
	exigerRefusDeDroit(t, pool, "DELETE FROM article")
	exigerRefusDeDroit(t, pool, "DROP TABLE article")
	exigerRefusDeDroit(t, pool, "ALTER TABLE article ADD COLUMN intrus integer")
	exigerRefusDeDroit(t, pool, "TRUNCATE article")
	exigerRefusDeDroit(t, pool, "DELETE FROM outbox")
	exigerRefusDeDroit(t, pool, "DELETE FROM resultat_controle")
	var hors, possedes int
	err := base.maitre.QueryRow(t.Context(), `
		SELECT
			count(*) FILTER (WHERE c.relname NOT IN ('article', 'categorie', 'outbox', 'resultat_controle') AND has_table_privilege($1, c.oid, 'SELECT,INSERT,UPDATE,DELETE')),
			count(*) FILTER (WHERE pg_get_userbyid(c.relowner) = $1)
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p')`, roleExecution).Scan(&hors, &possedes)
	if err != nil {
		t.Fatalf("inventaire des droits : %v", err)
	}
	if hors != 0 || possedes != 0 {
		t.Errorf("rôle d'exécution : %d table(s) hors service accessibles, %d objet(s) possédé(s)", hors, possedes)
	}
}

func TestUS009IntegrationPostgreSQL(t *testing.T) {
	e, base, pool := demarrerSurPostgreSQL(t)
	for _, s := range scenariosMetier() {
		t.Run(s.nom, func(t *testing.T) {
			s.jouer(t, e)
		})
	}
	t.Run("réponses conformes au contrat OpenAPI sur PostgreSQL", func(t *testing.T) {
		routeur := chargerContrat(t, e.adresse)
		for _, ech := range e.historique() {
			verifierEchange(t, routeur, ech)
		}
	})
	t.Run("rôle d'exécution limité au DML de ses tables", func(t *testing.T) {
		verifierDroitsMinimaux(t, base, pool)
	})
}
