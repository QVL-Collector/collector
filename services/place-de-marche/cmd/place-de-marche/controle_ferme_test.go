package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"

	"collector/libs/platform/logging"
)

const (
	delaiDuFilet          = 50 * time.Second
	delaiDeReprise        = 60 * time.Second
	delaiDAlerte          = 2 * time.Minute
	tailleHorsContrat     = 70 * 1024
	tailleHorsBroker      = 130 * 1024
	conservationOutbox    = 7 * 24 * time.Hour
	evenementEcarte       = "canal.message_ecarte"
	evenementRelance      = "controle.demande_relancee"
	evenementIgnore       = "controle.resultat_ignore"
	evenementRejeu        = "outbox.rejeu"
	raisonRejetConsomm    = "rejected"
	prixCourant           = 2500
	articlesSoumisPendant = 3
)

const requeteI3 = `
WITH demande AS (
    SELECT (convert_from(o.corps, 'UTF8')::jsonb #>> '{data,article_id}')::uuid AS article_id,
           (convert_from(o.corps, 'UTF8')::jsonb #>> '{data,revision}')::integer AS revision,
           count(*) AS envois,
           bool_and(o.publie_le IS NULL) AS jamais_publiee
    FROM outbox AS o
    WHERE o.type_message = 'controle.demande'
    GROUP BY 1, 2
)
SELECT count(*)                                                                 AS demandes_emises,
       coalesce(sum(d.envois), 0) - count(*)                                    AS envois_relances,
       count(*) FILTER (WHERE d.jamais_publiee)                                 AS demandes_non_publiees,
       count(r.message_id)                                                      AS resultats_appliques,
       count(*) FILTER (WHERE r.message_id IS NULL AND a.revision > d.revision) AS demandes_depassees,
       count(*) FILTER (WHERE r.message_id IS NULL AND a.revision = d.revision) AS demandes_sans_resultat
FROM demande AS d
JOIN article AS a ON a.id = d.article_id
LEFT JOIN resultat_controle AS r ON r.article_id = d.article_id AND r.revision = d.revision`

type indicateurI3 struct {
	emises       int64
	relances     int64
	nonPubliees  int64
	appliques    int64
	depassees    int64
	sansResultat int64
}

type ligneBrute struct {
	contenu  string
	publieLe *time.Time
}

type bancControleFerme struct {
	*bancComplet
	binaireControle string
	controle        *processusControle
	execution       *pgx.Conn
	dejaPublie      articleRecu
	soumis          []articleRecu
	modifie         articleRecu
	relance         articleRecu
	temoinRecent    articleRecu
}

func releverI3(t *testing.T, conn *pgx.Conn) indicateurI3 {
	t.Helper()
	var i indicateurI3
	if err := conn.QueryRow(t.Context(), requeteI3).Scan(&i.emises, &i.relances, &i.nonPubliees, &i.appliques, &i.depassees, &i.sansResultat); err != nil {
		t.Fatalf("requête I3 : %v", err)
	}
	return i
}

func evenementsJournalises(j *journalPartage, nom string) []map[string]any {
	j.mu.Lock()
	defer j.mu.Unlock()
	var trouves []map[string]any
	lecteur := bufio.NewScanner(strings.NewReader(j.tampon.String()))
	lecteur.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for lecteur.Scan() {
		var ligne map[string]any
		if json.Unmarshal(lecteur.Bytes(), &ligne) == nil && ligne["event"] == nom {
			trouves = append(trouves, ligne)
		}
	}
	return trouves
}

func evenementsPortant(j *journalPartage, nom, valeur string) []map[string]any {
	var trouves []map[string]any
	for _, e := range evenementsJournalises(j, nom) {
		if slices.Contains(feuilles(e), valeur) {
			trouves = append(trouves, e)
		}
	}
	return trouves
}

func TestUS024ControleFermeParDefautEtRepriseSansPerte(t *testing.T) {
	b := &bancControleFerme{bancComplet: demarrerBancComplet(t)}
	b.binaireControle = construireControle(t)
	b.execution = connecter(t.Context(), t, b.base.execution)
	b.controle = controleDisponible(t, b.binaireControle, b.canal)
	b.dejaPublie, _ = b.attendreEtat(t, b.creerEtSoumettre(t, corpsComplet("Figurine Lucky Luke", categorieSure, 4000, 0)).ID, etatPublie, delaiDeTraitement)
	b.controle.arreter()
	t.Run("US-024-AC1 articles soumis pendant l'arrêt du Contrôle restent en contrôle et hors du catalogue", b.ac1SoumisPendantLArret)
	t.Run("US-024-AC1 article publié modifié pendant l'arrêt quitte le catalogue et reste en contrôle", b.ac1PublieModifiePendantLArret)
	t.Run("US-024-AC1 filet de relance : nouvelle demande pour la même révision, article toujours en contrôle", b.ac1FiletDeRelance)
	t.Run("US-024-AC4 I3 pendant l'arrêt : chaque demande en attente comptée sans résultat", b.ac4I3PendantLArret)
	b.controle = controleDisponible(t, b.binaireControle, b.canal)
	t.Run("US-024-AC2 au retour du Contrôle chaque article soumis pendant l'arrêt est contrôlé et publié une seule fois", b.ac2ReprisePublieChaqueArticle)
	t.Run("US-024-AC2 une seule issue par révision malgré la demande relancée", b.ac2UneSeuleIssueMalgreLaRelance)
	t.Run("US-024-AC2 aucune demande ni aucun résultat perdu ou mis à l'écart", b.ac2AucunePerte)
	t.Run("US-024-AC4 I3 en fin de test : demandes émises égales aux résultats appliqués", b.ac4I3EnFinDeTest)
	t.Run("MC9 rejeu refusé hors fenêtre sans aucune écriture", b.mc9RejeuRefuse)
	t.Run("MC9 rejeu de la fenêtre : seule la date de publication réécrite, republié sans double effet, tracé", b.mc9RejeuNominal)
	t.Run("MC9 rejeu en limite de conservation accepté", b.mc9RejeuEnLimiteDeConservation)
	t.Run("US-024-AC3 MC6 demandes non conformes écartées dès la première livraison sans bloquer la suivante", b.ac3DemandesInvalidesEcartees)
	t.Run("US-024-AC3 MC6 résultats non conformes écartés dès la première livraison sans bloquer le suivant", b.ac3ResultatsInvalidesEcartes)
	t.Run("MC6 message au-delà de la taille admise par le broker refusé à la publication", b.mc6MessageTropGrandRefuseParLeBroker)
}

func (b *bancControleFerme) articlesEnAttente() []articleRecu {
	return append(append([]articleRecu{}, b.soumis...), b.modifie, b.temoinRecent)
}

func (b *bancControleFerme) exigerEnControleHorsCatalogue(t *testing.T, articles ...articleRecu) {
	t.Helper()
	catalogue := b.toutLeCatalogue(t, "")
	exigerAbsents(t, catalogue, articles...)
	for _, a := range articles {
		relu := b.relire(t, b.jeton, a.ID)
		exigerEtat(t, relu, etatEnControle, "")
		if relu.Revision != a.Revision {
			t.Errorf("article %s : révision %d, %d attendue", a.ID, relu.Revision, a.Revision)
		}
		exigerEtat(t, b.dansMesArticles(t, b.jeton, a.ID), etatEnControle, "")
		b.fiche(t, a.ID).exiger(t, http.StatusNotFound)
		exigerAbsents(t, b.toutLeCatalogue(t, a.Categorie.ID), a)
	}
}

func (b *bancControleFerme) ac1SoumisPendantLArret(t *testing.T) {
	for _, c := range b.categories(t)[:articlesSoumisPendant] {
		b.soumis = append(b.soumis, b.creerEtSoumettre(t, corpsComplet("Pendant l'arrêt "+c.Libelle, c.ID, prixCourant, 0)))
	}
	b.exigerMessages(t, fileDemandes, articlesSoumisPendant)
	for _, a := range b.soumis {
		if lignes := b.outboxDe(t, a.ID); len(lignes) != 1 || !lignes[0].publiee {
			t.Fatalf("demande de %s : %+v", a.ID, lignes)
		}
	}
	time.Sleep(3 * time.Second)
	b.exigerEnControleHorsCatalogue(t, b.soumis...)
	if n := len(b.resultatsEnregistres(t, b.soumis[0].ID)); n != 0 {
		t.Fatalf("%d résultat(s) appliqué(s) Contrôle arrêté", n)
	}
}

func (b *bancControleFerme) ac1PublieModifiePendantLArret(t *testing.T) {
	exigerPresents(t, b.toutLeCatalogue(t, ""), b.dejaPublie)
	b.modifie = b.modifierAvecSucces(t, b.dejaPublie, corpsComplet("Figurine Lucky Luke et Jolly Jumper", categorieSure, 4000, 0))
	exigerEtat(t, b.modifie, etatEnControle, "")
	b.exigerMessages(t, fileDemandes, articlesSoumisPendant+1)
	time.Sleep(2 * time.Second)
	b.exigerEnControleHorsCatalogue(t, b.modifie)
}

func (b *bancControleFerme) ac1FiletDeRelance(t *testing.T) {
	b.relance = b.soumis[0]
	b.temoinRecent = b.creerEtSoumettre(t, corpsComplet("Témoin récent", categorieSure, prixCourant, 0))
	if _, err := b.base.maitre.Exec(t.Context(), `UPDATE article SET demande_controle_le = now() - interval '3 minutes' WHERE id = $1`, b.relance.ID); err != nil {
		t.Fatalf("vieillissement de la demande : %v", err)
	}
	duree := attendre(t, delaiDuFilet, "relance de la demande par le filet", func() bool { return len(b.outboxDe(t, b.relance.ID)) == 2 })
	t.Logf("demande relancée %s après le vieillissement", duree.Round(time.Second))
	lignes := b.outboxDe(t, b.relance.ID)
	var identifiants []string
	for _, l := range lignes {
		var demande demandeRecue
		enveloppe, err := b.validateur.Decode(l.corps, &demande)
		if err != nil {
			t.Fatalf("demande relancée non conforme au contrat v1 : %v", err)
		}
		if l.echange != echangeDemandes || l.typ != typeDemande || demande.ArticleID != b.relance.ID || demande.Revision != b.relance.Revision {
			t.Fatalf("demande %+v sur %q pour %s en révision %d", demande, l.echange, b.relance.ID, b.relance.Revision)
		}
		identifiants = append(identifiants, enveloppe.ID)
	}
	if identifiants[0] == identifiants[1] {
		t.Errorf("demande relancée sous le même identifiant de message %s", identifiants[0])
	}
	attendre(t, delaiDeTraitement, "publication de la demande relancée", func() bool { return b.outboxDe(t, b.relance.ID)[1].publiee })
	if len(evenementsPortant(b.journal, evenementRelance, b.relance.ID)) == 0 {
		t.Errorf("aucun journal %s pour %s", evenementRelance, b.relance.ID)
	}
	if n := len(b.outboxDe(t, b.temoinRecent.ID)); n != 1 {
		t.Errorf("article soumis il y a moins de 2 min relancé : %d demande(s)", n)
	}
	b.exigerEnControleHorsCatalogue(t, b.relance, b.temoinRecent)
}

func (b *bancControleFerme) ac4I3PendantLArret(t *testing.T) {
	i3 := releverI3(t, b.base.maitre)
	enAttente := int64(len(b.articlesEnAttente()))
	attendu := indicateurI3{emises: enAttente + 1, nonPubliees: 0, appliques: 1, depassees: 0, sansResultat: enAttente}
	attendu.relances = i3.relances
	if i3 != attendu || i3.relances < 1 {
		t.Fatalf("I3 Contrôle arrêté : %+v, attendu %+v avec au moins une relance", i3, attendu)
	}
}

func (b *bancControleFerme) ac2ReprisePublieChaqueArticle(t *testing.T) {
	for _, a := range b.articlesEnAttente() {
		_, duree := b.attendreEtat(t, a.ID, etatPublie, delaiDeReprise)
		t.Logf("%s publié %s après la relance du Contrôle", a.ID, duree.Round(time.Second))
	}
	catalogue := b.toutLeCatalogue(t, "")
	exigerPresents(t, catalogue, b.articlesEnAttente()...)
	exigerSansDoublon(t, catalogue)
	for _, a := range b.articlesEnAttente() {
		relu := b.relire(t, b.jeton, a.ID)
		if relu.Revision != a.Revision {
			t.Errorf("%s : révision %d après la reprise, %d attendue", a.ID, relu.Revision, a.Revision)
		}
		b.fiche(t, a.ID).exiger(t, http.StatusOK)
	}
}

func (b *bancControleFerme) ac2UneSeuleIssueMalgreLaRelance(t *testing.T) {
	attendre(t, delaiDeReprise, "deux résultats publiés par le Contrôle pour la demande relancée", func() bool {
		b.releverLesResultats(t)
		return len(b.captures[b.relance.ID]) >= 2
	})
	attendre(t, delaiDeTraitement, "doublon de résultat ignoré et journalisé", func() bool {
		return len(evenementsPortant(b.journal, evenementIgnore, b.relance.ID)) >= 1
	})
	avant := b.relire(t, b.jeton, b.relance.ID)
	time.Sleep(2 * time.Second)
	if apres := b.relire(t, b.jeton, b.relance.ID); apres.ModifieLe != avant.ModifieLe || apres.Etat != etatPublie {
		t.Errorf("doublon avec effet : %+v devenu %+v", avant, apres)
	}
	for _, a := range append(b.articlesEnAttente(), b.dejaPublie) {
		parRevision := map[int]int{}
		for _, r := range b.resultatsEnregistres(t, a.ID) {
			parRevision[r.revision]++
		}
		for revision, n := range parRevision {
			if n != 1 {
				t.Errorf("%s révision %d : %d résultats appliqués", a.ID, revision, n)
			}
		}
		if parRevision[a.Revision] != 1 {
			t.Errorf("%s : aucune issue appliquée pour la révision %d", a.ID, a.Revision)
		}
	}
}

func (b *bancControleFerme) ac2AucunePerte(t *testing.T) {
	attendre(t, delaiDeTraitement, "files de travail vidées", func() bool {
		return b.canal.compter(t, fileDemandes) == 0 && b.canal.compter(t, fileResultats) == 0
	})
	for _, f := range []string{fileDemandesEcartees, fileResultatsEcartes} {
		if n := b.canal.compter(t, f); n != 0 {
			t.Errorf("%d message(s) dans %s", n, f)
		}
	}
	var enAttente int
	if err := b.base.maitre.QueryRow(t.Context(), `SELECT count(*) FROM outbox WHERE publie_le IS NULL`).Scan(&enAttente); err != nil {
		t.Fatalf("outbox : %v", err)
	}
	if enAttente != 0 {
		t.Errorf("%d ligne(s) d'outbox jamais publiée(s)", enAttente)
	}
}

func (b *bancControleFerme) ac4I3EnFinDeTest(t *testing.T) {
	parMigration, parExecution := releverI3(t, b.base.maitre), releverI3(t, b.execution)
	if parMigration != parExecution {
		t.Fatalf("I3 différent selon le rôle : %+v et %+v", parMigration, parExecution)
	}
	i3 := parExecution
	t.Logf("I3 : %+v", i3)
	if i3.sansResultat != 0 || i3.nonPubliees != 0 || i3.depassees != 0 || i3.emises != i3.appliques {
		t.Fatalf("I3 non tenu : %+v", i3)
	}
	if want := int64(len(b.articlesEnAttente()) + 1); i3.emises != want || i3.relances < 1 {
		t.Fatalf("I3 : %d demande(s) émise(s) et %d relance(s), %d demandes et au moins une relance attendues", i3.emises, i3.relances, want)
	}
	var appliques int64
	if err := b.base.maitre.QueryRow(t.Context(), `SELECT count(*) FROM resultat_controle`).Scan(&appliques); err != nil {
		t.Fatalf("résultats : %v", err)
	}
	if appliques != i3.appliques {
		t.Fatalf("%d ligne(s) dans resultat_controle, I3 en compte %d", appliques, i3.appliques)
	}
}

func (b *bancControleFerme) instantaneOutbox(t *testing.T) map[string]ligneBrute {
	t.Helper()
	lignes, err := b.base.maitre.Query(t.Context(), `
		SELECT id::text, concat_ws('|', echange, cle_routage, type_message, correlation_id::text, encode(corps, 'hex'), cree_le::text), publie_le
		FROM outbox`)
	if err != nil {
		t.Fatalf("lecture de l'outbox : %v", err)
	}
	defer lignes.Close()
	instantane := map[string]ligneBrute{}
	for lignes.Next() {
		var id string
		var l ligneBrute
		if err := lignes.Scan(&id, &l.contenu, &l.publieLe); err != nil {
			t.Fatalf("ligne d'outbox : %v", err)
		}
		instantane[id] = l
	}
	return instantane
}

func memeInstant(a, b *time.Time) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && a.Equal(*b))
}

func exigerOutboxInchangee(t *testing.T, avant, apres map[string]ligneBrute) {
	t.Helper()
	if len(avant) != len(apres) {
		t.Fatalf("outbox : %d ligne(s) avant, %d après", len(avant), len(apres))
	}
	for id, l := range avant {
		if apres[id].contenu != l.contenu || !memeInstant(apres[id].publieLe, l.publieLe) {
			t.Errorf("ligne %s modifiée", id)
		}
	}
}

func (b *bancControleFerme) lancerLeRejeu(t *testing.T, arguments ...string) (*journalPartage, error) {
	t.Helper()
	t.Setenv("BASE_DE_DONNEES_URL", b.base.migration)
	journal := &journalPartage{}
	err := dispatch(contexteBorne(t, time.Minute), logging.New(serviceName, journal, slog.LevelInfo), replayCommand, arguments)
	return journal, err
}

func horodatage(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func (b *bancControleFerme) mc9RejeuRefuse(t *testing.T) {
	maintenant := time.Now()
	cas := []struct {
		nom       string
		arguments []string
	}{
		{"sans argument", nil},
		{"--depuis sans valeur", []string{"--depuis"}},
		{"--depuis absent", []string{"--jusqu-a", horodatage(maintenant.Add(-time.Hour))}},
		{"--depuis hors RFC3339", []string{"--depuis", "hier"}},
		{"--depuis date sans heure", []string{"--depuis", maintenant.Format(time.DateOnly)}},
		{"--jusqu-a hors RFC3339", []string{"--depuis", horodatage(maintenant.Add(-time.Hour)), "--jusqu-a", "demain"}},
		{"début égal à la fin", []string{"--depuis", horodatage(maintenant.Add(-time.Hour)), "--jusqu-a", horodatage(maintenant.Add(-time.Hour))}},
		{"début après la fin", []string{"--depuis", horodatage(maintenant.Add(-time.Hour)), "--jusqu-a", horodatage(maintenant.Add(-2 * time.Hour))}},
		{"début dans le futur", []string{"--depuis", horodatage(maintenant.Add(time.Hour))}},
		{"fin dans le futur", []string{"--depuis", horodatage(maintenant.Add(-time.Hour)), "--jusqu-a", horodatage(maintenant.Add(time.Hour))}},
		{"début au-delà de 7 jours", []string{"--depuis", horodatage(maintenant.Add(-conservationOutbox - time.Hour))}},
		{"début au-delà de 7 jours, fenêtre courte", []string{"--depuis", horodatage(maintenant.Add(-conservationOutbox - time.Hour)), "--jusqu-a", horodatage(maintenant.Add(-conservationOutbox + time.Hour))}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			avant := b.instantaneOutbox(t)
			journal, err := b.lancerLeRejeu(t, c.arguments...)
			if err == nil {
				t.Errorf("rejeu %v accepté", c.arguments)
			}
			t.Logf("refus : %v", err)
			if len(evenementsJournalises(journal, evenementRejeu)) != 0 {
				t.Errorf("rejeu refusé tracé comme effectué : %s", journal.tampon.String())
			}
			exigerOutboxInchangee(t, avant, b.instantaneOutbox(t))
		})
	}
}

func (b *bancControleFerme) premiereDemandeDuPublie(t *testing.T) string {
	t.Helper()
	var id string
	err := b.base.maitre.QueryRow(t.Context(), `
		SELECT id::text FROM outbox WHERE convert_from(corps, 'UTF8')::jsonb -> 'data' ->> 'article_id' = $1
		ORDER BY cree_le LIMIT 1`, b.dejaPublie.ID).Scan(&id)
	if err != nil {
		t.Fatalf("première demande : %v", err)
	}
	return id
}

func (b *bancControleFerme) etatsDesArticles(t *testing.T) map[string]articleRecu {
	t.Helper()
	etats := map[string]articleRecu{}
	for _, a := range append(b.articlesEnAttente(), b.dejaPublie) {
		etats[a.ID] = b.relire(t, b.jeton, a.ID)
	}
	return etats
}

func (b *bancControleFerme) nombreDeResultatsAppliques(t *testing.T) int {
	t.Helper()
	var n int
	if err := b.base.maitre.QueryRow(t.Context(), `SELECT count(*) FROM resultat_controle`).Scan(&n); err != nil {
		t.Fatalf("résultats : %v", err)
	}
	return n
}

type etatAvantRejeu struct {
	outbox    map[string]ligneBrute
	exclue    string
	articles  map[string]articleRecu
	resultats int
	i3        indicateurI3
	ignores   int
}

func (b *bancControleFerme) releverAvantRejeu(t *testing.T) etatAvantRejeu {
	t.Helper()
	return etatAvantRejeu{
		outbox:    b.instantaneOutbox(t),
		exclue:    b.premiereDemandeDuPublie(t),
		articles:  b.etatsDesArticles(t),
		resultats: b.nombreDeResultatsAppliques(t),
		i3:        releverI3(t, b.base.maitre),
		ignores:   len(evenementsJournalises(b.journal, evenementIgnore)),
	}
}

func (e etatAvantRejeu) debutDeFenetreApresLaPremiereDemande(t *testing.T) time.Time {
	t.Helper()
	premiere := *e.outbox[e.exclue].publieLe
	var suivante time.Time
	for id, l := range e.outbox {
		if id != e.exclue && (suivante.IsZero() || l.publieLe.Before(suivante)) {
			suivante = *l.publieLe
		}
	}
	if !premiere.Before(suivante) {
		t.Fatalf("la première demande n'est pas publiée avant les autres")
	}
	return premiere.Add(suivante.Sub(premiere) / 2)
}

func (b *bancControleFerme) toutEstPublie(t *testing.T) bool {
	t.Helper()
	for _, l := range b.instantaneOutbox(t) {
		if l.publieLe == nil {
			return false
		}
	}
	return true
}

func (b *bancControleFerme) exigerSeulePublieLeReecriteDansLaFenetre(t *testing.T, avant etatAvantRejeu, lancement time.Time) {
	t.Helper()
	apres := b.instantaneOutbox(t)
	if len(apres) != len(avant.outbox) {
		t.Fatalf("outbox : %d ligne(s) avant, %d après le rejeu", len(avant.outbox), len(apres))
	}
	for id, l := range avant.outbox {
		switch {
		case apres[id].contenu != l.contenu:
			t.Errorf("ligne %s : colonne autre que publie_le modifiée", id)
		case id == avant.exclue && !apres[id].publieLe.Equal(*l.publieLe):
			t.Errorf("ligne %s hors fenêtre republiée", id)
		case id != avant.exclue && apres[id].publieLe.Before(lancement):
			t.Errorf("ligne %s de la fenêtre non republiée", id)
		}
	}
}

func (b *bancControleFerme) exigerSansDoubleEffet(t *testing.T, avant etatAvantRejeu, rejouees int) {
	t.Helper()
	attendre(t, delaiDeReprise, "doublons de résultats ignorés", func() bool {
		return len(evenementsJournalises(b.journal, evenementIgnore)) >= avant.ignores+rejouees
	})
	time.Sleep(2 * time.Second)
	if n := len(evenementsJournalises(b.journal, evenementIgnore)) - avant.ignores; n != rejouees {
		t.Errorf("%d résultat(s) ignoré(s) après le rejeu, %d attendu(s)", n, rejouees)
	}
	if n := b.nombreDeResultatsAppliques(t); n != avant.resultats {
		t.Errorf("%d résultat(s) appliqué(s) après le rejeu, %d avant", n, avant.resultats)
	}
	for id, a := range b.etatsDesArticles(t) {
		if a.Etat != avant.articles[id].Etat || a.Revision != avant.articles[id].Revision || a.ModifieLe != avant.articles[id].ModifieLe {
			t.Errorf("article %s modifié par le rejeu : %+v devenu %+v", id, avant.articles[id], a)
		}
	}
	if i3 := releverI3(t, b.base.maitre); i3 != avant.i3 {
		t.Errorf("I3 modifié par le rejeu : %+v devenu %+v", avant.i3, i3)
	}
}

func (b *bancControleFerme) mc9RejeuNominal(t *testing.T) {
	avant := b.releverAvantRejeu(t)
	depuis := avant.debutDeFenetreApresLaPremiereDemande(t)
	rejouees := len(avant.outbox) - 1
	lancement := time.Now()
	journal, err := b.lancerLeRejeu(t, "--depuis", depuis.UTC().Format(time.RFC3339Nano))
	if err != nil {
		t.Fatalf("rejeu refusé : %v", err)
	}
	traces := evenementsJournalises(journal, evenementRejeu)
	if len(traces) != 1 {
		t.Fatalf("%d trace(s) %s : %s", len(traces), evenementRejeu, journal.tampon.String())
	}
	t.Logf("trace du rejeu : %v", traces[0])
	if traces[0]["lignes"] != float64(rejouees) || traces[0]["correlation_id"] == nil || traces[0]["depuis"] == nil || traces[0]["jusqu_a"] == nil {
		t.Errorf("trace %v, %d ligne(s) attendue(s)", traces[0], rejouees)
	}
	attendre(t, delaiDeReprise, "republication des lignes rejouées", func() bool { return b.toutEstPublie(t) })
	b.exigerSeulePublieLeReecriteDansLaFenetre(t, avant, lancement)
	b.exigerSansDoubleEffet(t, avant, rejouees)
}

func (b *bancControleFerme) mc9RejeuEnLimiteDeConservation(t *testing.T) {
	avant := b.instantaneOutbox(t)
	depuis := time.Now().Add(-conservationOutbox + time.Minute)
	journal, err := b.lancerLeRejeu(t, "--depuis", horodatage(depuis), "--jusqu-a", horodatage(depuis.Add(time.Minute)))
	if err != nil {
		t.Fatalf("fenêtre de 7 jours refusée : %v", err)
	}
	traces := evenementsJournalises(journal, evenementRejeu)
	if len(traces) != 1 || traces[0]["lignes"] != float64(0) {
		t.Errorf("traces %v, une trace à 0 ligne attendue", traces)
	}
	exigerOutboxInchangee(t, avant, b.instantaneOutbox(t))
}

func varianteDe(t *testing.T, modele []byte, alterer func(map[string]any)) []byte {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(modele, &m); err != nil {
		t.Fatalf("modèle : %v", err)
	}
	m["id"] = uuid.NewString()
	alterer(m)
	corps, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("variante : %v", err)
	}
	return corps
}

func messagesNonConformes(t *testing.T, modele []byte, champObligatoire string) map[string][]byte {
	t.Helper()
	valide := varianteDe(t, modele, func(map[string]any) {})
	return map[string][]byte{
		"JSON illisible":             []byte(`{"id":"tronqué"`),
		"clé dupliquée":              append([]byte(`{"version":1,`), valide[1:]...),
		"champ obligatoire manquant": varianteDe(t, modele, func(m map[string]any) { delete(m["data"].(map[string]any), champObligatoire) }),
		"version inconnue":           varianteDe(t, modele, func(m map[string]any) { m["version"] = 2 }),
		"type inconnu":               varianteDe(t, modele, func(m map[string]any) { m["type"] = "controle.inconnu" }),
		"taille au-delà du contrat":  append(varianteDe(t, modele, func(map[string]any) {}), bytes.Repeat([]byte(" "), tailleHorsContrat)...),
	}
}

type ecartAttendu struct {
	file     string
	ecartes  string
	echange  string
	emetteur string
	journal  func() *journalPartage
	suivant  func(t *testing.T)
}

func (b *bancControleFerme) exigerEcartesSansBloquer(t *testing.T, e ecartAttendu, messages map[string][]byte) {
	t.Helper()
	ecartesAvant := b.canal.compter(t, e.ecartes)
	publies := map[string]string{}
	debut := time.Now()
	for nom, corps := range messages {
		p := publication{echange: e.echange, cle: cleDeRoutageV1, id: uuid.NewString(), emetteur: e.emetteur, corps: corps}
		if err := b.canal.publier(t, e.emetteur, p); err != nil {
			t.Fatalf("publication du message %q : %v", nom, err)
		}
		publies[p.id] = nom
	}
	e.suivant(t)
	b.exigerMessages(t, e.ecartes, ecartesAvant+len(messages))
	for id, nom := range publies {
		attendre(t, delaiDAlerte, "journal de mise à l'écart du message "+nom, func() bool {
			return len(evenementsPortant(e.journal(), evenementEcarte, id)) > 0
		})
		t.Logf("%s : %s journalisé %s après le début des publications", nom, evenementEcarte, time.Since(debut).Round(time.Millisecond))
		traces := evenementsPortant(e.journal(), evenementEcarte, id)
		if len(traces) != 1 {
			t.Errorf("%s : %d mise(s) à l'écart journalisée(s), 1 attendue", nom, len(traces))
		}
		trace := traces[0]
		if trace["file"] != e.file || trace["file_ecartes"] != e.ecartes || trace["emetteur"] != e.emetteur || trace["relivre"] != false || trace["motif"] == nil {
			t.Errorf("%s : trace %v", nom, trace)
		}
	}
	if duree := time.Since(debut); duree >= delaiDAlerte {
		t.Errorf("mises à l'écart journalisées en %s", duree)
	}
	b.exigerUneSeuleLivraison(t, e.ecartes, publies)
}

func (b *bancControleFerme) exigerUneSeuleLivraison(t *testing.T, file string, publies map[string]string) {
	t.Helper()
	vus := 0
	for {
		livraison, presente, err := b.canal.admin.Get(file, true)
		if err != nil {
			t.Fatalf("lecture de %s : %v", file, err)
		}
		if !presente {
			break
		}
		nom, attendu := publies[livraison.MessageId]
		if !attendu {
			continue
		}
		vus++
		morts, _ := livraison.Headers["x-death"].([]any)
		if len(morts) != 1 {
			t.Errorf("%s : en-tête x-death %v", nom, livraison.Headers["x-death"])
			continue
		}
		mort, _ := morts[0].(amqp.Table)
		if mort["reason"] != raisonRejetConsomm || mort["count"] != int64(1) {
			t.Errorf("%s : mis à l'écart pour %v après %v passage(s), rejet par le consommateur dès la première livraison attendu (en-têtes %v)", nom, mort["reason"], mort["count"], livraison.Headers)
		}
	}
	if vus != len(publies) {
		t.Errorf("%d message(s) retrouvé(s) dans %s sur %d", vus, file, len(publies))
	}
}

func (b *bancControleFerme) publierUnSuivant(titre string) func(t *testing.T) {
	return func(t *testing.T) {
		suivant := b.creerEtSoumettre(t, corpsComplet(titre, categorieSure, prixCourant, 0))
		_, duree := b.attendreEtat(t, suivant.ID, etatPublie, delaiDeTraitement)
		t.Logf("article suivant publié en %s", duree.Round(time.Millisecond))
	}
}

func (b *bancControleFerme) ac3DemandesInvalidesEcartees(t *testing.T) {
	modele := b.outboxDe(t, b.temoinRecent.ID)[0].corps
	b.exigerEcartesSansBloquer(t, ecartAttendu{
		file: fileDemandes, ecartes: fileDemandesEcartees, echange: echangeDemandes, emetteur: identiteMarche,
		journal: func() *journalPartage { return b.controle.journal },
		suivant: b.publierUnSuivant("Après des demandes invalides"),
	}, messagesNonConformes(t, modele, "article_id"))
}

func (b *bancControleFerme) ac3ResultatsInvalidesEcartes(t *testing.T) {
	modele := resultatForge(b.temoinRecent, issuePublie, nil)
	b.exigerEcartesSansBloquer(t, ecartAttendu{
		file: fileResultats, ecartes: fileResultatsEcartes, echange: echangeResultats, emetteur: identiteControle,
		journal: func() *journalPartage { return b.journal },
		suivant: b.publierUnSuivant("Après des résultats invalides"),
	}, messagesNonConformes(t, modele, "issue"))
}

func (b *bancControleFerme) mc6MessageTropGrandRefuseParLeBroker(t *testing.T) {
	ecartesAvant := b.canal.compter(t, fileDemandesEcartees)
	corps := append(varianteDe(t, b.outboxDe(t, b.temoinRecent.ID)[0].corps, func(map[string]any) {}), bytes.Repeat([]byte(" "), tailleHorsBroker)...)
	err := b.canal.publier(t, identiteMarche, publication{echange: echangeDemandes, cle: cleDeRoutageV1, id: uuid.NewString(), emetteur: identiteMarche, corps: corps})
	if err == nil {
		t.Fatalf("message de %d octets accepté par le broker", len(corps))
	}
	t.Logf("refus du broker : %v", err)
	b.publierUnSuivant("Après un message trop grand")(t)
	if n := b.canal.compter(t, fileDemandesEcartees); n != ecartesAvant {
		t.Errorf("%d message(s) écarté(s), %d attendu(s)", n, ecartesAvant)
	}
}
