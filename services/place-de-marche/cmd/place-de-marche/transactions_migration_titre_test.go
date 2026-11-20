package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	versionAvantLeTitre = 1
	violationDeControle = "23514"
)

func migrerTransactionsJusqua(t *testing.T, adresse string, version int64) {
	t.Helper()
	sources := os.DirFS(filepath.Join("..", "..", "..", "transactions", "internal", "migrations"))
	noms, err := fs.Glob(sources, "*.sql")
	if err != nil || len(noms) == 0 {
		t.Fatalf("scripts de migration de Transactions : %v", err)
	}
	scripts := fstest.MapFS{}
	for _, nom := range noms {
		contenu, err := fs.ReadFile(sources, nom)
		if err != nil {
			t.Fatalf("lecture de %s : %v", nom, err)
		}
		scripts[nom] = &fstest.MapFile{Data: contenu}
	}
	configuration, err := pgx.ParseConfig(adresse)
	if err != nil {
		t.Fatalf("adresse de migration : %v", err)
	}
	db := stdlib.OpenDB(*configuration)
	defer func() { _ = db.Close() }()
	fournisseur, err := goose.NewProvider(goose.DialectPostgres, db, scripts)
	if err != nil {
		t.Fatalf("préparation des migrations : %v", err)
	}
	if _, err := fournisseur.UpTo(t.Context(), version); err != nil {
		t.Fatalf("migrations de Transactions jusqu'à la version %d : %v", version, err)
	}
}

const insertionDeCommande = `INSERT INTO commande (id, reservation_id, article_id, acheteur_id, statut, devise, prix_centimes, frais_port_centimes, total_paye_centimes, commission_centimes, montant_vendeur_centimes, echeance)
	VALUES ($1, $2, $3, $4, 'en_attente_paiement', 'EUR', 8000, 500, 8500, 425, 8075, $5)`

func insererCommandeTitree(t *testing.T, conn *pgx.Conn, titre string) error {
	t.Helper()
	_, err := conn.Exec(t.Context(), `INSERT INTO commande (id, reservation_id, article_id, acheteur_id, statut, devise, prix_centimes, frais_port_centimes, total_paye_centimes, commission_centimes, montant_vendeur_centimes, echeance, titre)
	VALUES ($1, $2, $3, $4, 'en_attente_paiement', 'EUR', 8000, 500, 8500, 425, 8075, $5, $6)`, uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), time.Now().Add(time.Hour), titre)
	return err
}

func exigerCommandeRepriseAvecUnTitre(t *testing.T, maitre *pgx.Conn, id string) {
	t.Helper()
	var titre string
	var prix int64
	if err := maitre.QueryRow(t.Context(), `SELECT titre, prix_centimes FROM commande WHERE id = $1`, id).Scan(&titre, &prix); err != nil {
		t.Fatalf("commande %s après migration : %v", id, err)
	}
	if n := len([]rune(titre)); n < 1 || n > 200 || prix != 8000 {
		t.Errorf("commande %s après migration : titre %q (%d caractères), prix %d", id, titre, n, prix)
	}
	t.Logf("commande d'avant la migration reprise avec le titre %q", titre)
}

func exigerViolationDeContrainte(t *testing.T, err error, cas string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != violationDeControle {
		t.Errorf("%s : %v, violation de contrainte attendue", cas, err)
	}
}

func TestUS032MigrationDuTitreSurDesCommandesExistantes(t *testing.T) {
	ctx := contexteBorne(t, 5*time.Minute)
	adresseMaitre := demarrerPostgreSQL(t)
	base := amorcerBaseDeService(ctx, t, adresseMaitre, baseTransactions, roleTransactionsMigr, roleTransactionsExec)
	migrerTransactionsJusqua(t, base.migration, versionAvantLeTitre)
	execution := connecter(ctx, t, base.execution)
	anciennes := []string{uuid.NewString(), uuid.NewString()}
	for _, id := range anciennes {
		if _, err := execution.Exec(ctx, insertionDeCommande, id, uuid.NewString(), uuid.NewString(), uuid.NewString(), time.Now().Add(-time.Hour)); err != nil {
			t.Fatalf("commande d'avant la migration : %v", err)
		}
	}
	migrerTransactions(t, construireBinaire(t, filepath.Join("..", "..", "..", "transactions", "cmd", "transactions"), "transactions"), base)
	for _, id := range anciennes {
		exigerCommandeRepriseAvecUnTitre(t, base.maitre, id)
	}
	t.Run("titre de 1 et de 200 caractères acceptés par le rôle d'exécution", func(t *testing.T) {
		for _, titre := range []string{"A", strings.Repeat("é", 200)} {
			if err := insererCommandeTitree(t, execution, titre); err != nil {
				t.Errorf("titre de %d caractères refusé : %v", len([]rune(titre)), err)
			}
		}
	})
	t.Run("titre vide ou de 201 caractères refusé par la base", func(t *testing.T) {
		for _, titre := range []string{"", strings.Repeat("a", 201)} {
			exigerViolationDeContrainte(t, insererCommandeTitree(t, execution, titre), fmt.Sprintf("titre de %d caractères", len([]rune(titre))))
		}
	})
	t.Run("titre obligatoire pour une nouvelle commande", func(t *testing.T) {
		var pgErr *pgconn.PgError
		_, err := execution.Exec(ctx, insertionDeCommande, uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), time.Now())
		if !errors.As(err, &pgErr) {
			t.Fatalf("commande sans titre acceptée après la migration")
		}
		t.Logf("commande sans titre : %s %s", pgErr.Code, pgErr.Message)
	})
}
