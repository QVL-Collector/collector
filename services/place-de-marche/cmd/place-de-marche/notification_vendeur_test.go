package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	amqp "github.com/rabbitmq/amqp091-go"

	"collector/libs/platform/message"
	"collector/services/place-de-marche/internal/migrations"
)

const (
	typeDecision             = "publication.decidee"
	echangeEvenements        = "pdm.evenements"
	cleDecision              = "publication.decidee.v1"
	fileNotifications        = "interactions.publication-decidee"
	fileNotificationsEcartee = "interactions.publication-decidee.ecartes"
	fileEspionDeDecisions    = "test.espion-decisions"
	identiteInteractions     = "interactions"
	baseInteractions         = "interactions"
	roleInteractionsMigr     = "interactions_migration"
	roleInteractionsExec     = "interactions_execution"
	cheminNotifications      = "/interactions/v1/notifications"
	delaiDeNotification      = 10 * time.Second
	titreSignature           = "Figurine Tintin ZQX-7731"
)

type decisionDiffusee struct {
	ArticleID string  `json:"article_id"`
	Revision  int     `json:"revision"`
	SellerID  string  `json:"seller_id"`
	Issue     string  `json:"issue"`
	Reason    *string `json:"reason"`
	DecidedAt string  `json:"decided_at"`
}

type notificationRecue struct {
	ID        string  `json:"id"`
	ArticleID string  `json:"articleId"`
	Revision  int     `json:"revision"`
	Issue     string  `json:"issue"`
	Motif     *string `json:"motif"`
	DecideLe  string  `json:"decideLe"`
}

type pageNotifications struct {
	Elements []notificationRecue `json:"elements"`
	Total    int                 `json:"total"`
}

type vendeurNotifie struct {
	sujet         string
	jeton         string
	jetonEspace   string
	articlesSuivi []string
}

type bancNotification struct {
	*environnement
	canal        *canalDeTest
	base         basePostgreSQL
	interactions basePostgreSQL
	journal      *journalPartage
	validateur   *message.Validator
	api          string
	journalInter *journalPartage
	admin        administrateur
	espionnees   []amqp.Delivery
}

func amorcerInteractions(ctx context.Context, t *testing.T, adresseMaitre string) basePostgreSQL {
	t.Helper()
	maitre := connecter(ctx, t, adresseMaitre)
	secretMigration, secretExecution := secretAleatoire(t), secretAleatoire(t)
	executerFormate(ctx, t, maitre, "CREATE ROLE %I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD %L", roleInteractionsMigr, secretMigration)
	executerFormate(ctx, t, maitre, "CREATE ROLE %I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD %L", roleInteractionsExec, secretExecution)
	executerFormate(ctx, t, maitre, "CREATE DATABASE %I OWNER %I", baseInteractions, roleInteractionsMigr)
	executerFormate(ctx, t, maitre, "REVOKE CONNECT ON DATABASE %I FROM PUBLIC", baseInteractions)
	executerFormate(ctx, t, maitre, "GRANT CONNECT ON DATABASE %I TO %I, %I", baseInteractions, roleInteractionsMigr, roleInteractionsExec)
	base := basePostgreSQL{
		maitre:    connecter(ctx, t, versUneBase(t, adresseMaitre, baseInteractions, enTantQueMaitre)),
		migration: versUneBase(t, adresseMaitre, baseInteractions, enTantQue(roleInteractionsMigr, secretMigration)),
		execution: versUneBase(t, adresseMaitre, baseInteractions, enTantQue(roleInteractionsExec, secretExecution)),
	}
	if _, err := connecter(ctx, t, base.migration).Exec(ctx, "REVOKE CREATE ON SCHEMA public FROM PUBLIC"); err != nil {
		t.Fatalf("révocation de CREATE sur public d'interactions : %v", err)
	}
	return base
}

func versUneBase(t *testing.T, adresseMaitre, nom string, utilisateur func(*url.Userinfo) *url.Userinfo) string {
	t.Helper()
	adresse, err := url.Parse(adresseMaitre)
	if err != nil {
		t.Fatalf("adresse du conteneur : %v", err)
	}
	adresse.User = utilisateur(adresse.User)
	adresse.Path = "/" + nom
	return adresse.String()
}

func avecLaBase(adresseRole, nom string) string {
	adresse, err := url.Parse(adresseRole)
	if err != nil {
		return ""
	}
	adresse.Path = "/" + nom
	return adresse.String()
}

func environnementDuProcessus(variables map[string]string) []string {
	env := os.Environ()
	for nom, valeur := range variables {
		env = append(env, nom+"="+valeur)
	}
	return env
}

func migrerInteractions(t *testing.T, binaire string, base basePostgreSQL) {
	t.Helper()
	for range 2 {
		migration := exec.Command(binaire, "migrate")
		migration.Env = environnementDuProcessus(map[string]string{"BASE_DE_DONNEES_URL": base.migration, "BASE_DE_DONNEES_TLS_EXIGE": "false"})
		if sortie, err := migration.CombinedOutput(); err != nil {
			t.Fatalf("migrations d'Interactions : %v\n%s", err, sortie)
		}
	}
}

type instanceInteractions struct {
	api       string
	gestion   string
	journal   *journalPartage
	processus *exec.Cmd
	termine   chan error
	arretee   bool
}

func lancerProcessusInteractions(t *testing.T, binaire string, canal *canalDeTest, base basePostgreSQL, emetteur emetteurDeJetons) *instanceInteractions {
	t.Helper()
	api, gestion := adresseLibre(t), adresseLibre(t)
	variables := canal.environnementDeService(identiteInteractions)
	variables["BASE_DE_DONNEES_URL"] = base.execution
	variables["BASE_DE_DONNEES_TLS_EXIGE"] = "false"
	variables["OIDC_EMETTEUR"] = emetteurAttendu
	variables["OIDC_JEU_DE_CLES_URL"] = emetteur.jwks.URL
	variables["OIDC_AUDIENCE"] = identiteInteractions
	variables["OIDC_CLIENT_AUTORISE"] = clientAutorise
	variables["CANAL_EMETTEUR_ATTENDU"] = identiteMarche
	variables["ADRESSE_API"] = api
	variables["ADRESSE_GESTION"] = gestion
	instance := &instanceInteractions{api: "http://" + api, gestion: "http://" + gestion, journal: &journalPartage{}, termine: make(chan error, 1)}
	instance.processus = exec.Command(binaire, "serve")
	instance.processus.Env = environnementDuProcessus(variables)
	instance.processus.Stdout, instance.processus.Stderr = instance.journal, instance.journal
	if err := instance.processus.Start(); err != nil {
		t.Fatalf("lancement d'Interactions : %v", err)
	}
	go func() { instance.termine <- instance.processus.Wait() }()
	t.Cleanup(func() { instance.arreterBrutalement(t) })
	return instance
}

func (i *instanceInteractions) exigerEnVie(t *testing.T) {
	t.Helper()
	select {
	case err := <-i.termine:
		i.arretee = true
		t.Fatalf("Interactions s'est arrêté : %v\n%s", err, i.journal.tampon.String())
	default:
	}
}

func (i *instanceInteractions) attendreDisponibilite(t *testing.T) *instanceInteractions {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	attendre(t, time.Minute, "démarrage d'Interactions", func() bool {
		i.exigerEnVie(t)
		return repond(client, i.gestion+"/disponibilite")
	})
	return i
}

func (i *instanceInteractions) arreterBrutalement(t *testing.T) {
	t.Helper()
	if i.arretee {
		return
	}
	i.arretee = true
	_ = i.processus.Process.Kill()
	select {
	case erreur := <-i.termine:
		t.Logf("arrêt d'Interactions : %v", erreur)
	case <-time.After(10 * time.Second):
		t.Errorf("Interactions ne s'est pas arrêté en 10 s")
	}
}

type lanceurInteractions func(t *testing.T) *instanceInteractions

func demarrerBancNotification(t *testing.T) *bancNotification {
	t.Helper()
	b, lancer := demarrerBancSansInteractions(t)
	instance := lancer(t).attendreDisponibilite(t)
	b.api, b.journalInter = instance.api, instance.journal
	return b
}

func demarrerBancSansInteractions(t *testing.T) (*bancNotification, lanceurInteractions) {
	t.Helper()
	ctx := contexteBorne(t, 4*time.Minute)
	adresseMaitre := demarrerPostgreSQL(t)
	base := amorcer(ctx, t, adresseMaitre)
	if err := migrations.Apply(ctx, journalMuet(), base.migration); err != nil {
		t.Fatalf("migrations de la Place de marché : %v", err)
	}
	interactions := amorcerInteractions(ctx, t, adresseMaitre)
	canal := demarrerRabbitMQ(t)
	canal.poserUnEspion(t)
	poserUnEspionDeDecisions(t, canal)
	binaireInteractions := construireBinaire(t, filepath.Join("..", "..", "..", "interactions", "cmd", "interactions"), "interactions")
	migrerInteractions(t, binaireInteractions, interactions)
	emetteur := nouvelEmetteurDeJetons(t)
	e, journal := lancerPlaceDeMarche(t, canal, base.execution, emetteur)
	if _, err := base.maitre.Exec(ctx, `INSERT INTO categorie (id, libelle) VALUES ($1, 'Timbres')`, categorieNonReconnue); err != nil {
		t.Fatalf("catégorie inconnue du Contrôle : %v", err)
	}
	lancerControle(t, construireControle(t), canal)
	b := &bancNotification{
		environnement: e,
		canal:         canal,
		base:          base,
		interactions:  interactions,
		journal:       journal,
		validateur:    validateurDeContrats(t),
		admin:         e.administrateur(),
	}
	return b, func(t *testing.T) *instanceInteractions {
		t.Helper()
		return lancerProcessusInteractions(t, binaireInteractions, canal, interactions, emetteur)
	}
}

func poserUnEspionDeDecisions(t *testing.T, canal *canalDeTest) {
	t.Helper()
	if _, err := canal.admin.QueueDeclare(fileEspionDeDecisions, true, false, false, false, amqp.Table{"x-queue-type": "classic"}); err != nil {
		t.Fatalf("abonnement de test : %v", err)
	}
	if err := canal.admin.QueueBind(fileEspionDeDecisions, cleDecision, echangeEvenements, false, nil); err != nil {
		t.Fatalf("liaison de l'abonnement de test : %v", err)
	}
}

func (b *bancNotification) nouveauVendeur() *vendeurNotifie {
	sujet := uuid.NewString()
	return &vendeurNotifie{
		sujet:       sujet,
		jeton:       bearer(b.jetons.vendeur(sujet)),
		jetonEspace: bearer(b.jetons.emettre(sujet, []string{roleVendeur}, avecRevendication("aud", []string{identiteInteractions, "account"}))),
	}
}

func (b *bancNotification) notificationsDe(t *testing.T, v *vendeurNotifie) pageNotifications {
	t.Helper()
	requete, err := http.NewRequestWithContext(t.Context(), http.MethodGet, b.api+cheminNotifications+"?taille=50", nil)
	if err != nil {
		t.Fatalf("requête : %v", err)
	}
	requete.Header.Set("Authorization", v.jetonEspace)
	r := b.executer(t, requete).exiger(t, http.StatusOK)
	var page pageNotifications
	r.lire(t, &page)
	return page
}

func (b *bancNotification) notificationsDeLArticle(t *testing.T, v *vendeurNotifie, id string) []notificationRecue {
	t.Helper()
	var trouvees []notificationRecue
	for _, n := range b.notificationsDe(t, v).Elements {
		if n.ArticleID == id {
			trouvees = append(trouvees, n)
		}
	}
	return trouvees
}

func (b *bancNotification) attendreEtat(t *testing.T, v *vendeurNotifie, id, etat string) articleRecu {
	t.Helper()
	var dernier articleRecu
	attendre(t, delaiDeTraitement, fmt.Sprintf("article %s %s", id, etat), func() bool {
		dernier = b.relire(t, v.jeton, id)
		return dernier.Etat == etat
	})
	return dernier
}

func (b *bancNotification) attendreNotification(t *testing.T, v *vendeurNotifie, article articleRecu, depuis time.Time) notificationRecue {
	t.Helper()
	id := article.ID
	var trouvee notificationRecue
	attendre(t, delaiDeTraitement, "notification de "+id, func() bool {
		for _, n := range b.notificationsDeLArticle(t, v, id) {
			if n.Revision == article.Revision && n.Issue == article.Etat {
				trouvee = n
				return true
			}
		}
		return false
	})
	if duree := time.Since(depuis); duree > delaiDeNotification {
		t.Fatalf("notification de %s listée %s après l'action, plus de 10 s", id, duree)
	} else {
		t.Logf("notification de %s (%s) listée %s après l'action", id, trouvee.Issue, duree.Round(time.Millisecond))
	}
	return trouvee
}

func (b *bancNotification) decisionsEnOutbox(t *testing.T, id string) []ligneOutbox {
	t.Helper()
	return lignesDOutbox(t, b.base, id, typeDecision)
}

func (b *bancNotification) exigerUneDecisionDiffusee(t *testing.T, v *vendeurNotifie, article articleRecu, issue string, motif *string) decisionDiffusee {
	t.Helper()
	var correspondantes []decisionDiffusee
	var ligne ligneOutbox
	attendre(t, delaiDeTraitement, "décision de "+article.ID+" publiée", func() bool {
		correspondantes = nil
		for _, l := range b.decisionsEnOutbox(t, article.ID) {
			var d decisionDiffusee
			if _, err := b.validateur.Decode(l.corps, &d); err != nil {
				t.Fatalf("décision de l'outbox non conforme au contrat v1 : %v", err)
			}
			if d.Revision == article.Revision && d.Issue == issue {
				correspondantes = append(correspondantes, d)
				ligne = l
			}
		}
		return len(correspondantes) == 1 && ligne.publiee
	})
	d := correspondantes[0]
	if ligne.echange != echangeEvenements || ligne.cle != cleDecision || ligne.typ != typeDecision {
		t.Fatalf("décision écrite vers %s / %s (%s), attendue vers %s / %s", ligne.echange, ligne.cle, ligne.typ, echangeEvenements, cleDecision)
	}
	if d.ArticleID != article.ID || d.SellerID != v.sujet || d.Issue != issue || valeurDuMotif(d.Reason) != valeurDuMotif(motif) {
		t.Fatalf("décision diffusée %+v, attendue pour %s, vendeur %s, issue %s, motif %s", d, article.ID, v.sujet, issue, valeurDuMotif(motif))
	}
	if _, err := time.Parse(time.RFC3339Nano, d.DecidedAt); err != nil {
		t.Fatalf("date de décision %q : %v", d.DecidedAt, err)
	}
	return d
}

func exigerNotificationConforme(t *testing.T, n notificationRecue, article articleRecu, issue string, motif *string) {
	t.Helper()
	if n.ArticleID != article.ID || n.Revision != article.Revision || n.Issue != issue || valeurDuMotif(n.Motif) != valeurDuMotif(motif) {
		t.Fatalf("notification %+v, attendue pour %s révision %d, issue %s, motif %s", n, article.ID, article.Revision, issue, valeurDuMotif(motif))
	}
}

func (b *bancNotification) creerEtSoumettre(t *testing.T, v *vendeurNotifie, corps string) articleRecu {
	t.Helper()
	soumis := b.soumettreAvecSucces(t, v.jeton, b.creer(t, v.jeton, corps))
	v.articlesSuivi = append(v.articlesSuivi, soumis.ID)
	return soumis
}

func (b *bancNotification) decisionDuControle(t *testing.T, corps, etat, issue string, motif *string) {
	t.Helper()
	v := b.nouveauVendeur()
	debut := time.Now()
	soumis := b.creerEtSoumettre(t, v, corps)
	decide := b.attendreEtat(t, v, soumis.ID, etat)
	b.exigerUneDecisionDiffusee(t, v, decide, issue, motif)
	exigerNotificationConforme(t, b.attendreNotification(t, v, decide, debut), decide, issue, motif)
	time.Sleep(time.Second)
	if n := b.notificationsDe(t, v); n.Total != 1 {
		t.Fatalf("%d notification(s) pour une décision, une seule attendue : %+v", n.Total, n)
	}
}

func (b *bancNotification) articleAVerifier(t *testing.T, v *vendeurNotifie) articleRecu {
	t.Helper()
	soumis := b.creerEtSoumettre(t, v, corpsComplet(titreSignature, categorieNonReconnue, 4500, 600))
	return b.attendreEtat(t, v, soumis.ID, etatAVerifier)
}

func TestUS028BoutEnBoutPlaceDeMarcheInteractions(t *testing.T) {
	b := demarrerBancNotification(t)
	t.Run("AC1 AC2 contrôle publie : décision dans l'outbox, diffusée, une notification en moins de 10 s", b.ac1ac2ControlePublie)
	t.Run("AC1 AC2 contrôle refuse : notification refusée avec la catégorie de motif", b.ac1ac2ControleRefuse)
	t.Run("AC1 AC2 contrôle à vérifier : notification à vérifier avec la catégorie de motif", b.ac1ac2ControleAVerifier)
	t.Run("AC1 AC2 décision de l'administrateur : publier et refuser avec chaque motif", b.ac1ac2DecisionsAdministrateur)
	t.Run("AC1 décision admin sans écriture possible de l'événement : rien n'est appliqué", b.ac1DecisionAdminAtomique)
	t.Run("AC1 résultat du contrôle sans écriture possible de l'événement : rien n'est appliqué", b.ac1ResultatControleAtomique)
	t.Run("AC1 AC3 résultat en double ou périmé : aucun événement, aucune notification en plus", b.ac1ac3ResultatDoubleOuPerime)
	t.Run("AC4 ni la décision diffusée ni la notification ne portent de contenu saisi", b.ac4SansContenuSaisi)
	t.Run("AC6 un vendeur ne lit que ses notifications sur le banc complet", b.ac6CloisonnementDesVendeurs)
	t.Run("AC8 bases marche et interactions cloisonnées sur le même PostgreSQL", b.ac8BasesCloisonnees)
	t.Run("AC9 contrat de la paire : chaque décision émise respecte le schéma v1 et est acceptée par Interactions", b.ac9ContratDeLaPaire)
}

func (b *bancNotification) ac1ac2ControlePublie(t *testing.T) {
	b.decisionDuControle(t, corpsComplet("Figurine Haddock", categorieSure, 4500, 600), etatPublie, issuePublie, nil)
}

func (b *bancNotification) ac1ac2ControleRefuse(t *testing.T) {
	b.decisionDuControle(t, corpsComplet("Figurine Milou, appelez le "+telephoneFictif, categorieSure, 3000, 0), etatRefuse, issueRefuse, motif(motifCoordonnees))
}

func (b *bancNotification) ac1ac2ControleAVerifier(t *testing.T) {
	b.decisionDuControle(t, corpsComplet("Timbre rare", categorieNonReconnue, 4500, 600), etatAVerifier, issueAVerifier, motif(motifCategorieInconnu))
}

func (b *bancNotification) ac1ac2DecisionsAdministrateur(t *testing.T) {
	v := b.nouveauVendeur()
	cas := []struct {
		decision string
		etat     string
		issue    string
		motif    *string
	}{{decisionPublier, etatPublie, issuePublie, nil}}
	for _, code := range motifsDeRefus {
		cas = append(cas, struct {
			decision string
			etat     string
			issue    string
			motif    *string
		}{decisionRefuser, etatRefuse, issueRefuse, motif(code)})
	}
	for _, c := range cas {
		t.Run(c.decision+" "+valeurDuMotif(c.motif), func(t *testing.T) {
			aVerifier := b.articleAVerifier(t, v)
			debut := time.Now()
			tranche := b.deciderAvecSucces(t, b.admin.jeton, aVerifier.ID, aVerifier.Revision, c.decision, c.motif)
			exigerEtat(t, tranche, c.etat, valeurOuVide(c.motif))
			b.exigerUneDecisionDiffusee(t, v, tranche, c.issue, c.motif)
			exigerNotificationConforme(t, b.attendreNotification(t, v, tranche, debut), tranche, c.issue, c.motif)
			if n := len(b.notificationsDeLArticle(t, v, tranche.ID)); n != 2 {
				t.Fatalf("%d notification(s) pour l'article, 2 attendues (à vérifier puis décision de l'admin)", n)
			}
		})
	}
}

func valeurOuVide(m *string) string {
	if m == nil {
		return ""
	}
	return *m
}

func (b *bancNotification) simulerUnePanneDeLEvenement(t *testing.T) func() {
	t.Helper()
	for _, instruction := range []string{
		`CREATE FUNCTION test_decision_indisponible() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.type_message = 'publication.decidee' THEN RAISE EXCEPTION 'outbox indisponible'; END IF; RETURN NEW; END $$`,
		`CREATE TRIGGER test_decision_indisponible BEFORE INSERT ON outbox FOR EACH ROW EXECUTE FUNCTION test_decision_indisponible()`,
	} {
		if _, err := b.base.maitre.Exec(t.Context(), instruction); err != nil {
			t.Fatalf("panne simulée de l'outbox : %v", err)
		}
	}
	retablie := false
	retablir := func() {
		if retablie {
			return
		}
		retablie = true
		if _, err := b.base.maitre.Exec(context.Background(), `DROP TRIGGER test_decision_indisponible ON outbox; DROP FUNCTION test_decision_indisponible()`); err != nil {
			t.Fatalf("fin de la panne simulée : %v", err)
		}
	}
	t.Cleanup(retablir)
	return retablir
}

func (b *bancNotification) nombreDeLignes(t *testing.T, table, id string) int {
	t.Helper()
	var n int
	if err := b.base.maitre.QueryRow(t.Context(), "SELECT count(*) FROM "+pgx.Identifier{table}.Sanitize()+" WHERE article_id = $1", id).Scan(&n); err != nil {
		t.Fatalf("lecture de %s : %v", table, err)
	}
	return n
}

func (b *bancNotification) ac1DecisionAdminAtomique(t *testing.T) {
	v := b.nouveauVendeur()
	debutDuControle := time.Now()
	aVerifier := b.articleAVerifier(t, v)
	b.attendreNotification(t, v, aVerifier, debutDuControle)
	retablir := b.simulerUnePanneDeLEvenement(t)
	statut := b.decider(t, b.admin.jeton, aVerifier.ID, corpsDecision(aVerifier.Revision, decisionRefuser, motif("article_interdit"))).statut
	retablir()
	if statut < http.StatusInternalServerError {
		t.Fatalf("décision acceptée (%d) alors que l'événement n'a pas pu être écrit", statut)
	}
	relu := b.relire(t, v.jeton, aVerifier.ID)
	exigerEtat(t, relu, etatAVerifier, motifCategorieInconnu)
	if relu.Revision != aVerifier.Revision || relu.ModifieLe != aVerifier.ModifieLe || b.nombreDeLignes(t, "decision_admin", aVerifier.ID) != 0 {
		t.Fatalf("décision partiellement appliquée : %+v, %d ligne(s) decision_admin", relu, b.nombreDeLignes(t, "decision_admin", aVerifier.ID))
	}
	if n := len(b.decisionsEnOutbox(t, aVerifier.ID)); n != 1 {
		t.Fatalf("%d décision(s) en outbox, seule celle du contrôle attendue", n)
	}
	time.Sleep(time.Second)
	if n := len(b.notificationsDeLArticle(t, v, aVerifier.ID)); n != 1 {
		t.Fatalf("%d notification(s), seule celle du contrôle attendue", n)
	}
	debut := time.Now()
	tranche := b.deciderAvecSucces(t, b.admin.jeton, aVerifier.ID, aVerifier.Revision, decisionRefuser, motif("article_interdit"))
	b.exigerUneDecisionDiffusee(t, v, tranche, issueRefuse, motif("article_interdit"))
	b.attendreNotification(t, v, tranche, debut)
}

func (b *bancNotification) ac1ResultatControleAtomique(t *testing.T) {
	v := b.nouveauVendeur()
	retablir := b.simulerUnePanneDeLEvenement(t)
	soumis := b.creerEtSoumettre(t, v, corpsComplet("Figurine Tournesol", categorieSure, 4500, 600))
	attendre(t, delaiDeTraitement, "résultat du contrôle publié", func() bool { return b.resultatPublie(t, soumis.ID) })
	time.Sleep(3 * time.Second)
	relu := b.relire(t, v.jeton, soumis.ID)
	exigerEtat(t, relu, etatEnControle, "")
	if relu.Revision != soumis.Revision || b.nombreDeLignes(t, "resultat_controle", soumis.ID) != 0 || len(b.decisionsEnOutbox(t, soumis.ID)) != 0 {
		t.Fatalf("résultat partiellement appliqué : %+v, %d ligne(s) resultat_controle, %d décision(s) en outbox",
			relu, b.nombreDeLignes(t, "resultat_controle", soumis.ID), len(b.decisionsEnOutbox(t, soumis.ID)))
	}
	if n := len(b.notificationsDeLArticle(t, v, soumis.ID)); n != 0 {
		t.Fatalf("%d notification(s) pour une décision non appliquée", n)
	}
	retablir()
	debut := time.Now()
	for time.Since(debut) < delaiDeTraitement && b.relire(t, v.jeton, soumis.ID).Etat == etatEnControle {
		time.Sleep(200 * time.Millisecond)
	}
	t.Logf("%s après le rétablissement : article %s, %d décision(s) en outbox", time.Since(debut).Round(time.Millisecond), b.relire(t, v.jeton, soumis.ID).Etat, len(b.decisionsEnOutbox(t, soumis.ID)))
}

func (b *bancNotification) resultatPublie(t *testing.T, id string) bool {
	t.Helper()
	for _, livraison := range b.canal.releverLEspion(t) {
		var resultat struct {
			Data resultatEmis `json:"data"`
		}
		if json.Unmarshal(livraison.Body, &resultat) == nil && resultat.Data.ArticleID == id {
			return true
		}
	}
	return false
}

func (b *bancNotification) ac1ac3ResultatDoubleOuPerime(t *testing.T) {
	v := b.nouveauVendeur()
	soumis := b.creerEtSoumettre(t, v, corpsComplet("Figurine Castafiore", categorieSure, 4500, 600))
	debut := time.Now()
	publie := b.attendreEtat(t, v, soumis.ID, etatPublie)
	b.attendreNotification(t, v, publie, debut)
	ignoresAvant := len(evenementsJournalises(b.journal, evenementIgnore))
	if err := b.canal.publier(t, identiteControle, versLesResultats(identiteControle, resultatForge(publie, issuePublie, nil))); err != nil {
		t.Fatalf("résultat en double : %v", err)
	}
	if err := b.canal.publier(t, identiteControle, versLesResultats(identiteControle, resultatForge(publie, issueRefuse, motif(motifCoordonnees)))); err != nil {
		t.Fatalf("second résultat pour la même révision : %v", err)
	}
	attendre(t, delaiDeTraitement, "résultats en double ignorés", func() bool {
		return len(evenementsJournalises(b.journal, evenementIgnore)) >= ignoresAvant+2
	})
	var modifie articleRecu
	debut = time.Now()
	b.modifier(t, v.jeton, publie.ID, corpsRevision(publie.Revision, corpsComplet("Figurine Castafiore et Nestor", categorieSure, 4500, 600))).exiger(t, http.StatusOK).lire(t, &modifie)
	republie := b.attendreEtat(t, v, publie.ID, etatPublie)
	b.attendreNotification(t, v, republie, debut)
	ignoresAvant = len(evenementsJournalises(b.journal, evenementIgnore))
	if err := b.canal.publier(t, identiteControle, versLesResultats(identiteControle, resultatForge(publie, issueRefuse, motif(motifCoordonnees)))); err != nil {
		t.Fatalf("résultat périmé : %v", err)
	}
	attendre(t, delaiDeTraitement, "résultat périmé ignoré", func() bool {
		return len(evenementsJournalises(b.journal, evenementIgnore)) >= ignoresAvant+1
	})
	time.Sleep(time.Second)
	if n := len(b.decisionsEnOutbox(t, publie.ID)); n != 2 {
		t.Fatalf("%d décision(s) en outbox, 2 attendues (une par révision appliquée)", n)
	}
	notifications := b.notificationsDeLArticle(t, v, publie.ID)
	if len(notifications) != 2 || notifications[0].Issue != issuePublie || notifications[1].Issue != issuePublie {
		t.Fatalf("notifications %+v, deux « publié » attendues (révisions %d et %d)", notifications, publie.Revision, republie.Revision)
	}
}

func (b *bancNotification) ac4SansContenuSaisi(t *testing.T) {
	v := b.nouveauVendeur()
	corps := strings.Replace(corpsComplet(titreSignature+" appelez le "+telephoneFictif, categorieSure, 3000, 0), "Belle pièce", "Description ZQX-7731 vendeur@exemple.fr", 1)
	debut := time.Now()
	soumis := b.creerEtSoumettre(t, v, corps)
	refuse := b.attendreEtat(t, v, soumis.ID, etatRefuse)
	b.exigerUneDecisionDiffusee(t, v, refuse, issueRefuse, motif(motifCoordonnees))
	b.attendreNotification(t, v, refuse, debut)
	r := b.reponseBruteDesNotifications(t, v)
	evenements := b.decisionsEnOutbox(t, refuse.ID)
	for _, revele := range []string{"ZQX-7731", "Tintin", telephoneFictif, "vendeur@exemple.fr", "Description"} {
		if strings.Contains(string(r), revele) {
			t.Errorf("la liste des notifications révèle %q : %s", revele, r)
		}
		for _, l := range evenements {
			if strings.Contains(string(l.corps), revele) {
				t.Errorf("la décision diffusée révèle %q : %s", revele, l.corps)
			}
		}
	}
}

func (b *bancNotification) reponseBruteDesNotifications(t *testing.T, v *vendeurNotifie) []byte {
	t.Helper()
	requete, err := http.NewRequestWithContext(t.Context(), http.MethodGet, b.api+cheminNotifications+"?taille=50", nil)
	if err != nil {
		t.Fatalf("requête : %v", err)
	}
	requete.Header.Set("Authorization", v.jetonEspace)
	return b.executer(t, requete).exiger(t, http.StatusOK).corps
}

func (b *bancNotification) ac6CloisonnementDesVendeurs(t *testing.T) {
	a, autre := b.nouveauVendeur(), b.nouveauVendeur()
	debut := time.Now()
	deA := b.creerEtSoumettre(t, a, corpsComplet("Figurine de A", categorieSure, 4500, 600))
	deB := b.creerEtSoumettre(t, autre, corpsComplet("Figurine de B", categorieSure, 4500, 600))
	b.attendreNotification(t, a, b.attendreEtat(t, a, deA.ID, etatPublie), debut)
	b.attendreNotification(t, autre, b.attendreEtat(t, autre, deB.ID, etatPublie), debut)
	for _, cas := range []struct {
		lecteur  *vendeurNotifie
		sien     string
		interdit string
	}{{a, deA.ID, deB.ID}, {autre, deB.ID, deA.ID}} {
		page := b.notificationsDe(t, cas.lecteur)
		var articles []string
		for _, n := range page.Elements {
			articles = append(articles, n.ArticleID)
		}
		if !slices.Contains(articles, cas.sien) || slices.Contains(articles, cas.interdit) || page.Total != 1 {
			t.Fatalf("le vendeur %s lit %v (total %d), attendu uniquement %s", cas.lecteur.sujet, articles, page.Total, cas.sien)
		}
	}
}

func exigerConnexionRefusee(t *testing.T, adresse, quoi string) {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), adresse)
	if err == nil {
		_ = conn.Close(t.Context())
		t.Fatalf("%s : connexion acceptée", quoi)
	}
	var erreur *pgconn.PgError
	if !errors.As(err, &erreur) || erreur.Code != droitInsuffisant {
		t.Fatalf("%s : %v ; refus 42501 attendu", quoi, err)
	}
	t.Logf("%s : %s", quoi, erreur.Message)
}

func (b *bancNotification) ac8BasesCloisonnees(t *testing.T) {
	cas := map[string]string{
		"interactions_execution sur marche": avecLaBase(b.interactions.execution, baseDuService),
		"interactions_migration sur marche": avecLaBase(b.interactions.migration, baseDuService),
		"marche_execution sur interactions": avecLaBase(b.base.execution, baseInteractions),
		"marche_migration sur interactions": avecLaBase(b.base.migration, baseInteractions),
	}
	for quoi, adresse := range cas {
		t.Run(quoi, func(t *testing.T) {
			exigerConnexionRefusee(t, adresse, quoi)
		})
	}
	t.Run("interactions_execution sans UPDATE, DELETE ni CREATE", func(t *testing.T) {
		execution := connecter(t.Context(), t, b.interactions.execution)
		for _, instruction := range []string{"UPDATE notification SET issue = 'publie'", "DELETE FROM notification", "TRUNCATE notification", "CREATE TABLE intrus (id integer)"} {
			_, err := execution.Exec(t.Context(), instruction)
			var erreur *pgconn.PgError
			if !errors.As(err, &erreur) || erreur.Code != droitInsuffisant {
				t.Errorf("%q devait être refusé (42501), obtenu %v", instruction, err)
			}
		}
	})
}

func (b *bancNotification) releverLEspion(t *testing.T) {
	t.Helper()
	for {
		livraison, presente, err := b.canal.admin.Get(fileEspionDeDecisions, true)
		if err != nil {
			t.Fatalf("relevé de l'abonnement de test : %v", err)
		}
		if !presente {
			return
		}
		b.espionnees = append(b.espionnees, livraison)
	}
}

func (b *bancNotification) decisionsPublieesEnOutbox(t *testing.T) [][]byte {
	t.Helper()
	lignes, err := b.base.maitre.Query(t.Context(), `SELECT corps FROM outbox WHERE type_message = $1 AND publie_le IS NOT NULL`, typeDecision)
	if err != nil {
		t.Fatalf("outbox : %v", err)
	}
	corps, err := pgx.CollectRows(lignes, pgx.RowTo[[]byte])
	if err != nil {
		t.Fatalf("outbox : %v", err)
	}
	return corps
}

func (b *bancNotification) vendeursDesDecisionsDiffusees(t *testing.T) map[string]int {
	t.Helper()
	vendeurs := map[string]int{}
	for _, livraison := range b.espionnees {
		if livraison.UserId != identiteMarche || livraison.RoutingKey != cleDecision || livraison.Exchange != echangeEvenements {
			t.Errorf("décision publiée en %q sur %s / %s", livraison.UserId, livraison.Exchange, livraison.RoutingKey)
		}
		if err := b.validateur.Validate(livraison.Body); err != nil {
			t.Errorf("décision diffusée non conforme au schéma v1 : %v", err)
			continue
		}
		var d decisionDiffusee
		enveloppe, err := b.validateur.Decode(livraison.Body, &d)
		if err != nil || enveloppe.Type != typeDecision || enveloppe.Version != 1 {
			t.Errorf("enveloppe %+v : %v", enveloppe, err)
		}
		vendeurs[d.SellerID]++
	}
	return vendeurs
}

func (b *bancNotification) ac9ContratDeLaPaire(t *testing.T) {
	attendre(t, delaiDeTraitement, "toutes les décisions consommées par Interactions", func() bool {
		return b.canal.compter(t, fileNotifications) == 0
	})
	b.releverLEspion(t)
	corpsEmis := b.decisionsPublieesEnOutbox(t)
	if len(corpsEmis) == 0 || len(b.espionnees) != len(corpsEmis) {
		t.Fatalf("%d décision(s) publiées en outbox, %d reçue(s) par l'abonnement de test", len(corpsEmis), len(b.espionnees))
	}
	vendeurs := b.vendeursDesDecisionsDiffusees(t)
	if n := b.canal.compter(t, fileNotificationsEcartee); n != 0 {
		t.Fatalf("%d décision(s) de la Place de marché écartée(s) par Interactions", n)
	}
	if b.journalInter.contient("notifications.decision_ecartee") {
		t.Fatalf("Interactions a écarté une décision de la Place de marché")
	}
	total := 0
	for sujet := range vendeurs {
		v := &vendeurNotifie{sujet: sujet, jetonEspace: bearer(b.jetons.emettre(sujet, []string{roleVendeur}, avecRevendication("aud", []string{identiteInteractions, "account"})))}
		total += b.notificationsDe(t, v).Total
	}
	if total != len(corpsEmis) {
		t.Fatalf("%d notification(s) créée(s) pour %d décision(s) diffusée(s)", total, len(corpsEmis))
	}
	t.Logf("%d décision(s) émises, conformes au schéma v1, toutes acceptées par Interactions (%d vendeurs)", len(corpsEmis), len(vendeurs))
}
