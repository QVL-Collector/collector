package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"collector/libs/platform/message"
	"collector/services/place-de-marche/internal/migrations"
)

const (
	fileEspionDeDemandes  = "test.espion-demandes-us014"
	enteteTraceparent     = "traceparent"
	enteteCorrelation     = "X-Correlation-Id"
	evenementApplique     = "controle.resultat_applique"
	jetonUtilisateurPiege = "eyJhbGciOiJSUzI1NiJ9" + "." + "eyJzdWIiOiJwaWVnZS11czAxNC11dGlsaXNhdGV1ciJ9" + "." + "UGllZ2VVUzAxNFV0aWxpc2F0ZXVyU2lnbmF0dXJl"
	cookiePiege           = "SessionPiegeeUS014-c5V"
	signatureAWSPiegee    = "a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f9a"
	jetonSessionAWSPiege  = "IQoJb3JpZ2luUS014PiegeSessionAWS0123456789"
	basicPiege            = "vendeur-us014:MdpBasicPiegeUS014"
	marqueurDeForge       = "forge-us014"
)

var (
	formatTrace       = regexp.MustCompile(`^[0-9a-f]{32}$`)
	formatTraceparent = regexp.MustCompile(`^00-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})$`)
)

type journalDeService struct {
	service string
	journal *journalPartage
}

type idsDuParcours struct {
	correlation string
	trace       string
}

type bancDesJournaux struct {
	*bancNotification
	marche       journalDeService
	controle     journalDeService
	interactions journalDeService
	jetonsVus    []string
	secrets      []string
}

func demarrerBancDesJournaux(t *testing.T) *bancDesJournaux {
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
	poserUnEspionDeDemandes(t, canal)
	binaireInteractions := construireBinaire(t, "../../../interactions/cmd/interactions", "interactions")
	migrerInteractions(t, binaireInteractions, interactions)
	emetteur := nouvelEmetteurDeJetons(t)
	e, journal := lancerPlaceDeMarche(t, canal, base.execution, emetteur)
	if _, err := base.maitre.Exec(ctx, `INSERT INTO categorie (id, libelle) VALUES ($1, 'Timbres')`, categorieNonReconnue); err != nil {
		t.Fatalf("catégorie inconnue du Contrôle : %v", err)
	}
	controle := lancerControle(t, construireControle(t), canal)
	instance := lancerProcessusInteractions(t, binaireInteractions, canal, interactions, emetteur).attendreDisponibilite(t)
	b := &bancDesJournaux{
		bancNotification: &bancNotification{
			environnement: e,
			canal:         canal,
			base:          base,
			interactions:  interactions,
			journal:       journal,
			validateur:    validateurDeContrats(t),
			api:           instance.api,
			journalInter:  instance.journal,
			admin:         e.administrateur(),
		},
		marche:       journalDeService{service: serviceName, journal: journal},
		controle:     journalDeService{service: identiteControle, journal: controle},
		interactions: journalDeService{service: identiteInteractions, journal: instance.journal},
	}
	b.secrets = append(b.secrets, motsDePasseDesAdresses(t, base.execution, base.migration, interactions.execution, interactions.migration)...)
	for _, secret := range canal.secrets {
		b.secrets = append(b.secrets, secret)
	}
	return b
}

func motsDePasseDesAdresses(t *testing.T, adresses ...string) []string {
	t.Helper()
	var motsDePasse []string
	for _, adresse := range adresses {
		lue, err := url.Parse(adresse)
		if err != nil {
			t.Fatalf("adresse de base : %v", err)
		}
		if motDePasse, present := lue.User.Password(); present && motDePasse != "" {
			motsDePasse = append(motsDePasse, motDePasse)
		}
	}
	return motsDePasse
}

func poserUnEspionDeDemandes(t *testing.T, canal *canalDeTest) {
	t.Helper()
	if _, err := canal.admin.QueueDeclare(fileEspionDeDemandes, true, false, false, false, amqp.Table{"x-queue-type": "classic"}); err != nil {
		t.Fatalf("espion des demandes : %v", err)
	}
	if err := canal.admin.QueueBind(fileEspionDeDemandes, cleDeRoutageV1, echangeDemandes, false, nil); err != nil {
		t.Fatalf("liaison de l'espion des demandes : %v", err)
	}
}

func (b *bancDesJournaux) releverFile(t *testing.T, file string) []amqp.Delivery {
	t.Helper()
	var relevees []amqp.Delivery
	for {
		livraison, presente, err := b.canal.admin.Get(file, true)
		if err != nil {
			t.Fatalf("relevé de %s : %v", file, err)
		}
		if !presente {
			return relevees
		}
		relevees = append(relevees, livraison)
	}
}

func (b *bancDesJournaux) requete(t *testing.T, methode, chemin, jeton, corps string, entetes map[string]string) reponse {
	t.Helper()
	requete, err := http.NewRequestWithContext(t.Context(), methode, b.adresse+chemin, strings.NewReader(corps))
	if err != nil {
		t.Fatalf("requête %s %s : %v", methode, chemin, err)
	}
	requete.Header.Set("Content-Type", typeJSON)
	requete.Header.Set("Authorization", jeton)
	for nom, valeur := range entetes {
		requete.Header.Set(nom, valeur)
	}
	b.jetonsVus = append(b.jetonsVus, strings.TrimPrefix(jeton, "Bearer "))
	return b.executer(t, requete)
}

func entetesDeTrace(correlation, trace string) map[string]string {
	return map[string]string{enteteCorrelation: correlation, enteteTraceparent: "00-" + trace + "-" + nouveauSpan() + "-01"}
}

func nouvelleTrace() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

func nouveauSpan() string {
	return nouvelleTrace()[:16]
}

func entetesPieges(base map[string]string) map[string]string {
	entetes := map[string]string{
		"X-Collector-User-Token": jetonUtilisateurPiege,
		"Cookie":                 "session=" + cookiePiege,
		"Proxy-Authorization":    "Basic " + base64.StdEncoding.EncodeToString([]byte(basicPiege)),
	}
	for nom, valeur := range base {
		entetes[nom] = valeur
	}
	return entetes
}

func parametresPresignes() string {
	return "?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Signature=" + signatureAWSPiegee + "&X-Amz-Security-Token=" + jetonSessionAWSPiege
}

type parcours struct {
	vendeur  *vendeurNotifie
	creation idsDuParcours
	envoi    idsDuParcours
	article  articleRecu
	debut    time.Time
}

func (b *bancDesJournaux) creerAvec(t *testing.T, v *vendeurNotifie, corps string) (articleRecu, idsDuParcours) {
	t.Helper()
	trace := nouvelleTrace()
	var cree articleRecu
	r := b.requete(t, http.MethodPost, cheminArticles, v.jeton, corps, entetesDeTrace(uuid.NewString(), trace)).exiger(t, http.StatusCreated)
	r.lire(t, &cree)
	return cree, idsDuParcours{correlation: r.entetes.Get(enteteCorrelation), trace: trace}
}

func (b *bancDesJournaux) soumettreAvec(t *testing.T, corps string, entetes map[string]string, suffixe string) parcours {
	t.Helper()
	v := b.nouveauVendeur()
	cree, creation := b.creerAvec(t, v, corps)
	debut := time.Now()
	var soumis articleRecu
	r := b.requete(t, http.MethodPost, cheminSoumission(cree.ID)+suffixe, v.jeton, fmt.Sprintf(`{"revision":%d}`, cree.Revision), entetes).exiger(t, http.StatusAccepted)
	r.lire(t, &soumis)
	correlation := r.entetes.Get(enteteCorrelation)
	if _, err := uuid.Parse(correlation); err != nil {
		t.Fatalf("X-Correlation-Id renvoyé %q n'est pas un UUID", correlation)
	}
	return parcours{vendeur: v, creation: creation, envoi: idsDuParcours{correlation: correlation}, article: soumis, debut: debut}
}

func lignesJSON(j *journalPartage) ([]map[string]any, []string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	var lignes []map[string]any
	var illisibles []string
	lecteur := bufio.NewScanner(strings.NewReader(j.tampon.String()))
	lecteur.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for lecteur.Scan() {
		var ligne map[string]any
		if json.Unmarshal(lecteur.Bytes(), &ligne) == nil {
			lignes = append(lignes, ligne)
			continue
		}
		illisibles = append(illisibles, lecteur.Text())
	}
	return lignes, illisibles
}

func lignesPortant(j journalDeService, valeur string) []map[string]any {
	lignes, _ := lignesJSON(j.journal)
	var portant []map[string]any
	for _, l := range lignes {
		if slices.Contains(feuilles(l), valeur) || strings.Contains(fmt.Sprint(l["msg"]), valeur) {
			portant = append(portant, l)
		}
	}
	return portant
}

func lignesDeCorrelation(j journalDeService, correlation string) []map[string]any {
	lignes, _ := lignesJSON(j.journal)
	var trouvees []map[string]any
	for _, l := range lignes {
		if l["correlation_id"] == correlation {
			trouvees = append(trouvees, l)
		}
	}
	return trouvees
}

func traceDe(t *testing.T, j journalDeService, correlation string) string {
	t.Helper()
	lignes := lignesDeCorrelation(j, correlation)
	if len(lignes) == 0 {
		t.Fatalf("aucune ligne de %s pour la corrélation %s", j.service, correlation)
	}
	trace, _ := lignes[0]["trace_id"].(string)
	if !formatTrace.MatchString(trace) || trace == strings.Repeat("0", 32) {
		t.Fatalf("trace_id %q invalide dans %v", trace, lignes[0])
	}
	return trace
}

func exigerLignesDuParcours(t *testing.T, j journalDeService, article string, attendus ...idsDuParcours) {
	t.Helper()
	portant := lignesPortant(j, article)
	if len(portant) == 0 {
		t.Errorf("aucune ligne de %s ne porte l'article %s", j.service, article)
	}
	for _, l := range portant {
		if l["service"] != j.service {
			t.Errorf("ligne de %s avec service %v : %v", j.service, l["service"], l)
		}
		retenu := slices.IndexFunc(attendus, func(a idsDuParcours) bool { return a.correlation == l["correlation_id"] })
		if retenu < 0 {
			t.Errorf("ligne de %s sur %s hors des corrélations du parcours %v : %v", j.service, article, attendus, l)
			continue
		}
		if l["trace_id"] != attendus[retenu].trace {
			t.Errorf("ligne de %s : trace_id %v, %s attendu : %v", j.service, l["trace_id"], attendus[retenu].trace, l)
		}
	}
}

func exigerLignesDeCorrelation(t *testing.T, j journalDeService, attendu idsDuParcours, minimum int) {
	t.Helper()
	lignes := lignesDeCorrelation(j, attendu.correlation)
	if len(lignes) < minimum {
		t.Errorf("%d ligne(s) de %s pour la corrélation %s, au moins %d attendue(s)", len(lignes), j.service, attendu.correlation, minimum)
	}
	for _, l := range lignes {
		if l["service"] != j.service || l["trace_id"] != attendu.trace {
			t.Errorf("ligne de %s : service %v, trace %v ; %s et %s attendus : %v", j.service, l["service"], l["trace_id"], j.service, attendu.trace, l)
		}
	}
}

func traceDuMessage(t *testing.T, livraison amqp.Delivery) string {
	t.Helper()
	valeur, _ := livraison.Headers[enteteTraceparent].(string)
	morceaux := formatTraceparent.FindStringSubmatch(valeur)
	if morceaux == nil {
		t.Fatalf("message %s sans en-tête traceparent W3C : %q", livraison.MessageId, valeur)
	}
	return morceaux[1]
}

func (b *bancDesJournaux) messagesDe(t *testing.T, file, article string) []amqp.Delivery {
	t.Helper()
	var trouves []amqp.Delivery
	for _, livraison := range b.releverFile(t, file) {
		if bytes.Contains(livraison.Body, []byte(article)) {
			trouves = append(trouves, livraison)
		}
	}
	return trouves
}

func (b *bancDesJournaux) exigerMessagesCorreles(t *testing.T, file, article string, attendu idsDuParcours) {
	t.Helper()
	messages := b.messagesDe(t, file, article)
	if len(messages) == 0 {
		t.Errorf("aucun message de %s sur %s", file, article)
	}
	for _, m := range messages {
		var enveloppe message.Envelope
		if err := json.Unmarshal(m.Body, &enveloppe); err != nil {
			t.Fatalf("enveloppe illisible : %v", err)
		}
		if enveloppe.CorrelationID != attendu.correlation || m.CorrelationId != attendu.correlation {
			t.Errorf("message de %s : correlation_id %q (enveloppe) / %q (propriété), %s attendu", file, enveloppe.CorrelationID, m.CorrelationId, attendu.correlation)
		}
		if trace := traceDuMessage(t, m); trace != attendu.trace {
			t.Errorf("message de %s : trace %s, %s attendue", file, trace, attendu.trace)
		}
	}
}

type ligneTracee struct {
	typ         string
	correlation string
	traceparent *string
	publiee     bool
}

func (b *bancDesJournaux) outboxTracee(t *testing.T, article string) []ligneTracee {
	t.Helper()
	lignes, err := b.base.maitre.Query(t.Context(), `
		SELECT type_message, correlation_id::text, traceparent, publie_le IS NOT NULL FROM outbox
		WHERE convert_from(corps, 'UTF8')::jsonb -> 'data' ->> 'article_id' = $1 ORDER BY cree_le`, article)
	if err != nil {
		t.Fatalf("lecture de l'outbox : %v", err)
	}
	defer lignes.Close()
	var resultat []ligneTracee
	for lignes.Next() {
		var l ligneTracee
		if err := lignes.Scan(&l.typ, &l.correlation, &l.traceparent, &l.publiee); err != nil {
			t.Fatalf("ligne d'outbox : %v", err)
		}
		resultat = append(resultat, l)
	}
	return resultat
}

func (b *bancDesJournaux) exigerOutboxTracee(t *testing.T, article string, attendu idsDuParcours, types ...string) {
	t.Helper()
	lignes := b.outboxTracee(t, article)
	for _, typ := range types {
		trouve := false
		for _, l := range lignes {
			if l.typ != typ {
				continue
			}
			trouve = true
			if l.correlation != attendu.correlation || l.traceparent == nil || formatTraceparent.FindStringSubmatch(*l.traceparent) == nil || formatTraceparent.FindStringSubmatch(*l.traceparent)[1] != attendu.trace {
				t.Errorf("outbox %s : correlation %s, traceparent %v ; %s et trace %s attendus", typ, l.correlation, l.traceparent, attendu.correlation, attendu.trace)
			}
		}
		if !trouve {
			t.Errorf("aucune ligne d'outbox %s pour %s", typ, article)
		}
	}
}

func (b *bancDesJournaux) exigerAuditDuResultat(t *testing.T, decide articleRecu, issue string, motifAttendu *string, attendu idsDuParcours) {
	t.Helper()
	var retenus []map[string]any
	attendre(t, 5*time.Second, "audit de "+decide.ID, func() bool {
		retenus = nil
		for _, ev := range evenementsJournalises(b.journal, evenementApplique) {
			if slices.Contains(feuilles(ev), decide.ID) {
				retenus = append(retenus, ev)
			}
		}
		return len(retenus) > 0
	})
	if len(retenus) != 1 {
		t.Fatalf("%d événement(s) %s pour %s, un attendu : %v", len(retenus), evenementApplique, decide.ID, retenus)
	}
	exigerEvenementDAudit(t, retenus[0], decide, issue, motifAttendu)
	if retenus[0]["correlation_id"] != attendu.correlation || retenus[0]["trace_id"] != attendu.trace {
		t.Errorf("audit sans la corrélation %s et la trace %s du parcours : %v", attendu.correlation, attendu.trace, retenus[0])
	}
}

func exigerEvenementDAudit(t *testing.T, ev map[string]any, decide articleRecu, issue string, motifAttendu *string) {
	t.Helper()
	if ev["audit"] != true {
		t.Errorf("événement sans audit=true : %v", ev)
	}
	horodatage, err := time.Parse(time.RFC3339Nano, fmt.Sprint(ev["time"]))
	if err != nil || time.Since(horodatage) > 5*time.Minute || time.Until(horodatage) > time.Minute {
		t.Errorf("horodatage %v invalide : %v", ev["time"], err)
	}
	valeurs := feuilles(ev)
	for _, attendue := range []string{decide.ID, fmt.Sprint(decide.Revision), issue} {
		if attendue != "" && !slices.Contains(valeurs, attendue) {
			t.Errorf("audit %v sans la valeur %q (article, révision, issue)", ev, attendue)
		}
	}
	if motifAttendu != nil && !slices.Contains(valeurs, *motifAttendu) {
		t.Errorf("audit %v sans le motif %q", ev, *motifAttendu)
	}
}

func (b *bancDesJournaux) suivreJusquALaDecision(t *testing.T, p parcours, etat, issue string, motifAttendu *string) articleRecu {
	t.Helper()
	decide := b.attendreEtat(t, p.vendeur, p.article.ID, etat)
	b.exigerUneDecisionDiffusee(t, p.vendeur, decide, issue, motifAttendu)
	b.attendreNotification(t, p.vendeur, decide, p.debut)
	attendre(t, 5*time.Second, "journal d'Interactions sur "+decide.ID, func() bool { return len(lignesPortant(b.interactions, decide.ID)) > 0 })
	return decide
}

func (b *bancDesJournaux) exigerParcoursCorrele(t *testing.T, p parcours, decide articleRecu, issue string, motifAttendu *string) {
	t.Helper()
	exigerLignesDeCorrelation(t, b.marche, p.envoi, 2)
	exigerLignesDeCorrelation(t, b.controle, p.envoi, 1)
	exigerLignesDuParcours(t, b.marche, p.article.ID, p.creation, p.envoi)
	exigerLignesDuParcours(t, b.controle, p.article.ID, p.envoi)
	exigerLignesDuParcours(t, b.interactions, p.article.ID, p.envoi)
	b.exigerOutboxTracee(t, p.article.ID, p.envoi, typeDemande, typeDecision)
	b.exigerMessagesCorreles(t, fileEspionDeDemandes, p.article.ID, p.envoi)
	b.exigerMessagesCorreles(t, fileEspionDeResultats, p.article.ID, p.envoi)
	b.exigerMessagesCorreles(t, fileEspionDeDecisions, p.article.ID, p.envoi)
	b.exigerAuditDuResultat(t, decide, issue, motifAttendu, p.envoi)
	t.Logf("corrélation %s, trace %s : %d ligne(s) Place de marché, %d Contrôle, %d Interactions sur l'article ; exemple Contrôle %v",
		p.envoi.correlation, p.envoi.trace, len(lignesPortant(b.marche, p.article.ID)), len(lignesPortant(b.controle, p.article.ID)), len(lignesPortant(b.interactions, p.article.ID)), lignesPortant(b.controle, p.article.ID)[0])
}

func TestUS014JournauxCorrelesDeLaSoumissionAuControle(t *testing.T) {
	b := demarrerBancDesJournaux(t)
	t.Run("AC1 AC4 traceparent et corrélation reçus suivis de la soumission à la publication", b.ac1ac4EnTetesRecusSuivis)
	t.Run("AC1 AC4 sans en-tête : identifiants tirés à la soumission et suivis jusqu'au refus", b.ac1ac4IdentifiantsTiresSuivis)
	t.Run("AC1 AC4 en-têtes mal formés remplacés puis suivis jusqu'à à vérifier", b.ac1ac4EnTetesMalFormesRemplaces)
	t.Run("AC4 décision de l'administrateur auditée avec sa propre corrélation", b.ac4DecisionAdministrateurAuditee)
	t.Run("AC1 message du canal sans traceparent : trace tirée, corrélation de l'enveloppe recopiée", b.ac1MessageSansTraceparent)
	t.Run("AC1 message du canal avec traceparent : trace reprise jusqu'au résultat", b.ac1MessageAvecTraceparent)
	t.Run("AC1 migration 00010 : ligne d'outbox sans traceparent relayée et traitée", b.ac1LigneDOutboxHeriteeRelayee)
	t.Run("AC6 contenu et en-têtes forgés ne produisent ni audit ni service usurpé", b.ac6ContenuForgeSansAuditNiServiceUsurpe)
	t.Run("AC1 chaque ligne JSON des services porte service", b.ac1ChaqueLignePorteService)
	t.Run("AC5 aucune valeur piégée ni secret dans les journaux des trois services", b.ac5AucuneValeurPiegee)
}

func (b *bancDesJournaux) ac1ac4EnTetesRecusSuivis(t *testing.T) {
	envoi := idsDuParcours{correlation: uuid.NewString(), trace: nouvelleTrace()}
	p := b.soumettreAvec(t, corpsComplet("Figurine Tournesol", categorieSure, 4500, 600), entetesPieges(entetesDeTrace(envoi.correlation, envoi.trace)), parametresPresignes())
	if p.envoi.correlation != envoi.correlation {
		t.Fatalf("X-Correlation-Id renvoyé %s, %s attendu", p.envoi.correlation, envoi.correlation)
	}
	p.envoi.trace = envoi.trace
	decide := b.suivreJusquALaDecision(t, p, etatPublie, issuePublie, nil)
	b.exigerParcoursCorrele(t, p, decide, issuePublie, nil)
}

func (b *bancDesJournaux) ac1ac4IdentifiantsTiresSuivis(t *testing.T) {
	p := b.soumettreAvec(t, corpsComplet("Figurine Lampion, appelez le "+telephoneFictif, categorieSure, 3000, 0), entetesPieges(nil), "")
	p.envoi.trace = traceDe(t, b.marche, p.envoi.correlation)
	decide := b.suivreJusquALaDecision(t, p, etatRefuse, issueRefuse, motif(motifCoordonnees))
	b.exigerParcoursCorrele(t, p, decide, issueRefuse, motif(motifCoordonnees))
}

func (b *bancDesJournaux) ac1ac4EnTetesMalFormesRemplaces(t *testing.T) {
	recus := map[string]string{
		enteteCorrelation: "pas-un-uuid-" + marqueurDeForge,
		enteteTraceparent: "00-" + strings.Repeat("0", 32) + "-" + nouveauSpan() + "-01",
	}
	p := b.soumettreAvec(t, corpsComplet("Timbre Séraphin", categorieNonReconnue, 4500, 600), entetesPieges(recus), "")
	if p.envoi.correlation == recus[enteteCorrelation] {
		t.Fatalf("corrélation mal formée reprise")
	}
	p.envoi.trace = traceDe(t, b.marche, p.envoi.correlation)
	decide := b.suivreJusquALaDecision(t, p, etatAVerifier, "a_verifier", motif(motifCategorieInconnu))
	b.exigerParcoursCorrele(t, p, decide, "a_verifier", motif(motifCategorieInconnu))
	for _, j := range []journalDeService{b.marche, b.controle, b.interactions} {
		if j.journal.contient("pas-un-uuid-" + marqueurDeForge) {
			t.Errorf("corrélation mal formée présente dans le journal de %s", j.service)
		}
	}
}

func (b *bancDesJournaux) ac4DecisionAdministrateurAuditee(t *testing.T) {
	v := b.nouveauVendeur()
	aVerifier := b.articleAVerifier(t, v)
	b.exigerUneDecisionDiffusee(t, v, aVerifier, "a_verifier", motif(motifCategorieInconnu))
	b.releverFile(t, fileEspionDeDecisions)
	decision := idsDuParcours{correlation: uuid.NewString(), trace: nouvelleTrace()}
	var tranche articleRecu
	b.requete(t, http.MethodPost, cheminDecision(aVerifier.ID), b.admin.jeton, corpsDecision(aVerifier.Revision, decisionPublier, nil), entetesDeTrace(decision.correlation, decision.trace)).exiger(t, http.StatusOK).lire(t, &tranche)
	var audits []map[string]any
	for _, ev := range evenementsJournalises(b.journal, evenementDecisionAdmin) {
		if slices.Contains(feuilles(ev), aVerifier.ID) {
			audits = append(audits, ev)
		}
	}
	if len(audits) != 1 {
		t.Fatalf("%d événement(s) %s pour %s, un attendu", len(audits), evenementDecisionAdmin, aVerifier.ID)
	}
	exigerEvenementDAudit(t, audits[0], aVerifier, "", nil)
	if valeurs := feuilles(audits[0]); !slices.Contains(valeurs, decisionPublier) && !slices.Contains(valeurs, issuePublie) {
		t.Errorf("audit de décision sans l'issue : %v", audits[0])
	}
	if audits[0]["correlation_id"] != decision.correlation || audits[0]["trace_id"] != decision.trace || !slices.Contains(feuilles(audits[0]), b.admin.sujet) {
		t.Errorf("audit de décision sans la corrélation, la trace ou l'administrateur : %v", audits[0])
	}
	b.exigerUneDecisionDiffusee(t, v, tranche, issuePublie, nil)
	b.exigerMessagesCorreles(t, fileEspionDeDecisions, aVerifier.ID, decision)
}

func (b *bancDesJournaux) demandePubliee(t *testing.T) (articleRecu, []byte) {
	t.Helper()
	p := b.soumettreAvec(t, corpsComplet("Figurine Abdallah", categorieSure, 4500, 600), nil, "")
	b.attendreEtat(t, p.vendeur, p.article.ID, etatPublie)
	for _, l := range b.demandesEnOutbox(t, p.article.ID) {
		return p.article, l.corps
	}
	t.Fatalf("aucune demande en outbox pour %s", p.article.ID)
	return articleRecu{}, nil
}

func (b *bancDesJournaux) demandesEnOutbox(t *testing.T, id string) []ligneOutbox {
	t.Helper()
	return lignesDOutbox(t, b.base, id, typeDemande)
}

func enveloppeRecorrelee(t *testing.T, corps []byte, id, correlation string) []byte {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(corps, &document); err != nil {
		t.Fatalf("demande : %v", err)
	}
	document["id"], document["correlation_id"] = id, correlation
	modifie, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("demande : %v", err)
	}
	return modifie
}

func (b *bancDesJournaux) publierDemande(t *testing.T, id, correlationPropriete string, corps []byte, entetes amqp.Table) {
	t.Helper()
	canal := b.canal.ouvrir(t, identiteMarche)
	if err := canal.Confirm(false); err != nil {
		t.Fatalf("confirmations : %v", err)
	}
	confirmation, err := canal.PublishWithDeferredConfirmWithContext(contexteBorne(t, 10*time.Second), echangeDemandes, cleDeRoutageV1, true, false, amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: id, UserId: identiteMarche,
		CorrelationId: correlationPropriete, Headers: entetes, Type: typeDemande, Body: corps,
	})
	if err != nil || !confirmation.Wait() {
		t.Fatalf("publication de la demande : %v", err)
	}
}

func (b *bancDesJournaux) exigerTraitementDuMessage(t *testing.T, id string, attendu idsDuParcours) {
	t.Helper()
	attendre(t, delaiDeTraitement, "traitement de "+id+" par le Contrôle", func() bool { return len(lignesPortant(b.controle, id)) > 0 })
	lignes := lignesPortant(b.controle, id)
	if attendu.trace == "" {
		attendu.trace, _ = lignes[0]["trace_id"].(string)
	}
	if !formatTrace.MatchString(attendu.trace) || attendu.trace == strings.Repeat("0", 32) {
		t.Errorf("Contrôle : trace_id %q invalide : %v", attendu.trace, lignes[0])
	}
	for _, l := range lignes {
		if l["correlation_id"] != attendu.correlation || l["trace_id"] != attendu.trace {
			t.Errorf("Contrôle : correlation_id %v, trace_id %v ; %s (enveloppe) et %s attendus : %v", l["correlation_id"], l["trace_id"], attendu.correlation, attendu.trace, l)
		}
	}
	resultat := b.resultatDeCorrelation(t, id, attendu.correlation)
	if trace := traceDuMessage(t, resultat); trace != attendu.trace {
		t.Errorf("résultat : trace %s, %s attendue", trace, attendu.trace)
	}
	if resultat.CorrelationId != attendu.correlation {
		t.Errorf("résultat : propriété correlation-id %q, %s attendu", resultat.CorrelationId, attendu.correlation)
	}
}

func (b *bancDesJournaux) resultatDeCorrelation(t *testing.T, id, correlation string) amqp.Delivery {
	t.Helper()
	var resultat []amqp.Delivery
	attendre(t, delaiDeTraitement, "résultat de "+id, func() bool {
		for _, m := range b.releverFile(t, fileEspionDeResultats) {
			var enveloppe message.Envelope
			if json.Unmarshal(m.Body, &enveloppe) == nil && enveloppe.CorrelationID == correlation {
				resultat = append(resultat, m)
			}
		}
		return len(resultat) > 0
	})
	return resultat[0]
}

func (b *bancDesJournaux) ac1MessageSansTraceparent(t *testing.T) {
	_, corps := b.demandePubliee(t)
	b.releverFile(t, fileEspionDeResultats)
	id, correlation := uuid.NewString(), uuid.NewString()

	b.publierDemande(t, id, uuid.NewString(), enveloppeRecorrelee(t, corps, id, correlation), nil)

	b.exigerTraitementDuMessage(t, id, idsDuParcours{correlation: correlation})
}

func (b *bancDesJournaux) ac1MessageAvecTraceparent(t *testing.T) {
	_, corps := b.demandePubliee(t)
	b.releverFile(t, fileEspionDeResultats)
	id, attendu := uuid.NewString(), idsDuParcours{correlation: uuid.NewString(), trace: nouvelleTrace()}

	b.publierDemande(t, id, attendu.correlation, enveloppeRecorrelee(t, corps, id, attendu.correlation), amqp.Table{enteteTraceparent: "00-" + attendu.trace + "-" + nouveauSpan() + "-01"})

	b.exigerTraitementDuMessage(t, id, attendu)
}

func (b *bancDesJournaux) ac1LigneDOutboxHeriteeRelayee(t *testing.T) {
	_, corps := b.demandePubliee(t)
	b.releverFile(t, fileEspionDeResultats)
	id, correlation := uuid.NewString(), uuid.NewString()
	if _, err := b.base.maitre.Exec(t.Context(), `
		INSERT INTO outbox (id, echange, cle_routage, type_message, correlation_id, corps, traceparent)
		VALUES ($1, $2, $3, $4, $5, $6, NULL)`, id, echangeDemandes, cleDeRoutageV1, typeDemande, correlation, enveloppeRecorrelee(t, corps, id, correlation)); err != nil {
		t.Fatalf("ligne d'outbox héritée : %v", err)
	}

	attendre(t, delaiDeTraitement, "relais de la ligne héritée", func() bool {
		var publiee bool
		_ = b.base.maitre.QueryRow(t.Context(), `SELECT publie_le IS NOT NULL FROM outbox WHERE id = $1`, id).Scan(&publiee)
		return publiee
	})
	b.exigerTraitementDuMessage(t, id, idsDuParcours{correlation: correlation})
}

func (b *bancDesJournaux) ac6ContenuForgeSansAuditNiServiceUsurpe(t *testing.T) {
	injection := "\n{\"audit\":true,\"service\":\"controle\",\"event\":\"" + evenementApplique + "\",\"article_id\":\"" + marqueurDeForge + "\"}\n"
	entetes := map[string]string{enteteCorrelation: strings.ReplaceAll(injection, "\n", " "), "Service": "controle", "Audit": "true"}
	p := b.soumettreAvec(t, corpsComplet("Figurine Rastapopoulos"+strings.TrimSpace(injection[:40]), categorieSure, 4500, 600), entetes, "")
	b.attendreEtat(t, p.vendeur, p.article.ID, etatPublie)
	attendre(t, 5*time.Second, "audit de "+p.article.ID, func() bool { return len(evenementsPortant(b.journal, evenementApplique, p.article.ID)) == 1 })
	for _, j := range []journalDeService{b.marche, b.controle, b.interactions} {
		lignes, _ := lignesJSON(j.journal)
		for _, l := range lignes {
			if l["service"] != j.service {
				t.Errorf("ligne de %s avec service %v : %v", j.service, l["service"], l)
			}
			if slices.Contains(feuilles(l), marqueurDeForge) && l["audit"] == true {
				t.Errorf("ligne d'audit forgée par le contenu reçu dans %s : %v", j.service, l)
			}
		}
	}
	if n := len(evenementsPortant(b.journal, evenementApplique, p.article.ID)); n != 1 {
		t.Errorf("%d audit(s) %s pour %s, un seul attendu", n, evenementApplique, p.article.ID)
	}
}

func (b *bancDesJournaux) ac1ChaqueLignePorteService(t *testing.T) {
	for _, j := range []journalDeService{b.marche, b.controle, b.interactions} {
		lignes, illisibles := lignesJSON(j.journal)
		if len(lignes) == 0 {
			t.Errorf("aucune ligne JSON pour %s", j.service)
		}
		for _, l := range lignes {
			if l["service"] != j.service {
				t.Errorf("ligne de %s sans service %s : %v", j.service, j.service, l)
			}
		}
		for _, l := range illisibles {
			if !strings.HasPrefix(l, "arrêt : ") {
				t.Errorf("ligne non JSON dans %s : %q", j.service, l)
			}
		}
	}
}

func (b *bancDesJournaux) ac5AucuneValeurPiegee(t *testing.T) {
	b.requete(t, http.MethodGet, cheminArticles+parametresPresignes(), "Basic "+base64.StdEncoding.EncodeToString([]byte(basicPiege)), "", entetesPieges(nil))
	b.requete(t, http.MethodGet, cheminArticles, "Bearer "+jetonUtilisateurPiege, "", nil)
	pieges := append([]string{jetonUtilisateurPiege, cookiePiege, signatureAWSPiegee, jetonSessionAWSPiege, base64.StdEncoding.EncodeToString([]byte(basicPiege)), "MdpBasicPiegeUS014"}, b.secrets...)
	for _, jeton := range b.jetonsVus {
		if len(jeton) > 20 {
			pieges = append(pieges, jeton)
		}
	}
	time.Sleep(time.Second)
	for _, j := range []journalDeService{b.marche, b.controle, b.interactions} {
		for _, piege := range pieges {
			if j.journal.contient(piege) {
				t.Errorf("valeur piégée %q présente dans le journal de %s", tronquer(piege), j.service)
			}
		}
	}
	t.Logf("%d valeurs piégées cherchées dans %d, %d et %d octets de journaux", len(pieges), longueur(b.marche.journal), longueur(b.controle.journal), longueur(b.interactions.journal))
}

func tronquer(valeur string) string {
	if len(valeur) > 12 {
		return valeur[:12] + "…"
	}
	return valeur
}

func longueur(j *journalPartage) int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.tampon.Len()
}
