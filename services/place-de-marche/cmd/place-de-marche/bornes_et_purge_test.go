package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"

	"collector/services/place-de-marche/internal/migrations"
)

const (
	politiqueDesDemandes    = "travail-controle-demandes"
	livraisonsMaximales     = 5
	dureeDeVieReduite       = 5000
	longueurMaximaleReduite = 2
	dureeDeVie24h           = 86400000
	dureeDeVie7j            = 604800000
	echangeDesEcartes       = "collector.ecartes"
	raisonLimiteAtteinte    = "delivery_limit"
	raisonExpiration        = "expired"
	identiteTierce          = "tiers"
	evenementPurge          = "outbox.purge"
)

var filesDeTravail = []string{"controle.demandes", "pdm.resultats-controle", "interactions.publication-decidee"}

type fileListee struct {
	Nom       string         `json:"name"`
	Politique string         `json:"policy"`
	Effective map[string]any `json:"effective_policy_definition"`
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

func (c *canalDeTest) politiquesEffectives(t *testing.T) map[string]fileListee {
	t.Helper()
	var files []fileListee
	sortie := c.rabbitmqctl(t, "list_queues", "-p", vhostDuCanal, "name", "policy", "effective_policy_definition", "--formatter", "json")
	if err := json.Unmarshal(sortie, &files); err != nil {
		t.Fatalf("liste des files : %v : %s", err, sortie)
	}
	parNom := map[string]fileListee{}
	for _, f := range files {
		parNom[f.Nom] = f
	}
	return parNom
}

func exigerValeur(t *testing.T, file string, politique map[string]any, cle string, attendu any) {
	t.Helper()
	if politique[cle] != attendu {
		t.Errorf("%s : %s = %v, %v attendu (politique %v)", file, cle, politique[cle], attendu, politique)
	}
}

func (c *canalDeTest) prendreSansAcquitter(t *testing.T, file string) bool {
	t.Helper()
	connexion, err := amqp.Dial(c.adresse(identiteAdmin))
	if err != nil {
		t.Fatalf("connexion : %v", err)
	}
	defer func() { _ = connexion.Close() }()
	canal, err := connexion.Channel()
	if err != nil {
		t.Fatalf("canal : %v", err)
	}
	_, presente, err := canal.Get(file, false)
	if err != nil {
		t.Fatalf("lecture de %s : %v", file, err)
	}
	return presente
}

func (c *canalDeTest) premiereMiseALEcart(t *testing.T, file string) (amqp.Table, string) {
	t.Helper()
	livraison, presente, err := c.admin.Get(file, true)
	if err != nil || !presente {
		t.Fatalf("lecture de %s : présent %v, %v", file, presente, err)
	}
	morts, _ := livraison.Headers["x-death"].([]any)
	if len(morts) == 0 {
		t.Fatalf("message mis à l'écart sans en-tête x-death : %v", livraison.Headers)
	}
	mort, _ := morts[0].(amqp.Table)
	return mort, livraison.MessageId
}

func (c *canalDeTest) publierUneDemande(t *testing.T) (string, error) {
	t.Helper()
	id := uuid.NewString()
	return id, c.publier(t, identiteMarche, publication{echange: echangeDemandes, cle: cleDeRoutageV1, id: id, emetteur: identiteMarche, corps: []byte(`{"demande":"en attente"}`)})
}

func bornesReduites(topologie map[string]any) {
	for _, p := range topologie["policies"].([]any) {
		if politique := p.(map[string]any); politique["name"] == politiqueDesDemandes {
			definition := politique["definition"].(map[string]any)
			definition["message-ttl"] = dureeDeVieReduite
			definition["max-length"] = longueurMaximaleReduite
		}
	}
}

func TestUS024MC8BornesDesFilesDuCanal(t *testing.T) {
	canal := demarrerRabbitMQ(t)
	t.Run("MC8 politiques effectives des files de travail et des files .ecartes", canal.exigerPolitiquesEffectives)
	t.Run("US-024-AC3 MC8 message relivré sans fin mis à l'écart après 5 livraisons", canal.exigerMiseALEcartApresCinqLivraisons)
	reduit := demarrerRabbitMQ(t, bornesReduites)
	t.Run("MC8 message expiré mis à l'écart, jamais perdu en silence", reduit.exigerExpirationMiseALEcart)
	t.Run("MC8 file pleine : publication refusée au producteur sans perte des messages en file", reduit.exigerFilePleineRefuseeSansPerte)
}

func (c *canalDeTest) exigerPolitiquesEffectives(t *testing.T) {
	files := c.politiquesEffectives(t)
	for _, nom := range filesDeTravail {
		travail, ecartes := files[nom], files[nom+".ecartes"]
		if travail.Nom == "" || ecartes.Nom == "" {
			t.Fatalf("file %s ou sa file .ecartes absente", nom)
		}
		exigerValeur(t, nom, travail.Effective, "delivery-limit", float64(livraisonsMaximales-1))
		exigerValeur(t, nom, travail.Effective, "overflow", "reject-publish")
		exigerValeur(t, nom, travail.Effective, "message-ttl", float64(dureeDeVie24h))
		exigerValeur(t, nom, travail.Effective, "dead-letter-exchange", echangeDesEcartes)
		exigerValeur(t, nom, travail.Effective, "dead-letter-routing-key", nom)
		exigerValeur(t, nom, travail.Effective, "dead-letter-strategy", "at-least-once")
		if longueur, _ := travail.Effective["max-length"].(float64); longueur <= 0 {
			t.Errorf("%s : longueur maximale %v", nom, travail.Effective["max-length"])
		}
		exigerValeur(t, ecartes.Nom, ecartes.Effective, "message-ttl", float64(dureeDeVie7j))
		exigerValeur(t, ecartes.Nom, ecartes.Effective, "overflow", "reject-publish")
	}
}

func (c *canalDeTest) livrerSansAcquitterJusquALEcart(t *testing.T) int {
	t.Helper()
	livraisons := 0
	for c.compter(t, fileDemandesEcartees) == 0 && livraisons <= 2*livraisonsMaximales {
		pris := false
		attendre(t, 10*time.Second, "retour du message ou mise à l'écart", func() bool {
			pris = c.prendreSansAcquitter(t, fileDemandes)
			return pris || c.compter(t, fileDemandesEcartees) == 1
		})
		if pris {
			livraisons++
		}
	}
	return livraisons
}

func (c *canalDeTest) exigerMiseALEcartApresCinqLivraisons(t *testing.T) {
	id, err := c.publierUneDemande(t)
	if err != nil {
		t.Fatalf("publication : %v", err)
	}
	if livraisons := c.livrerSansAcquitterJusquALEcart(t); livraisons != livraisonsMaximales {
		t.Errorf("%d livraison(s) avant la mise à l'écart, %d attendues", livraisons, livraisonsMaximales)
	}
	mort, ecarte := c.premiereMiseALEcart(t, fileDemandesEcartees)
	if ecarte != id || mort["reason"] != raisonLimiteAtteinte {
		t.Errorf("message %s mis à l'écart pour %v, %s pour %s attendu", ecarte, mort["reason"], id, raisonLimiteAtteinte)
	}
	if n := c.compter(t, fileDemandes); n != 0 {
		t.Errorf("%d message(s) restant(s) dans %s", n, fileDemandes)
	}
}

func (c *canalDeTest) exigerExpirationMiseALEcart(t *testing.T) {
	id, err := c.publierUneDemande(t)
	if err != nil {
		t.Fatalf("publication : %v", err)
	}
	attendre(t, 30*time.Second, "mise à l'écart du message expiré", func() bool { return c.compter(t, fileDemandesEcartees) == 1 })
	mort, ecarte := c.premiereMiseALEcart(t, fileDemandesEcartees)
	if ecarte != id || mort["reason"] != raisonExpiration {
		t.Errorf("message %s mis à l'écart pour %v", ecarte, mort["reason"])
	}
}

func (c *canalDeTest) exigerFilePleineRefuseeSansPerte(t *testing.T) {
	acceptes := map[string]bool{}
	var refus error
	for range longueurMaximaleReduite + 5 {
		id, err := c.publierUneDemande(t)
		if err != nil {
			refus = err
			break
		}
		acceptes[id] = true
	}
	t.Logf("%d publication(s) acceptée(s) pour une longueur maximale de %d, puis : %v", len(acceptes), longueurMaximaleReduite, refus)
	if refus == nil {
		t.Fatalf("aucune publication refusée au-delà de la longueur maximale %d", longueurMaximaleReduite)
	}
	if n := c.compter(t, fileDemandes); n != len(acceptes) {
		t.Errorf("%d message(s) en file pour %d accepté(s) : un message accepté a été supprimé", n, len(acceptes))
	}
	attendre(t, 30*time.Second, "mise à l'écart des messages acceptés à expiration", func() bool {
		return c.compter(t, fileDemandesEcartees) == len(acceptes)
	})
	for range acceptes {
		if _, ecarte := c.premiereMiseALEcart(t, fileDemandesEcartees); !acceptes[ecarte] {
			t.Errorf("message %s inattendu dans %s", ecarte, fileDemandesEcartees)
		}
	}
}

type bancDePurge struct {
	base     basePostgreSQL
	pool     *pgxpool.Pool
	demandes []string
}

func TestUS024MC8PurgeDeLOutboxEtDroits(t *testing.T) {
	e, base, pool := demarrerSurPostgreSQL(t)
	b := &bancDePurge{base: base, pool: pool}
	jeton := e.vendeur()
	for i, publieLe := range []string{"now() - interval '8 days'", "now() - interval '7 days 1 minute'", "now() - interval '6 days 23 hours'", "NULL"} {
		soumis := e.soumettreAvecSucces(t, jeton, e.creer(t, jeton, corpsComplet("Purge "+publieLe, categorieSure, prixCourant, 0)))
		var id string
		err := base.maitre.QueryRow(t.Context(), `
			UPDATE outbox SET publie_le = `+publieLe+`, cree_le = now() - make_interval(days => 30 - $2::integer)
			WHERE convert_from(corps, 'UTF8')::jsonb -> 'data' ->> 'article_id' = $1
			RETURNING id::text`, soumis.ID, i).Scan(&id)
		if err != nil {
			t.Fatalf("préparation de l'outbox : %v", err)
		}
		b.demandes = append(b.demandes, id)
	}
	t.Run("MC8 rôle d'exécution : purge des seules lignes publiées depuis plus de 7 jours", b.purgeDesSeulesLignesPublieesDepuisPlusDe7Jours)
	t.Run("MC8 rôle d'exécution sans DELETE sur l'outbox", b.sansDeleteSurLOutbox)
	t.Run("MC8 fonction de purge SECURITY DEFINER exécutable par le seul rôle d'exécution", b.fonctionReserveeAuRoleDExecution)
}

func (b *bancDePurge) presentes(t *testing.T) []string {
	t.Helper()
	var ids []string
	lignes, err := b.base.maitre.Query(t.Context(), `SELECT id::text FROM outbox ORDER BY cree_le`)
	if err != nil {
		t.Fatalf("outbox : %v", err)
	}
	defer lignes.Close()
	for lignes.Next() {
		var id string
		if err := lignes.Scan(&id); err != nil {
			t.Fatalf("outbox : %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

func (b *bancDePurge) purger(t *testing.T) int64 {
	t.Helper()
	var purgees int64
	if err := b.pool.QueryRow(t.Context(), `SELECT purger_outbox_publiee()`).Scan(&purgees); err != nil {
		t.Fatalf("purge par le rôle d'exécution : %v", err)
	}
	return purgees
}

func (b *bancDePurge) purgeDesSeulesLignesPublieesDepuisPlusDe7Jours(t *testing.T) {
	purgees := b.purger(t)
	if restantes := b.presentes(t); purgees != 2 || !slices.Equal(restantes, b.demandes[2:]) {
		t.Fatalf("%d ligne(s) purgée(s), restantes %v ; attendu 2, restantes %v", purgees, restantes, b.demandes[2:])
	}
	if purgees := b.purger(t); purgees != 0 {
		t.Fatalf("seconde purge : %d ligne(s)", purgees)
	}
}

func (b *bancDePurge) sansDeleteSurLOutbox(t *testing.T) {
	exigerRefusDeDroit(t, b.pool, "DELETE FROM outbox")
	exigerRefusDeDroit(t, b.pool, "TRUNCATE outbox")
	if restantes := b.presentes(t); !slices.Equal(restantes, b.demandes[2:]) {
		t.Fatalf("outbox modifiée : %v", restantes)
	}
}

func (b *bancDePurge) fonctionReserveeAuRoleDExecution(t *testing.T) {
	var definisseur, execution, public bool
	err := b.base.maitre.QueryRow(t.Context(), `
		SELECT p.prosecdef,
		       has_function_privilege($1, p.oid, 'EXECUTE'),
		       coalesce(bool_or(a.grantee = 0 AND a.privilege_type = 'EXECUTE'), false)
		FROM pg_proc p LEFT JOIN LATERAL aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a ON true
		WHERE p.proname = 'purger_outbox_publiee'
		GROUP BY p.prosecdef, p.oid`, roleExecution).Scan(&definisseur, &execution, &public)
	if err != nil {
		t.Fatalf("lecture de la fonction : %v", err)
	}
	if !definisseur || !execution || public {
		t.Fatalf("SECURITY DEFINER %v, EXECUTE au rôle d'exécution %v, EXECUTE à PUBLIC %v", definisseur, execution, public)
	}
	secret := secretAleatoire(t)
	executerFormate(t.Context(), t, b.base.maitre, "CREATE ROLE %I LOGIN PASSWORD %L", identiteTierce, secret)
	executerFormate(t.Context(), t, b.base.maitre, "GRANT CONNECT ON DATABASE %I TO %I", baseDuService, identiteTierce)
	tiers := connecter(t.Context(), t, avecIdentite(t, b.base.execution, identiteTierce, secret))
	_, err = tiers.Exec(t.Context(), `SELECT purger_outbox_publiee()`)
	var refus *pgconn.PgError
	if !errors.As(err, &refus) || refus.Code != droitInsuffisant {
		t.Fatalf("purge par un rôle tiers : %v, refus 42501 attendu", err)
	}
}

func TestUS024MC8PurgeAuDemarrageDuService(t *testing.T) {
	ctx := contexteBorne(t, 3*time.Minute)
	base := amorcer(ctx, t, demarrerPostgreSQL(t))
	if err := migrations.Apply(ctx, journalMuet(), base.migration); err != nil {
		t.Fatalf("migrations : %v", err)
	}
	for _, publieLe := range []string{"8 days", "10 days", "1 day"} {
		_, err := base.maitre.Exec(ctx, `
			INSERT INTO outbox (id, echange, cle_routage, type_message, correlation_id, corps, cree_le, publie_le)
			VALUES (gen_random_uuid(), 'controle.demandes', 'v1', 'controle.demande', gen_random_uuid(), '{}'::bytea, now() - $1::interval, now() - $1::interval)`, publieLe)
		if err != nil {
			t.Fatalf("ligne d'outbox ancienne : %v", err)
		}
	}
	_, journal := lancerPlaceDeMarche(t, demarrerRabbitMQ(t), base.execution, nouvelEmetteurDeJetons(t))
	attendre(t, 10*time.Second, "purge des lignes publiées depuis plus de 7 jours au démarrage", func() bool {
		var restantes int
		if err := base.maitre.QueryRow(t.Context(), `SELECT count(*) FROM outbox`).Scan(&restantes); err != nil {
			t.Fatalf("outbox : %v", err)
		}
		return restantes == 1
	})
	traces := evenementsJournalises(journal, evenementPurge)
	if len(traces) != 1 || traces[0]["lignes"] != float64(2) {
		t.Fatalf("traces de purge %v, une trace à 2 lignes attendue", traces)
	}
}

func avecIdentite(t *testing.T, adresse, role, secret string) string {
	t.Helper()
	cible, err := url.Parse(adresse)
	if err != nil {
		t.Fatalf("adresse : %v", err)
	}
	cible.User = url.UserPassword(role, secret)
	return cible.String()
}
