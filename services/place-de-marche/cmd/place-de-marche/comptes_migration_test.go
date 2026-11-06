package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/annonces/controle"
	"collector/services/place-de-marche/internal/annonces/stockage"
	comptesstockage "collector/services/place-de-marche/internal/comptes/stockage"
	"collector/services/place-de-marche/internal/migrations"
)

const versionAvantLesComptes = 4

type vendeurHistorique struct {
	sujet    string
	articles []string
	inscrit  time.Time
}

type compteRepris struct {
	inscrit    time.Time
	suspendu   bool
	suspenduLe *time.Time
}

func instant(t *testing.T, valeur string) time.Time {
	t.Helper()
	moment, err := time.Parse(time.RFC3339Nano, valeur)
	if err != nil {
		t.Fatalf("instant %q : %v", valeur, err)
	}
	return moment
}

func vendeursHistoriques(t *testing.T) []vendeurHistorique {
	return []vendeurHistorique{
		{sujet: uuid.NewString(), articles: []string{"2025-06-01T08:00:00Z", "2025-03-14T10:00:00Z", "2025-09-30T12:00:00Z"}, inscrit: instant(t, "2025-03-14T10:00:00Z")},
		{sujet: uuid.NewString(), articles: []string{"2025-07-15T23:30:00-02:00"}, inscrit: instant(t, "2025-07-16T01:30:00Z")},
		{sujet: uuid.NewString(), articles: []string{"2026-01-31T23:59:59.999999Z", "2026-02-01T00:00:00Z"}, inscrit: instant(t, "2026-01-31T23:59:59.999999Z")},
	}
}

func migrerJusqua(t *testing.T, adresse string, version int64) {
	t.Helper()
	sources := os.DirFS(filepath.Join("..", "..", "internal", "migrations"))
	noms, err := fs.Glob(sources, "*.sql")
	if err != nil || len(noms) == 0 {
		t.Fatalf("scripts de migration : %v", err)
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
		t.Fatalf("migrations jusqu'à la version %d : %v", version, err)
	}
}

func insererLesArticlesHistoriques(t *testing.T, conn *pgx.Conn, vendeurs []vendeurHistorique) {
	t.Helper()
	for _, v := range vendeurs {
		for _, creeLe := range v.articles {
			_, err := conn.Exec(t.Context(), `
				INSERT INTO article (vendeur_id, etat, revision, titre, description, categorie_id, prix_centimes, cree_le, modifie_le)
				VALUES ($1, 'brouillon', 1, 'Figurine historique', 'Pièce ancienne', $2, 4500, $3::timestamptz, $3::timestamptz)`,
				v.sujet, categorieSure, creeLe)
			if err != nil {
				t.Fatalf("article historique de %s : %v", v.sujet, err)
			}
		}
	}
}

func comptesRepris(t *testing.T, conn *pgx.Conn) map[string]compteRepris {
	t.Helper()
	lignes, err := conn.Query(t.Context(), `SELECT id::text, inscrit_le, suspendu, suspendu_le FROM compte`)
	if err != nil {
		t.Fatalf("lecture des comptes : %v", err)
	}
	defer lignes.Close()
	comptes := map[string]compteRepris{}
	for lignes.Next() {
		var id string
		var c compteRepris
		if err := lignes.Scan(&id, &c.inscrit, &c.suspendu, &c.suspenduLe); err != nil {
			t.Fatalf("compte : %v", err)
		}
		comptes[id] = c
	}
	return comptes
}

func exigerComptesRepris(t *testing.T, conn *pgx.Conn, vendeurs []vendeurHistorique) {
	t.Helper()
	comptes := comptesRepris(t, conn)
	if len(comptes) != len(vendeurs) {
		t.Fatalf("%d compte(s) repris, %d attendu(s) : %+v", len(comptes), len(vendeurs), comptes)
	}
	for _, v := range vendeurs {
		c, present := comptes[v.sujet]
		if !present {
			t.Fatalf("vendeur %s non repris", v.sujet)
		}
		if !c.inscrit.Equal(v.inscrit) || c.suspendu || c.suspenduLe != nil {
			t.Errorf("vendeur %s repris %+v, inscrit le %s non suspendu attendu", v.sujet, c, v.inscrit)
		}
	}
}

func bancSansCanal(t *testing.T, base basePostgreSQL) *bancComplet {
	t.Helper()
	pool, err := stockage.Connect(t.Context(), base.execution)
	if err != nil {
		t.Fatalf("connexion en %s : %v", roleExecution, err)
	}
	t.Cleanup(pool.Close)
	depot := stockage.NewStore(pool, controle.NewRequestEncoder(validateurDeContrats(t)), comptesstockage.NewStore(pool))
	return &bancComplet{environnement: demarrer(t, annonces.NewService(depot, depot)), base: base, validateur: validateurDeContrats(t)}
}

func TestUS019Lot2MigrationInscritLesVendeursExistantsALaDateDeLeurPremierArticle(t *testing.T) {
	ctx := contexteBorne(t, 3*time.Minute)
	base := amorcer(ctx, t, demarrerPostgreSQL(t))
	migrerJusqua(t, base.migration, versionAvantLesComptes)
	vendeurs := vendeursHistoriques(t)
	insererLesArticlesHistoriques(t, connecter(ctx, t, base.migration), vendeurs)
	for range 2 {
		if err := migrations.Apply(ctx, journalMuet(), base.migration); err != nil {
			t.Fatalf("migrations : %v", err)
		}
	}

	t.Run("un compte par vendeur existant, inscrit au premier article, non suspendu", func(t *testing.T) {
		exigerComptesRepris(t, base.maitre, vendeurs)
	})

	t.Run("AC4 vendeur repris : sa demande porte la date UTC de son premier article, non suspendu", func(t *testing.T) {
		b := bancSansCanal(t, base)
		for _, v := range vendeurs {
			vendeur := vendeurAuSujet(b.environnement, v.sujet)
			soumis := b.soumettreAvecSucces(t, vendeur.jeton, b.creer(t, vendeur.jeton, corpsComplet("Figurine d'un vendeur repris", categorieSure, prixSousLaVigilance, 0)))
			exigerVendeurTransmis(t, b.vendeurTransmis(t, soumis), vendeurTransmis{AccountID: v.sujet, MemberSince: jourUTC(v.inscrit), Suspended: false})
		}
		exigerComptesRepris(t, base.maitre, vendeurs)
	})
}
