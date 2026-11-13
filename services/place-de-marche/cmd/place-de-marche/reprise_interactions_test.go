package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	identiteAbonneTest              = "abonne-test"
	fileAbonnementDeTest            = "test.publication-decidee"
	lectureAbonnementDeTest         = "^test[.]publication-decidee$"
	decisionsParVendeur             = 50
	vendeursDeLaRafale              = 40
	delaiDeNotificationApresRelance = 20 * time.Second
	delaiDeRepriseEnRafale          = 90 * time.Second
	delaiDeStabilisation            = 2 * time.Second
	refusDeDroit                    = 403
	evenementDoublonIgnore          = "notifications.doublon_ignore"
	evenementNotification           = "notifications.notification_creee"
)

var argumentsAbonnementDeTest = map[string]any{
	"x-queue-type":  "quorum",
	"x-max-length":  10000,
	"x-overflow":    "drop-head",
	"x-message-ttl": 86400000,
}

type decisionAttendue struct {
	vendeur *vendeurNotifie
	article articleRecu
	issue   string
	motif   *string
}

type bancDeReprise struct {
	*bancNotification
	lancer        lanceurInteractions
	instance      *instanceInteractions
	pendantLArret []decisionAttendue
	gestion       gestionDuCanal
}

type gestionDuCanal struct {
	racine string
	client *http.Client
}

type appelDeGestion struct {
	methode string
	chemin  string
	corps   any
}

func TestUS029RepriseDInteractionsApresArret(t *testing.T) {
	notification, lancer := demarrerBancSansInteractions(t)
	b := &bancDeReprise{bancNotification: notification, lancer: lancer, gestion: gestionDe(t, notification.canal)}
	t.Run("AC1 Interactions arrêté : publications, refus et catalogue non affectés, décisions en attente dans la file", b.ac1PublicationsSansInteractions)
	t.Run("AC2 Interactions relancé : une notification par décision survenue pendant l'arrêt, sans perte ni doublon", b.ac2RepriseSansPerteNiDoublon)
	t.Run("AC2 arrêt brutal en cours de consommation puis relance : aucune perte ni doublon", b.ac2ArretBrutalEnCoursDeConsommation)
	t.Run("AC3 abonnement de test posé par l'exploitation : même événement reçu qu'Interactions à sa relance", b.ac3AbonnementDeTestRecoitLeMemeEvenement)
	t.Run("AC4 l'identité de l'abonnement de test ne peut ni publier, ni déclarer, ni lier", b.ac4AbonneTestNePeutNiPublierNiDeclarer)
	t.Run("AC3 retrait de l'abonnement de test : liaison, file et identité supprimées, diffusion intacte", b.ac3RetraitDeLAbonnementDeTest)
}

func gestionDe(t *testing.T, canal *canalDeTest) gestionDuCanal {
	t.Helper()
	port, err := canal.conteneur.MappedPort(t.Context(), portDeGestion)
	if err != nil {
		t.Fatalf("port de l'API de gestion : %v", err)
	}
	return gestionDuCanal{racine: "http://" + canal.hote + ":" + port.Port() + "/api", client: &http.Client{Timeout: 10 * time.Second}}
}

func cheminDeGestion(segments ...string) string {
	echappes := make([]string, 0, len(segments))
	for _, s := range segments {
		echappes = append(echappes, url.PathEscape(s))
	}
	return "/" + strings.Join(echappes, "/")
}

func (g gestionDuCanal) appeler(t *testing.T, identite, secret string, a appelDeGestion) (int, []byte) {
	t.Helper()
	var corps io.Reader
	if a.corps != nil {
		contenu, err := json.Marshal(a.corps)
		if err != nil {
			t.Fatalf("corps de %s %s : %v", a.methode, a.chemin, err)
		}
		corps = bytes.NewReader(contenu)
	}
	requete, err := http.NewRequestWithContext(t.Context(), a.methode, g.racine+a.chemin, corps)
	if err != nil {
		t.Fatalf("requête de gestion : %v", err)
	}
	requete.SetBasicAuth(identite, secret)
	requete.Header.Set("Content-Type", "application/json")
	reponse, err := g.client.Do(requete)
	if err != nil {
		t.Fatalf("%s %s : %v", a.methode, a.chemin, err)
	}
	defer func() { _ = reponse.Body.Close() }()
	contenu, err := io.ReadAll(reponse.Body)
	if err != nil {
		t.Fatalf("réponse de %s %s : %v", a.methode, a.chemin, err)
	}
	return reponse.StatusCode, contenu
}

func (b *bancDeReprise) enAdministrateur(t *testing.T, a appelDeGestion, attendus ...int) []byte {
	t.Helper()
	statut, contenu := b.gestion.appeler(t, identiteAdmin, b.canal.secrets[identiteAdmin], a)
	for _, attendu := range attendus {
		if statut == attendu {
			return contenu
		}
	}
	t.Fatalf("%s %s en %s : %d %s, attendu %v", a.methode, a.chemin, identiteAdmin, statut, contenu, attendus)
	return nil
}

func (b *bancDeReprise) lireEnAdministrateur(t *testing.T, chemin string, cible any) {
	t.Helper()
	if err := json.Unmarshal(b.enAdministrateur(t, appelDeGestion{methode: http.MethodGet, chemin: chemin}, http.StatusOK), cible); err != nil {
		t.Fatalf("lecture de %s : %v", chemin, err)
	}
}

func (b *bancDeReprise) relancer(t *testing.T) *instanceInteractions {
	t.Helper()
	b.instance = b.lancer(t).attendreDisponibilite(t)
	b.api, b.journalInter = b.instance.api, b.instance.journal
	return b.instance
}

func (b *bancDeReprise) arreter(t *testing.T) {
	t.Helper()
	if b.instance != nil {
		b.instance.arreterBrutalement(t)
	}
}

func (b *bancDeReprise) nombreDeNotifications(t *testing.T) int {
	t.Helper()
	var n int
	if err := b.interactions.maitre.QueryRow(t.Context(), `SELECT count(*) FROM notification`).Scan(&n); err != nil {
		t.Fatalf("lecture des notifications : %v", err)
	}
	return n
}

func (b *bancDeReprise) exigerAucuneDecisionEcartee(t *testing.T) {
	t.Helper()
	if n := b.canal.compter(t, fileNotificationsEcartee); n != 0 {
		t.Fatalf("%d décision(s) écartée(s) vers %s", n, fileNotificationsEcartee)
	}
}

func (b *bancDeReprise) exigerLaFileStable(t *testing.T, file string, attendu int) {
	t.Helper()
	attendre(t, delaiDeTraitement, fmt.Sprintf("%s à %d message(s)", file, attendu), func() bool { return b.canal.compter(t, file) == attendu })
	time.Sleep(delaiDeStabilisation)
	if n := b.canal.compter(t, file); n != attendu {
		t.Fatalf("%d message(s) dans %s, exactement %d attendu(s)", n, file, attendu)
	}
}

func (b *bancDeReprise) decisionDuControlePendantLArret(t *testing.T, v *vendeurNotifie, corps, etat, issue string, motifAttendu *string) articleRecu {
	t.Helper()
	debut := time.Now()
	decide := b.attendreEtat(t, v, b.creerEtSoumettre(t, v, corps).ID, etat)
	b.exigerUneDecisionDiffusee(t, v, decide, issue, motifAttendu)
	t.Logf("article %s %s et décision diffusée %s après la soumission, Interactions arrêté", decide.ID, etat, time.Since(debut).Round(time.Millisecond))
	b.pendantLArret = append(b.pendantLArret, decisionAttendue{vendeur: v, article: decide, issue: issue, motif: motifAttendu})
	return decide
}

func (b *bancDeReprise) decisionDeLAdministrateurPendantLArret(t *testing.T, v *vendeurNotifie, decision, etat, issue string, motifAttendu *string) articleRecu {
	t.Helper()
	aVerifier := b.articleAVerifier(t, v)
	b.exigerUneDecisionDiffusee(t, v, aVerifier, issueAVerifier, motif(motifCategorieInconnu))
	b.pendantLArret = append(b.pendantLArret, decisionAttendue{vendeur: v, article: aVerifier, issue: issueAVerifier, motif: motif(motifCategorieInconnu)})
	tranche := b.deciderAvecSucces(t, b.admin.jeton, aVerifier.ID, aVerifier.Revision, decision, motifAttendu)
	exigerEtat(t, tranche, etat, valeurOuVide(motifAttendu))
	b.exigerUneDecisionDiffusee(t, v, tranche, issue, motifAttendu)
	b.pendantLArret = append(b.pendantLArret, decisionAttendue{vendeur: v, article: tranche, issue: issue, motif: motifAttendu})
	return tranche
}

func (b *bancDeReprise) ac1PublicationsSansInteractions(t *testing.T) {
	if b.instance != nil {
		t.Fatalf("Interactions ne doit pas avoir été lancé")
	}
	a, c := b.nouveauVendeur(), b.nouveauVendeur()
	publies := []articleRecu{
		b.decisionDuControlePendantLArret(t, a, corpsComplet("Figurine Haddock", categorieSure, 4500, 600), etatPublie, issuePublie, nil),
		b.decisionDuControlePendantLArret(t, a, corpsComplet("Figurine Milou", categorieSure, 3200, 500), etatPublie, issuePublie, nil),
		b.decisionDuControlePendantLArret(t, c, corpsComplet("Figurine Dupond", categorieSure, 2800, 400), etatPublie, issuePublie, nil),
		b.decisionDeLAdministrateurPendantLArret(t, c, decisionPublier, etatPublie, issuePublie, nil),
	}
	refuses := []articleRecu{
		b.decisionDuControlePendantLArret(t, a, corpsComplet("Figurine Nestor, appelez le "+telephoneFictif, categorieSure, 3000, 0), etatRefuse, issueRefuse, motif(motifCoordonnees)),
		b.decisionDuControlePendantLArret(t, c, corpsComplet("Figurine Lampion, appelez le "+telephoneFictif, categorieSure, 3000, 0), etatRefuse, issueRefuse, motif(motifCoordonnees)),
		b.decisionDeLAdministrateurPendantLArret(t, a, decisionRefuser, etatRefuse, issueRefuse, motif("article_interdit")),
	}
	catalogue := b.toutLeCatalogue(t, "")
	exigerSansDoublon(t, catalogue)
	exigerPresents(t, catalogue, publies...)
	exigerAbsents(t, catalogue, refuses...)
	for _, p := range publies {
		if f := b.lireFiche(t, p.ID); f.ID != p.ID || f.Titre != p.Titre {
			t.Errorf("fiche publique %+v, attendue pour %s (%q)", f, p.ID, p.Titre)
		}
	}
	for _, r := range refuses {
		b.fiche(t, r.ID).exiger(t, http.StatusNotFound)
	}
	b.exigerLaFileStable(t, fileNotifications, len(b.pendantLArret))
	b.exigerAucuneDecisionEcartee(t)
	if n := b.nombreDeNotifications(t); n != 0 {
		t.Fatalf("%d notification(s) créée(s) alors qu'Interactions est arrêté", n)
	}
	t.Logf("%d décisions appliquées et diffusées, %d articles au catalogue, %d messages en attente dans %s", len(b.pendantLArret), len(catalogue), b.canal.compter(t, fileNotifications), fileNotifications)
}

func (b *bancDeReprise) exigerUneNotificationParDecision(t *testing.T, decisions []decisionAttendue) {
	t.Helper()
	parVendeur := map[*vendeurNotifie][]decisionAttendue{}
	for _, d := range decisions {
		parVendeur[d.vendeur] = append(parVendeur[d.vendeur], d)
	}
	for v, attendues := range parVendeur {
		page := b.notificationsDe(t, v)
		if page.Total != len(attendues) {
			t.Errorf("vendeur %s : %d notification(s), %d décision(s) pendant l'arrêt : %+v", v.sujet, page.Total, len(attendues), page.Elements)
		}
		for _, d := range attendues {
			exigerUneSeuleNotification(t, page.Elements, d)
		}
	}
}

func exigerUneSeuleNotification(t *testing.T, notifications []notificationRecue, d decisionAttendue) {
	t.Helper()
	var trouvees []notificationRecue
	for _, n := range notifications {
		if n.ArticleID == d.article.ID && n.Revision == d.article.Revision && n.Issue == d.issue {
			trouvees = append(trouvees, n)
		}
	}
	if len(trouvees) != 1 {
		t.Errorf("article %s révision %d %s : %d notification(s), exactement une attendue", d.article.ID, d.article.Revision, d.issue, len(trouvees))
		return
	}
	exigerNotificationConforme(t, trouvees[0], d.article, d.issue, d.motif)
}

func (b *bancDeReprise) ac2RepriseSansPerteNiDoublon(t *testing.T) {
	attendues := len(b.pendantLArret)
	if attendues == 0 || b.canal.compter(t, fileNotifications) != attendues {
		t.Fatalf("%d décision(s) en attente dans %s, %d attendue(s) depuis AC1", b.canal.compter(t, fileNotifications), fileNotifications, attendues)
	}
	debut := time.Now()
	b.relancer(t)
	attendre(t, delaiDeNotificationApresRelance, "notifications des décisions survenues pendant l'arrêt", func() bool {
		return b.nombreDeNotifications(t) >= attendues
	})
	t.Logf("%d notification(s) créée(s) %s après la relance (borne %s)", attendues, time.Since(debut).Round(time.Millisecond), delaiDeNotificationApresRelance)
	b.exigerLaFileStable(t, fileNotifications, 0)
	if n := b.nombreDeNotifications(t); n != attendues {
		t.Fatalf("%d notification(s) pour %d décision(s) pendant l'arrêt", n, attendues)
	}
	b.exigerUneNotificationParDecision(t, b.pendantLArret)
	b.exigerAucuneDecisionEcartee(t)
	if crees := len(evenementsJournalises(b.instance.journal, evenementNotification)); crees != attendues {
		t.Errorf("%d %s journalisé(s), %d attendu(s)", crees, evenementNotification, attendues)
	}
}

func decisionForgee(t *testing.T, vendeur string) (string, []byte) {
	t.Helper()
	id, instant := uuid.NewString(), time.Now().UTC().Format(time.RFC3339Nano)
	corps, err := json.Marshal(map[string]any{
		"id":             id,
		"type":           typeDecision,
		"version":        1,
		"emitted_at":     instant,
		"correlation_id": uuid.NewString(),
		"data": map[string]any{
			"article_id": uuid.NewString(),
			"revision":   1,
			"seller_id":  vendeur,
			"issue":      issuePublie,
			"decided_at": instant,
		},
	})
	if err != nil {
		t.Fatalf("décision : %v", err)
	}
	return id, corps
}

func (b *bancDeReprise) diffuserEnRafale(t *testing.T) []*vendeurNotifie {
	t.Helper()
	canal := b.canal.ouvrir(t, identiteMarche)
	if err := canal.Confirm(false); err != nil {
		t.Fatalf("confirmations en %s : %v", identiteMarche, err)
	}
	ctx := contexteBorne(t, time.Minute)
	var vendeurs []*vendeurNotifie
	var confirmations []*amqp.DeferredConfirmation
	for range vendeursDeLaRafale {
		v := b.nouveauVendeur()
		vendeurs = append(vendeurs, v)
		for range decisionsParVendeur {
			id, corps := decisionForgee(t, v.sujet)
			if err := b.validateur.Validate(corps); err != nil {
				t.Fatalf("décision forgée non conforme au contrat v1 : %v", err)
			}
			confirmation, err := canal.PublishWithDeferredConfirmWithContext(ctx, echangeEvenements, cleDecision, true, false, amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				MessageId:    id,
				UserId:       identiteMarche,
				Body:         corps,
			})
			if err != nil {
				t.Fatalf("diffusion en %s : %v", identiteMarche, err)
			}
			confirmations = append(confirmations, confirmation)
		}
	}
	for _, c := range confirmations {
		if acquittee, err := c.WaitContext(ctx); err != nil || !acquittee {
			t.Fatalf("décision non confirmée par le broker : %v", err)
		}
	}
	return vendeurs
}

func (b *bancDeReprise) arreterDesLaPremiereNotification(t *testing.T, instance *instanceInteractions, avant int) int {
	t.Helper()
	limite := time.Now().Add(time.Minute)
	for time.Now().Before(limite) {
		instance.exigerEnVie(t)
		if b.nombreDeNotifications(t) > avant {
			instance.arreterBrutalement(t)
			return b.nombreDeNotifications(t)
		}
	}
	t.Fatalf("aucune notification créée une minute après le lancement d'Interactions")
	return 0
}

func (b *bancDeReprise) ac2ArretBrutalEnCoursDeConsommation(t *testing.T) {
	b.arreter(t)
	avant := b.nombreDeNotifications(t)
	rafale := vendeursDeLaRafale * decisionsParVendeur
	vendeurs := b.diffuserEnRafale(t)
	b.exigerLaFileStable(t, fileNotifications, rafale)
	notifiees := b.arreterDesLaPremiereNotification(t, b.lancer(t), avant) - avant
	if notifiees >= rafale {
		t.Fatalf("les %d décisions étaient déjà notifiées à l'arrêt brutal : l'arrêt n'a pas eu lieu en cours de consommation", rafale)
	}
	var enFile int
	attendre(t, delaiDeTraitement, "décisions non acquittées rendues à la file", func() bool {
		enFile = b.canal.compter(t, fileNotifications)
		return enFile+notifiees >= rafale
	})
	t.Logf("arrêt brutal après %d notification(s) sur %d ; %d décision(s) en file, dont %d déjà notifiée(s) mais non acquittée(s), donc relivrée(s)", notifiees, rafale, enFile, enFile+notifiees-rafale)
	debut := time.Now()
	b.relancer(t)
	attendre(t, delaiDeRepriseEnRafale, "notifications de toute la rafale", func() bool {
		return b.nombreDeNotifications(t) >= avant+rafale
	})
	t.Logf("%d notification(s) en base %s après la relance (borne %s)", rafale, time.Since(debut).Round(time.Millisecond), delaiDeRepriseEnRafale)
	b.exigerLaFileStable(t, fileNotifications, 0)
	if n := b.nombreDeNotifications(t) - avant; n != rafale {
		t.Fatalf("%d notification(s) pour %d décision(s) : perte ou doublon", n, rafale)
	}
	for _, v := range vendeurs {
		page := b.notificationsDe(t, v)
		distincts := map[string]bool{}
		for _, n := range page.Elements {
			distincts[n.ArticleID] = true
		}
		if page.Total != decisionsParVendeur || len(distincts) != decisionsParVendeur {
			t.Errorf("vendeur %s : total %d, %d article(s) distinct(s), %d attendus", v.sujet, page.Total, len(distincts), decisionsParVendeur)
		}
	}
	b.exigerAucuneDecisionEcartee(t)
	t.Logf("relance : %d %s, %d %s", len(evenementsJournalises(b.instance.journal, evenementNotification)), evenementNotification, len(evenementsJournalises(b.instance.journal, evenementDoublonIgnore)), evenementDoublonIgnore)
}

func (b *bancDeReprise) poserLAbonnementDeTest(t *testing.T) {
	t.Helper()
	b.canal.secrets[identiteAbonneTest] = secretAleatoire(t)
	for _, a := range []appelDeGestion{
		{http.MethodPut, cheminDeGestion("users", identiteAbonneTest), map[string]any{"password": b.canal.secrets[identiteAbonneTest], "tags": ""}},
		{http.MethodPut, cheminDeGestion("permissions", vhostDuCanal, identiteAbonneTest), map[string]any{"configure": "", "write": "", "read": lectureAbonnementDeTest}},
		{http.MethodPut, cheminDeGestion("queues", vhostDuCanal, fileAbonnementDeTest), map[string]any{"durable": true, "auto_delete": false, "arguments": argumentsAbonnementDeTest}},
		{http.MethodPost, cheminDeGestion("bindings", vhostDuCanal, "e", echangeEvenements, "q", fileAbonnementDeTest), map[string]any{"routing_key": cleDecision, "arguments": map[string]any{}}},
	} {
		b.enAdministrateur(t, a, http.StatusCreated, http.StatusNoContent)
	}
}

func (b *bancDeReprise) exigerLAbonnementPose(t *testing.T) {
	t.Helper()
	var utilisateur struct {
		Tags []string `json:"tags"`
	}
	b.lireEnAdministrateur(t, cheminDeGestion("users", identiteAbonneTest), &utilisateur)
	var droits struct {
		Configure string `json:"configure"`
		Write     string `json:"write"`
		Read      string `json:"read"`
	}
	b.lireEnAdministrateur(t, cheminDeGestion("permissions", vhostDuCanal, identiteAbonneTest), &droits)
	var file struct {
		Type      string         `json:"type"`
		Durable   bool           `json:"durable"`
		Arguments map[string]any `json:"arguments"`
	}
	b.lireEnAdministrateur(t, cheminDeGestion("queues", vhostDuCanal, fileAbonnementDeTest), &file)
	liaisons := b.clesDesLiaisonsDeLAbonnementDeTest(t)
	if len(utilisateur.Tags) != 0 || droits.Configure != "" || droits.Write != "" || droits.Read != lectureAbonnementDeTest {
		t.Fatalf("identité %s : étiquettes %v, droits %+v", identiteAbonneTest, utilisateur.Tags, droits)
	}
	if file.Type != "quorum" || !file.Durable || file.Arguments["x-max-length"] != float64(10000) || file.Arguments["x-overflow"] != "drop-head" || file.Arguments["x-message-ttl"] != float64(86400000) {
		t.Fatalf("file %s : %+v", fileAbonnementDeTest, file)
	}
	if !slices.Equal(liaisons, []string{cleDecision}) {
		t.Fatalf("liaisons %s → %s : %+v", echangeEvenements, fileAbonnementDeTest, liaisons)
	}
}

func (b *bancDeReprise) ac3AbonnementDeTestRecoitLeMemeEvenement(t *testing.T) {
	b.arreter(t)
	avant := b.nombreDeNotifications(t)
	b.poserLAbonnementDeTest(t)
	b.exigerLAbonnementPose(t)
	v := b.nouveauVendeur()
	publie := b.attendreEtat(t, v, b.creerEtSoumettre(t, v, corpsComplet("Figurine Séraphin", categorieSure, 4100, 500)).ID, etatPublie)
	diffusee := b.exigerUneDecisionDiffusee(t, v, publie, issuePublie, nil)
	emise := b.decisionsEnOutbox(t, publie.ID)[0].corps
	b.exigerLaFileStable(t, fileAbonnementDeTest, 1)
	b.exigerLaFileStable(t, fileNotifications, 1)
	abonne := b.canal.ouvrir(t, identiteAbonneTest)
	recue, presente, err := abonne.Get(fileAbonnementDeTest, false)
	if err != nil || !presente {
		t.Fatalf("lecture de %s en %s : présente %v, %v", fileAbonnementDeTest, identiteAbonneTest, presente, err)
	}
	if err := recue.Ack(false); err != nil {
		t.Fatalf("acquittement en %s : %v", identiteAbonneTest, err)
	}
	var donnees decisionDiffusee
	enveloppe, err := b.validateur.Decode(recue.Body, &donnees)
	if err != nil {
		t.Fatalf("événement reçu par l'abonnement de test non conforme : %v", err)
	}
	if !bytes.Equal(recue.Body, emise) || recue.UserId != identiteMarche || recue.Exchange != echangeEvenements || recue.RoutingKey != cleDecision || donnees != diffusee {
		t.Fatalf("événement reçu %s par %s / %s (%s), émis %s", recue.Body, recue.Exchange, recue.RoutingKey, recue.UserId, emise)
	}
	if _, encore, _ := abonne.Get(fileAbonnementDeTest, true); encore {
		t.Fatalf("plus d'un événement reçu par l'abonnement de test pour une décision")
	}
	if n := b.nombreDeNotifications(t); n != avant {
		t.Fatalf("notification créée alors qu'Interactions est arrêté")
	}
	relance := time.Now()
	b.relancer(t)
	n := b.attendreNotification(t, v, publie, relance)
	exigerNotificationConforme(t, n, publie, issuePublie, nil)
	if !instant(t, n.DecideLe).Equal(instant(t, donnees.DecidedAt)) {
		t.Fatalf("décision notifiée le %q, événement de l'abonnement de test daté %q", n.DecideLe, donnees.DecidedAt)
	}
	var memeEvenement int
	if err := b.interactions.maitre.QueryRow(t.Context(), `SELECT count(*) FROM notification WHERE message_id::text = $1`, enveloppe.ID).Scan(&memeEvenement); err != nil || memeEvenement != 1 {
		t.Fatalf("notification d'Interactions pour l'événement %s : %d (%v), exactement une attendue", enveloppe.ID, memeEvenement, err)
	}
	t.Logf("abonnement de test et Interactions ont reçu l'événement %s (message_id %s), même corps que l'outbox", enveloppe.ID, recue.MessageId)
}

func (b *bancDeReprise) exigerRefusDeDroit(t *testing.T, quoi string, operation func(*amqp.Channel) error) {
	t.Helper()
	canal := b.canal.ouvrir(t, identiteAbonneTest)
	fermeture := canal.NotifyClose(make(chan *amqp.Error, 1))
	err := operation(canal)
	if err == nil {
		select {
		case raison := <-fermeture:
			err = raison
		case <-time.After(5 * time.Second):
		}
	}
	if code := codeDeRefus(err); code != refusDeDroit {
		t.Errorf("%s en %s : %v ; refus 403 attendu", quoi, identiteAbonneTest, err)
		return
	}
	t.Logf("%s refusé : %v", quoi, err)
}

func (b *bancDeReprise) ac4AbonneTestNePeutNiPublierNiDeclarer(t *testing.T) {
	enFileInteractions, enFileTest, notifiees := b.canal.compter(t, fileNotifications), b.canal.compter(t, fileAbonnementDeTest), b.nombreDeNotifications(t)
	v := b.nouveauVendeur()
	for _, cible := range []struct{ echange, cle string }{
		{echangeEvenements, cleDecision},
		{echangeEvenements, "publication.decidee.v2"},
		{"controle.demandes", "v1"},
		{"controle.resultats", "v1"},
		{"", fileAbonnementDeTest},
		{"", fileNotifications},
	} {
		id, corps := decisionForgee(t, v.sujet)
		err := b.canal.publier(t, identiteAbonneTest, publication{echange: cible.echange, cle: cible.cle, id: id, emetteur: identiteAbonneTest, corps: corps})
		if code := codeDeRefus(err); code != refusDeDroit {
			t.Errorf("publication sur %q / %q en %s : %v ; refus 403 attendu", cible.echange, cible.cle, identiteAbonneTest, err)
			continue
		}
		t.Logf("publication sur %q / %q refusée : %v", cible.echange, cible.cle, err)
	}
	b.exigerRefusDeDroit(t, "déclaration d'une file", func(c *amqp.Channel) error {
		_, err := c.QueueDeclare("test.intrus", true, false, false, false, amqp.Table{"x-queue-type": "quorum"})
		return err
	})
	b.exigerRefusDeDroit(t, "déclaration d'un échange", func(c *amqp.Channel) error {
		return c.ExchangeDeclare("test.intrus", "topic", true, false, false, false, nil)
	})
	b.exigerRefusDeDroit(t, "liaison de sa file à pdm.evenements", func(c *amqp.Channel) error {
		return c.QueueBind(fileAbonnementDeTest, "#", echangeEvenements, false, nil)
	})
	b.exigerRefusDeDroit(t, "liaison de sa file à controle.resultats", func(c *amqp.Channel) error {
		return c.QueueBind(fileAbonnementDeTest, "v1", "controle.resultats", false, nil)
	})
	b.exigerRefusDeDroit(t, "suppression de sa file", func(c *amqp.Channel) error {
		_, err := c.QueueDelete(fileAbonnementDeTest, false, false, false)
		return err
	})
	for _, a := range []appelDeGestion{
		{http.MethodGet, "/whoami", nil},
		{http.MethodPut, cheminDeGestion("queues", vhostDuCanal, "test.intrus"), map[string]any{"durable": true}},
		{http.MethodPost, cheminDeGestion("bindings", vhostDuCanal, "e", echangeEvenements, "q", fileAbonnementDeTest), map[string]any{"routing_key": "#"}},
		{http.MethodPost, cheminDeGestion("exchanges", vhostDuCanal, echangeEvenements, "publish"), map[string]any{"routing_key": cleDecision, "payload": "{}", "payload_encoding": "string", "properties": map[string]any{}}},
	} {
		if statut, contenu := b.gestion.appeler(t, identiteAbonneTest, b.canal.secrets[identiteAbonneTest], a); statut != http.StatusUnauthorized {
			t.Errorf("API de gestion %s %s en %s : %d %s ; 401 attendu", a.methode, a.chemin, identiteAbonneTest, statut, contenu)
		}
	}
	time.Sleep(delaiDeStabilisation)
	liaisons := b.clesDesLiaisonsDeLAbonnementDeTest(t)
	if !slices.Equal(liaisons, []string{cleDecision}) {
		t.Errorf("liaisons de %s modifiées : %+v", fileAbonnementDeTest, liaisons)
	}
	if b.canal.compter(t, fileNotifications) != enFileInteractions || b.canal.compter(t, fileAbonnementDeTest) != enFileTest || b.nombreDeNotifications(t) != notifiees {
		t.Fatalf("un message de %s est parvenu : %s %d→%d, %s %d→%d, notifications %d→%d", identiteAbonneTest,
			fileNotifications, enFileInteractions, b.canal.compter(t, fileNotifications), fileAbonnementDeTest, enFileTest, b.canal.compter(t, fileAbonnementDeTest), notifiees, b.nombreDeNotifications(t))
	}
	var echanges []struct {
		Name string `json:"name"`
	}
	b.lireEnAdministrateur(t, cheminDeGestion("exchanges", vhostDuCanal), &echanges)
	for _, e := range echanges {
		if e.Name == "test.intrus" {
			t.Errorf("échange %s créé par %s", e.Name, identiteAbonneTest)
		}
	}
	b.enAdministrateur(t, appelDeGestion{methode: http.MethodGet, chemin: cheminDeGestion("queues", vhostDuCanal, "test.intrus")}, http.StatusNotFound)
}

type ressourceDeGestion struct {
	Name        string `json:"name"`
	Destination string `json:"destination"`
	RoutingKey  string `json:"routing_key"`
}

func (b *bancDeReprise) clesDesLiaisonsDeLAbonnementDeTest(t *testing.T) []string {
	t.Helper()
	var liaisons []ressourceDeGestion
	b.lireEnAdministrateur(t, cheminDeGestion("bindings", vhostDuCanal, "e", echangeEvenements, "q", fileAbonnementDeTest), &liaisons)
	cles := make([]string, 0, len(liaisons))
	for _, l := range liaisons {
		cles = append(cles, l.RoutingKey)
	}
	return cles
}

func (b *bancDeReprise) exigerAbsentDeLaListe(t *testing.T, chemin, quoi, nom string, cle func(ressourceDeGestion) string) {
	t.Helper()
	var ressources []ressourceDeGestion
	b.lireEnAdministrateur(t, chemin, &ressources)
	for _, r := range ressources {
		if cle(r) == nom {
			t.Errorf("%s %s restante après le retrait", quoi, nom)
		}
	}
}

func (b *bancDeReprise) ac3RetraitDeLAbonnementDeTest(t *testing.T) {
	for _, a := range []appelDeGestion{
		{http.MethodDelete, cheminDeGestion("bindings", vhostDuCanal, "e", echangeEvenements, "q", fileAbonnementDeTest, cleDecision), nil},
		{http.MethodDelete, cheminDeGestion("queues", vhostDuCanal, fileAbonnementDeTest), nil},
		{http.MethodDelete, cheminDeGestion("users", identiteAbonneTest), nil},
	} {
		b.enAdministrateur(t, a, http.StatusNoContent)
	}
	for _, chemin := range []string{
		cheminDeGestion("users", identiteAbonneTest),
		cheminDeGestion("permissions", vhostDuCanal, identiteAbonneTest),
		cheminDeGestion("queues", vhostDuCanal, fileAbonnementDeTest),
	} {
		b.enAdministrateur(t, appelDeGestion{methode: http.MethodGet, chemin: chemin}, http.StatusNotFound)
	}
	b.exigerAbsentDeLaListe(t, cheminDeGestion("bindings", vhostDuCanal), "liaison vers", fileAbonnementDeTest, func(r ressourceDeGestion) string { return r.Destination })
	b.exigerAbsentDeLaListe(t, cheminDeGestion("queues", vhostDuCanal), "file", fileAbonnementDeTest, func(r ressourceDeGestion) string { return r.Name })
	b.exigerAbsentDeLaListe(t, "/users", "identité", identiteAbonneTest, func(r ressourceDeGestion) string { return r.Name })
	if connexion, err := amqp.Dial(b.canal.adresse(identiteAbonneTest)); err == nil {
		_ = connexion.Close()
		t.Fatalf("connexion acceptée pour %s après son retrait", identiteAbonneTest)
	} else {
		t.Logf("connexion de %s refusée après retrait : %v", identiteAbonneTest, err)
	}
	b.relancer(t)
	b.decisionDuControle(t, corpsComplet("Figurine Abdallah", categorieSure, 3900, 500), etatPublie, issuePublie, nil)
}
