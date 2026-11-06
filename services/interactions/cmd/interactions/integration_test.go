package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	delaiDeNotification = 10 * time.Second
	delaiDeTraitement   = 20 * time.Second
	refusParLeBroker    = 403
	dossierDesExemples  = "messages/publication.decidee/exemples/v1"
)

type decisionEmise struct {
	id      string
	article string
	issue   string
	motif   *string
	corps   []byte
}

type bancInteractions struct {
	*serviceInteractions
	bases      basesPostgreSQL
	canal      *canalDeTest
	historique []reponseHTTP
}

func demarrerBancInteractions(t *testing.T) *bancInteractions {
	t.Helper()
	bases := demarrerPostgreSQL(t)
	canal := demarrerRabbitMQ(t)
	migrer(t, bases)
	return &bancInteractions{serviceInteractions: lancerInteractions(t, bases, canal), bases: bases, canal: canal}
}

func modeleDeDecision(t *testing.T) map[string]any {
	t.Helper()
	corps, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "contracts", dossierDesExemples, "valides", "refuse-par-un-administrateur.json"))
	if err != nil {
		t.Fatalf("exemple de décision : %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(corps, &document); err != nil {
		t.Fatalf("exemple de décision : %v", err)
	}
	return document
}

func decision(t *testing.T, vendeur, issue string, motif *string) decisionEmise {
	t.Helper()
	document := modeleDeDecision(t)
	d := decisionEmise{id: uuid.NewString(), article: uuid.NewString(), issue: issue, motif: motif}
	document["id"] = d.id
	data := document["data"].(map[string]any)
	data["article_id"] = d.article
	data["seller_id"] = vendeur
	data["issue"] = issue
	data["decided_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	delete(data, "reason")
	if motif != nil {
		data["reason"] = *motif
	}
	corps, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("décision : %v", err)
	}
	d.corps = corps
	return d
}

func motif(code string) *string {
	return &code
}

func versLesEvenements(corps []byte) publication {
	return publication{echange: echangeEvenements, cle: cleDecision, id: uuid.NewString(), emetteur: identiteMarche, corps: corps}
}

func (b *bancInteractions) diffuser(t *testing.T, corps []byte) {
	t.Helper()
	if err := b.canal.publier(t, identiteMarche, versLesEvenements(corps)); err != nil {
		t.Fatalf("diffusion de la décision par la Place de marché : %v", err)
	}
}

func (b *bancInteractions) lister(t *testing.T, autorisation, requete string) reponseHTTP {
	t.Helper()
	r := appeler(t, b.adresse, autorisation, requete)
	b.historique = append(b.historique, r)
	return r
}

func (s *serviceInteractions) lister(t *testing.T, autorisation string) pageRecue {
	t.Helper()
	return appeler(t, s.adresse, autorisation, "?taille=50").page(t)
}

func (b *bancInteractions) notificationsDe(t *testing.T, vendeur string) pageRecue {
	t.Helper()
	return b.lister(t, b.jetons.vendeur(vendeur), "?taille=50").page(t)
}

func (b *bancInteractions) attendreNotifications(t *testing.T, vendeur string, nombre int) time.Duration {
	t.Helper()
	return attendre(t, delaiDeTraitement, fmt.Sprintf("%d notification(s) pour %s", nombre, vendeur), func() bool {
		return b.notificationsDe(t, vendeur).Total >= nombre
	})
}

func exigerNotificationDe(t *testing.T, n notificationRecue, d decisionEmise) {
	t.Helper()
	if n.ArticleID != d.article || n.Issue != d.issue || !memeMotif(n.Motif, d.motif) || n.Revision != 1 {
		t.Fatalf("notification %+v, attendue pour l'article %s, issue %s, motif %v", n, d.article, d.issue, valeur(d.motif))
	}
}

func memeMotif(a, b *string) bool {
	return valeur(a) == valeur(b)
}

func valeur(m *string) string {
	if m == nil {
		return "<null>"
	}
	return *m
}

func TestUS028IntegrationInteractions(t *testing.T) {
	b := demarrerBancInteractions(t)
	t.Run("AC2 décision diffusée : une seule notification listée en moins de 10 secondes", b.ac2NotificationListeeEnMoinsDe10Secondes)
	t.Run("AC2 chaque issue et chaque motif donnent une notification", b.ac2ChaqueIssueEtMotif)
	t.Run("AC3 même décision livrée deux fois : une seule notification", b.ac3MemeDecisionDeuxFois)
	t.Run("AC3 même décision reçue par une seconde instance d'Interactions : une seule notification", b.ac3SecondeInstance)
	t.Run("AC4 la liste ne porte que des codes, jamais de contenu saisi", b.ac4ListeSansContenuSaisi)
	t.Run("AC5 messages invalides écartés dès la première livraison sans bloquer les suivants", b.ac5MessagesInvalidesEcartes)
	t.Run("AC5 décision publiée par une autre identité que la Place de marché écartée", b.ac5EmetteurInattenduEcarte)
	t.Run("AC5 nombre d'essais borné par la politique de la file", b.ac5NombreDEssaisBorne)
	t.Run("AC6 chaque vendeur ne lit que ses notifications", b.ac6ChaqueVendeurLesSiennes)
	t.Run("AC6 sans jeton ou jeton invalide : 401", b.ac6NonAuthentifie401)
	t.Run("AC7 interactions ne peut publier ni décision ni résultat de contrôle", b.ac7InteractionsNePeutPasPublier)
	t.Run("AC7 controle ne peut pas publier de décision de publication", b.ac7ControleNePeutPasPublierDeDecision)
	t.Run("AC8 bases cloisonnées et rôle d'exécution limité à SELECT et INSERT", b.ac8BasesCloisonneesEtDroitsMinimaux)
	t.Run("réponses conformes au contrat OpenAPI d'Interactions", b.reponsesConformesAuContrat)
}

func (b *bancInteractions) ac2NotificationListeeEnMoinsDe10Secondes(t *testing.T) {
	vendeur := uuid.NewString()
	d := decision(t, vendeur, "publie", nil)
	debut := time.Now()
	b.diffuser(t, d.corps)
	duree := attendre(t, delaiDeNotification, "notification listée", func() bool {
		return b.notificationsDe(t, vendeur).Total == 1
	})
	t.Logf("notification listée %s après la diffusion", duree.Round(time.Millisecond))
	if time.Since(debut) > delaiDeNotification {
		t.Fatalf("notification listée en %s, plus de 10 s", time.Since(debut))
	}
	time.Sleep(time.Second)
	page := b.notificationsDe(t, vendeur)
	if page.Total != 1 || len(page.Elements) != 1 {
		t.Fatalf("%d notification(s), une seule attendue : %+v", page.Total, page)
	}
	exigerNotificationDe(t, page.Elements[0], d)
}

func (b *bancInteractions) ac2ChaqueIssueEtMotif(t *testing.T) {
	vendeur := uuid.NewString()
	attendues := []decisionEmise{decision(t, vendeur, "publie", nil)}
	for _, issue := range []string{"refuse", "a_verifier"} {
		for _, code := range []string{"annonce_incomplete", "coordonnees_detectees", "compte_non_autorise", "prix_a_confirmer", "vendeur_a_verifier", "categorie_inconnue", "prix_abusif", "article_interdit", "vendeur_non_fiable"} {
			attendues = append(attendues, decision(t, vendeur, issue, motif(code)))
		}
	}
	for _, d := range attendues {
		b.diffuser(t, d.corps)
	}
	b.attendreNotifications(t, vendeur, len(attendues))
	page := b.notificationsDe(t, vendeur)
	if page.Total != len(attendues) {
		t.Fatalf("%d notification(s), %d attendue(s)", page.Total, len(attendues))
	}
	parArticle := map[string]notificationRecue{}
	for _, n := range page.Elements {
		parArticle[n.ArticleID] = n
	}
	for _, d := range attendues {
		exigerNotificationDe(t, parArticle[d.article], d)
	}
}

func (b *bancInteractions) ac3MemeDecisionDeuxFois(t *testing.T) {
	vendeur := uuid.NewString()
	d := decision(t, vendeur, "refuse", motif("article_interdit"))
	doublonsAvant := b.journal.compter("notifications.doublon_ignore")
	b.diffuser(t, d.corps)
	b.diffuser(t, d.corps)
	attendre(t, delaiDeTraitement, "doublon ignoré", func() bool {
		return b.journal.compter("notifications.doublon_ignore") > doublonsAvant
	})
	if page := b.notificationsDe(t, vendeur); page.Total != 1 {
		t.Fatalf("%d notification(s) pour une décision livrée deux fois, 1 attendue", page.Total)
	}
	if n := b.canal.compter(t, fileEcartes); n != 0 {
		t.Fatalf("doublon écarté vers %s (%d message(s)) au lieu d'être acquitté", fileEcartes, n)
	}
}

func (b *bancInteractions) ac3SecondeInstance(t *testing.T) {
	vendeur := uuid.NewString()
	d := decision(t, vendeur, "a_verifier", motif("prix_a_confirmer"))
	b.diffuser(t, d.corps)
	b.attendreNotifications(t, vendeur, 1)
	seconde := lancerInteractions(t, b.bases, b.canal)
	doublons := func() int {
		return b.journal.compter("notifications.doublon_ignore") + seconde.journal.compter("notifications.doublon_ignore")
	}
	avant := doublons()
	for range 4 {
		b.diffuser(t, d.corps)
	}
	attendre(t, delaiDeTraitement, "quatre doublons ignorés", func() bool { return doublons() == avant+4 })
	if page := b.notificationsDe(t, vendeur); page.Total != 1 {
		t.Fatalf("%d notification(s) après quatre relivraisons, 1 attendue", page.Total)
	}
	if page := seconde.lister(t, seconde.jetons.vendeur(vendeur)); page.Total != 1 {
		t.Fatalf("seconde instance : %d notification(s), 1 attendue", page.Total)
	}
}

func (b *bancInteractions) ac4ListeSansContenuSaisi(t *testing.T) {
	vendeur := uuid.NewString()
	d := decision(t, vendeur, "refuse", motif("coordonnees_detectees"))
	b.diffuser(t, d.corps)
	b.attendreNotifications(t, vendeur, 1)
	r := b.lister(t, b.jetons.vendeur(vendeur), "")
	element := exigerSeulementLesChampsDuContrat(t, r.corps, 1)[0]
	for champ, contenu := range element {
		if texte, ok := contenu.(string); ok && strings.Contains(texte, " ") {
			t.Fatalf("champ %s porte du texte libre : %q", champ, texte)
		}
	}
}

func exemplesInvalides(t *testing.T) map[string][]byte {
	t.Helper()
	racine := os.DirFS(filepath.Join("..", "..", "..", "..", "contracts"))
	exemples := map[string][]byte{}
	err := fs.WalkDir(racine, path.Join(dossierDesExemples, "invalides"), func(p string, entree fs.DirEntry, err error) error {
		if err != nil || entree.IsDir() {
			return err
		}
		corps, err := fs.ReadFile(racine, p)
		exemples[strings.TrimPrefix(p, dossierDesExemples+"/invalides/")] = corps
		return err
	})
	if err != nil || len(exemples) == 0 {
		t.Fatalf("exemples invalides : %v", err)
	}
	return exemples
}

func (b *bancInteractions) viderLesEcartes(t *testing.T) {
	t.Helper()
	b.canal.relever(t, fileEcartes)
}

func exigerUneSeuleLivraison(t *testing.T, livraison amqp.Delivery) {
	t.Helper()
	morts, _ := livraison.Headers["x-death"].([]any)
	if len(morts) != 1 {
		t.Fatalf("en-tête x-death %v", livraison.Headers["x-death"])
	}
	mort, _ := morts[0].(amqp.Table)
	if mort["reason"] != "rejected" || mort["count"] != int64(1) || mort["queue"] != fileDecisions {
		t.Fatalf("mise à l'écart %v, attendue après une seule livraison (rejected, count 1)", mort)
	}
}

func (b *bancInteractions) ac5MessagesInvalidesEcartes(t *testing.T) {
	b.viderLesEcartes(t)
	vendeur := uuid.NewString()
	invalides := exemplesInvalides(t)
	trop := append(decision(t, vendeur, "publie", nil).corps, bytes.Repeat([]byte(" "), 70<<10)...)
	invalides["taille_depassee/70-kio"] = trop
	invalides["schema_non_respecte/vendeur-du-banc-avec-titre"] = bytes.Replace(decision(t, vendeur, "publie", nil).corps, []byte(`"issue"`), []byte(`"title":"Figurine Tintin","issue"`), 1)
	debut := time.Now()
	for nom, corps := range invalides {
		if err := b.canal.publier(t, identiteMarche, versLesEvenements(corps)); err != nil {
			t.Fatalf("%s : publication refusée : %v", nom, err)
		}
	}
	suivante := decision(t, vendeur, "publie", nil)
	b.diffuser(t, suivante.corps)
	attendre(t, delaiDeTraitement, "décision suivante notifiée", func() bool { return b.notificationsDe(t, vendeur).Total == 1 })
	t.Logf("décision suivante notifiée %s après la première publication invalide", time.Since(debut).Round(time.Millisecond))
	attendre(t, delaiDeTraitement, fmt.Sprintf("%d messages dans %s", len(invalides), fileEcartes), func() bool {
		return b.canal.compter(t, fileEcartes) == len(invalides)
	})
	for _, livraison := range b.canal.relever(t, fileEcartes) {
		exigerUneSeuleLivraison(t, livraison)
	}
	page := b.notificationsDe(t, vendeur)
	if page.Total != 1 || page.Elements[0].ArticleID != suivante.article {
		t.Fatalf("notifications %+v : seule la décision valide attendue", page)
	}
	if n := b.canal.compter(t, fileDecisions); n != 0 {
		t.Fatalf("%d message(s) bloqué(s) dans %s", n, fileDecisions)
	}
}

func (b *bancInteractions) ac5EmetteurInattenduEcarte(t *testing.T) {
	b.viderLesEcartes(t)
	vendeur := uuid.NewString()
	usurpee := decision(t, vendeur, "publie", nil)
	p := versLesEvenements(usurpee.corps)
	p.emetteur = identiteAdmin
	if err := b.canal.publier(t, identiteAdmin, p); err != nil {
		t.Fatalf("publication par %s : %v", identiteAdmin, err)
	}
	attendre(t, delaiDeTraitement, "décision d'un autre émetteur écartée", func() bool { return b.canal.compter(t, fileEcartes) == 1 })
	time.Sleep(time.Second)
	if page := b.notificationsDe(t, vendeur); page.Total != 0 {
		t.Fatalf("%d notification(s) créée(s) par une décision d'un autre émetteur", page.Total)
	}
	b.viderLesEcartes(t)
}

func (b *bancInteractions) ac5NombreDEssaisBorne(t *testing.T) {
	var files []struct {
		Name       string         `json:"name"`
		Definition map[string]any `json:"effective_policy_definition"`
	}
	if err := json.Unmarshal(b.canal.rabbitmqctl(t, "list_queues", "-p", vhostDuCanal, "name", "effective_policy_definition", "--formatter", "json"), &files); err != nil {
		t.Fatalf("politiques des files : %v", err)
	}
	for _, f := range files {
		if f.Name != fileDecisions {
			continue
		}
		if f.Definition["delivery-limit"] == nil || f.Definition["dead-letter-exchange"] == nil || f.Definition["dead-letter-routing-key"] != fileDecisions {
			t.Fatalf("politique de %s : %v ; nombre de livraisons borné et mise à l'écart attendus", fileDecisions, f.Definition)
		}
		t.Logf("politique effective de %s : %v", fileDecisions, f.Definition)
		return
	}
	t.Fatalf("file %s absente", fileDecisions)
}

func (b *bancInteractions) ac6ChaqueVendeurLesSiennes(t *testing.T) {
	vendeurA, vendeurB := uuid.NewString(), uuid.NewString()
	deA := []decisionEmise{decision(t, vendeurA, "publie", nil), decision(t, vendeurA, "refuse", motif("prix_abusif"))}
	deB := decision(t, vendeurB, "a_verifier", motif("vendeur_a_verifier"))
	for _, d := range append(deA, deB) {
		b.diffuser(t, d.corps)
	}
	b.attendreNotifications(t, vendeurA, 2)
	b.attendreNotifications(t, vendeurB, 1)
	articlesDe := func(p pageRecue) []string {
		var articles []string
		for _, n := range p.Elements {
			articles = append(articles, n.ArticleID)
		}
		slices.Sort(articles)
		return articles
	}
	attendusA := slices.Sorted(slices.Values([]string{deA[0].article, deA[1].article}))
	if got := articlesDe(b.notificationsDe(t, vendeurA)); !slices.Equal(got, attendusA) {
		t.Fatalf("vendeur A lit %v, attendu %v", got, attendusA)
	}
	if got := articlesDe(b.notificationsDe(t, vendeurB)); !slices.Equal(got, []string{deB.article}) {
		t.Fatalf("vendeur B lit %v, attendu %v", got, []string{deB.article})
	}
	for _, requete := range []string{"?vendeur=" + vendeurA, "?vendeurId=" + vendeurA, "?sub=" + vendeurA, "?seller_id=" + vendeurA} {
		r := b.lister(t, b.jetons.vendeur(vendeurB), requete)
		if r.statut == http.StatusOK && slices.Contains(articlesDe(r.page(t)), deA[0].article) {
			t.Fatalf("vendeur B lit une notification de A avec %s", requete)
		}
	}
}

func (b *bancInteractions) ac6NonAuthentifie401(t *testing.T) {
	victime := uuid.NewString()
	b.diffuser(t, decision(t, victime, "refuse", motif("article_interdit")).corps)
	b.attendreNotifications(t, victime, 1)
	for nom, autorisation := range b.jetons.jetonsInvalides(uuid.NewString(), victime) {
		t.Run(nom, func(t *testing.T) {
			r := b.lister(t, autorisation, "")
			if r.statut != http.StatusUnauthorized {
				t.Fatalf("statut %d, 401 attendu : %s", r.statut, r.corps)
			}
			if bytes.Contains(r.corps, []byte("article_interdit")) {
				t.Fatalf("réponse 401 qui divulgue une notification : %s", r.corps)
			}
		})
	}
}

func (b *bancInteractions) exigerPublicationRefusee(t *testing.T, identite string, p publication) {
	t.Helper()
	err := b.canal.publier(t, identite, p)
	if code := codeDeRefus(err); code != refusParLeBroker {
		t.Fatalf("publication par %s sur %s (%s) : %v ; refus 403 attendu", identite, p.echange, p.cle, err)
	}
}

func (b *bancInteractions) ac7InteractionsNePeutPasPublier(t *testing.T) {
	vendeur := uuid.NewString()
	d := decision(t, vendeur, "publie", nil)
	cas := map[string]publication{
		"décision sur pdm.evenements":          {echange: echangeEvenements, cle: cleDecision, corps: d.corps},
		"résultat sur controle.resultats":      {echange: echangeResultats, cle: "v1", corps: []byte(`{}`)},
		"demande sur controle.demandes":        {echange: echangeDemandes, cle: "v1", corps: []byte(`{}`)},
		"décision directe dans sa propre file": {echange: "", cle: fileDecisions, corps: d.corps},
	}
	for nom, p := range cas {
		t.Run(nom, func(t *testing.T) {
			p.id, p.emetteur = uuid.NewString(), identiteInteraction
			b.exigerPublicationRefusee(t, identiteInteraction, p)
		})
	}
	time.Sleep(time.Second)
	if page := b.notificationsDe(t, vendeur); page.Total != 0 {
		t.Fatalf("%d notification(s) créée(s) par une publication refusée", page.Total)
	}
}

func (b *bancInteractions) ac7ControleNePeutPasPublierDeDecision(t *testing.T) {
	vendeur := uuid.NewString()
	d := decision(t, vendeur, "publie", nil)
	for nom, cle := range map[string]string{"clé de la décision": cleDecision, "autre clé": "publication.decidee.v2"} {
		t.Run(nom, func(t *testing.T) {
			b.exigerPublicationRefusee(t, identiteControle, publication{echange: echangeEvenements, cle: cle, id: uuid.NewString(), emetteur: identiteControle, corps: d.corps})
		})
	}
	time.Sleep(time.Second)
	if page := b.notificationsDe(t, vendeur); page.Total != 0 {
		t.Fatalf("%d notification(s) créée(s) par une décision du Contrôle", page.Total)
	}
}

func exigerConnexionRefusee(t *testing.T, adresse, role, base string) {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), adresse)
	if err == nil {
		_ = conn.Close(t.Context())
		t.Fatalf("%s connecté à la base %s", role, base)
	}
	var erreur *pgconn.PgError
	if !errors.As(err, &erreur) || erreur.Code != droitInsuffisant {
		t.Fatalf("%s sur %s : %v ; refus 42501 attendu", role, base, err)
	}
}

func (b *bancInteractions) ac8BasesCloisonneesEtDroitsMinimaux(t *testing.T) {
	interactions, marche := b.bases.interactions, b.bases.marche
	t.Run("interactions_execution refusé sur marche", func(t *testing.T) {
		exigerConnexionRefusee(t, b.bases.adresse(interactions.execution, baseMarche), interactions.execution, baseMarche)
	})
	t.Run("interactions_migration refusé sur marche", func(t *testing.T) {
		exigerConnexionRefusee(t, b.bases.adresse(interactions.migration, baseMarche), interactions.migration, baseMarche)
	})
	t.Run("marche_execution refusé sur interactions", func(t *testing.T) {
		exigerConnexionRefusee(t, b.bases.adresse(marche.execution, baseInteractions), marche.execution, baseInteractions)
	})
	t.Run("marche_migration refusé sur interactions", func(t *testing.T) {
		exigerConnexionRefusee(t, b.bases.adresse(marche.migration, baseInteractions), marche.migration, baseInteractions)
	})
	t.Run("rôle d'exécution : lecture et insertion seulement", func(t *testing.T) {
		vendeur := uuid.NewString()
		b.diffuser(t, decision(t, vendeur, "publie", nil).corps)
		b.attendreNotifications(t, vendeur, 1)
		execution := connecter(t.Context(), t, b.bases.adresse(interactions.execution, baseInteractions))
		var lues int
		if err := execution.QueryRow(t.Context(), "SELECT count(*) FROM notification").Scan(&lues); err != nil || lues == 0 {
			t.Fatalf("lecture des notifications par le rôle d'exécution : %d, %v", lues, err)
		}
		for _, instruction := range []string{
			"UPDATE notification SET issue = 'publie'",
			"DELETE FROM notification",
			"TRUNCATE notification",
			"CREATE TABLE intrus (id integer)",
			"DROP TABLE notification",
			"ALTER TABLE notification ADD COLUMN intrus integer",
		} {
			_, err := execution.Exec(t.Context(), instruction)
			var erreur *pgconn.PgError
			if !errors.As(err, &erreur) || erreur.Code != droitInsuffisant {
				t.Errorf("%q devait être refusé au rôle d'exécution (42501), obtenu %v", instruction, err)
			}
		}
		if page := b.notificationsDe(t, vendeur); page.Total != 1 {
			t.Fatalf("notification modifiée par une instruction refusée : %+v", page)
		}
	})
}
