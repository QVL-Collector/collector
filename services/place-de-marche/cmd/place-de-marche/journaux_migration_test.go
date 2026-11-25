package main

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"collector/services/place-de-marche/internal/migrations"
)

const (
	versionAvantLaTrace = 9
	traceparentValide   = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
)

var traceparentsHorsFormat = []string{
	"",
	"pas-un-traceparent",
	"00-4BF92F3577B34DA6A3CE929D0E0E4736-00f067aa0ba902b7-01",
	"00-4bf92f3577b34da6a3ce929d0e0e473-00f067aa0ba902b7-01",
	traceparentValide + "\n{\"audit\":true}",
}

type ligneHeritee struct {
	id          string
	correlation string
	corps       []byte
	creeLe      time.Time
}

type bancDeMigration struct {
	base    basePostgreSQL
	heritee ligneHeritee
}

func (b *bancDeMigration) insererLigneHeritee(t *testing.T) {
	t.Helper()
	b.heritee = ligneHeritee{id: uuid.NewString(), correlation: uuid.NewString(), corps: []byte(`{"data":{"article_id":"herite"}}`)}
	if _, err := b.base.maitre.Exec(t.Context(), `INSERT INTO outbox (id, echange, cle_routage, type_message, correlation_id, corps) VALUES ($1, $2, $3, $4, $5, $6)`,
		b.heritee.id, echangeDemandes, cleDeRoutageV1, typeDemande, b.heritee.correlation, b.heritee.corps); err != nil {
		t.Fatalf("ligne d'outbox avant la migration 00010 : %v", err)
	}
	if err := b.base.maitre.QueryRow(t.Context(), `SELECT cree_le FROM outbox WHERE id = $1`, b.heritee.id).Scan(&b.heritee.creeLe); err != nil {
		t.Fatalf("relecture : %v", err)
	}
}

func (b *bancDeMigration) insererTracee(t *testing.T, traceparent *string) error {
	t.Helper()
	_, err := b.base.maitre.Exec(t.Context(), `INSERT INTO outbox (id, echange, cle_routage, type_message, correlation_id, corps, traceparent) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.NewString(), echangeDemandes, cleDeRoutageV1, typeDemande, uuid.NewString(), []byte(`{}`), traceparent)
	return err
}

func (b *bancDeMigration) tracerUnArticle(t *testing.T, traceparent *string) error {
	t.Helper()
	_, err := b.base.maitre.Exec(t.Context(), `UPDATE article SET demande_controle_correlation_id = $1, demande_controle_traceparent = $2 WHERE id = (SELECT id FROM article LIMIT 1)`, uuid.NewString(), traceparent)
	return err
}

func exigerViolationDeControle(t *testing.T, err error, valeur string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != violationDeControle {
		t.Errorf("traceparent %q accepté (erreur %v), refus attendu", valeur, err)
	}
}

func (b *bancDeMigration) ligneEnAttenteConservee(t *testing.T) {
	var relue ligneHeritee
	var traceparent *string
	var publiee bool
	if err := b.base.maitre.QueryRow(t.Context(), `SELECT correlation_id::text, corps, cree_le, traceparent, publie_le IS NOT NULL FROM outbox WHERE id = $1`, b.heritee.id).
		Scan(&relue.correlation, &relue.corps, &relue.creeLe, &traceparent, &publiee); err != nil {
		t.Fatalf("relecture après migration : %v", err)
	}
	if relue.correlation != b.heritee.correlation || string(relue.corps) != string(b.heritee.corps) || !relue.creeLe.Equal(b.heritee.creeLe) || publiee || traceparent != nil {
		t.Fatalf("ligne héritée modifiée : %+v, traceparent %v, publiée %v", relue, traceparent, publiee)
	}
}

func (b *bancDeMigration) traceparentW3COuAbsentAccepte(t *testing.T) {
	valide := traceparentValide
	if err := b.insererTracee(t, &valide); err != nil {
		t.Errorf("traceparent W3C refusé : %v", err)
	}
	if err := b.insererTracee(t, nil); err != nil {
		t.Errorf("traceparent absent refusé : %v", err)
	}
}

func (b *bancDeMigration) traceparentHorsFormatRefuse(t *testing.T) {
	for _, valeur := range traceparentsHorsFormat {
		exigerViolationDeControle(t, b.insererTracee(t, &valeur), valeur)
	}
}

func (b *bancDeMigration) roleDExecutionEcritLaColonne(t *testing.T) {
	conn := connecter(t.Context(), t, b.base.execution)
	defer func() { _ = conn.Close(t.Context()) }()
	var traceparent *string
	if err := conn.QueryRow(t.Context(), `SELECT traceparent FROM outbox WHERE id = $1`, b.heritee.id).Scan(&traceparent); err != nil {
		t.Fatalf("lecture par marche_execution : %v", err)
	}
	if _, err := conn.Exec(t.Context(), `INSERT INTO outbox (id, echange, cle_routage, type_message, correlation_id, corps, traceparent) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.NewString(), echangeDemandes, cleDeRoutageV1, typeDemande, uuid.NewString(), []byte(`{}`), traceparentValide); err != nil {
		t.Fatalf("écriture par marche_execution : %v", err)
	}
}

func (b *bancDeMigration) articleTraceContraint(t *testing.T) {
	if _, err := b.base.maitre.Exec(t.Context(), `INSERT INTO article (vendeur_id, etat, revision, titre, description) VALUES ($1, 'brouillon', 1, 'Figurine', 'Belle pièce')`, uuid.NewString()); err != nil {
		t.Fatalf("article de test : %v", err)
	}
	valide := traceparentValide
	if err := b.tracerUnArticle(t, &valide); err != nil {
		t.Errorf("traceparent W3C refusé sur l'article : %v", err)
	}
	if err := b.tracerUnArticle(t, nil); err != nil {
		t.Errorf("traceparent absent refusé sur l'article : %v", err)
	}
	for _, valeur := range traceparentsHorsFormat {
		exigerViolationDeControle(t, b.tracerUnArticle(t, &valeur), valeur)
	}
}

func TestUS014MigrationsDeTraceLigneEnAttenteConserveeEtTraceparentContraint(t *testing.T) {
	ctx := contexteBorne(t, 3*time.Minute)
	b := &bancDeMigration{base: amorcer(ctx, t, demarrerPostgreSQL(t))}
	migrerJusqua(t, b.base.migration, versionAvantLaTrace)
	b.insererLigneHeritee(t)

	if err := migrations.Apply(ctx, journalMuet(), b.base.migration); err != nil {
		t.Fatalf("migrations 00010 et suivantes : %v", err)
	}

	t.Run("00010 ligne d'outbox en attente conservée à l'identique, traceparent vide", b.ligneEnAttenteConservee)
	t.Run("00010 traceparent W3C accepté, absent accepté", b.traceparentW3COuAbsentAccepte)
	t.Run("00010 traceparent hors format W3C refusé par la base", b.traceparentHorsFormatRefuse)
	t.Run("00010 le rôle d'exécution lit et écrit la colonne", b.roleDExecutionEcritLaColonne)
	t.Run("00011 traceparent de la demande gardé sur l'article : W3C ou absent accepté, hors format refusé", b.articleTraceContraint)
}
