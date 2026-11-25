package logging_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/libs/platform/logging"
)

const (
	traceRecue        = "4bf92f3577b34da6a3ce929d0e0e4736"
	spanRecu          = "00f067aa0ba902b7"
	traceparentRecu   = "00-" + traceRecue + "-" + spanRecu + "-01"
	correlationRecue  = "8d1f6a2e-3b4c-4d5e-9f60-718293a4b5c6"
	serviceDeTest     = "service-us014"
	messageDuTraiteur = "traitement de la requête"
)

var (
	formatTraceID     = regexp.MustCompile(`^[0-9a-f]{32}$`)
	formatSpanID      = regexp.MustCompile(`^[0-9a-f]{16}$`)
	formatTraceparent = regexp.MustCompile(`^00-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})$`)
)

type ligneJournal map[string]any

func journalJSON(t *testing.T, service string) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	sortie := &bytes.Buffer{}
	return logging.New(service, sortie, slog.LevelDebug), sortie
}

func lignesDe(t *testing.T, sortie *bytes.Buffer) []ligneJournal {
	t.Helper()
	var lignes []ligneJournal
	lecteur := bufio.NewScanner(strings.NewReader(sortie.String()))
	for lecteur.Scan() {
		var ligne ligneJournal
		if err := json.Unmarshal(lecteur.Bytes(), &ligne); err != nil {
			t.Fatalf("ligne de journal non JSON : %v : %s", err, lecteur.Text())
		}
		lignes = append(lignes, ligne)
	}
	return lignes
}

func ligneDuMessage(t *testing.T, sortie *bytes.Buffer, message string) ligneJournal {
	t.Helper()
	var trouvees []ligneJournal
	for _, l := range lignesDe(t, sortie) {
		if l["msg"] == message {
			trouvees = append(trouvees, l)
		}
	}
	if len(trouvees) != 1 {
		t.Fatalf("%d ligne(s) %q, une attendue : %s", len(trouvees), message, sortie)
	}
	return trouvees[0]
}

func texte(l ligneJournal, cle string) string {
	valeur, _ := l[cle].(string)
	return valeur
}

func exigerTraceValide(t *testing.T, l ligneJournal) {
	t.Helper()
	trace, span := texte(l, "trace_id"), texte(l, "span_id")
	if !formatTraceID.MatchString(trace) || trace == strings.Repeat("0", 32) {
		t.Fatalf("trace_id %q invalide : %v", trace, l)
	}
	if !formatSpanID.MatchString(span) || span == strings.Repeat("0", 16) {
		t.Fatalf("span_id %q invalide : %v", span, l)
	}
}

func exigerCorrelationUUID(t *testing.T, l ligneJournal) string {
	t.Helper()
	id := texte(l, "correlation_id")
	if _, err := uuid.Parse(id); err != nil || len(id) != 36 {
		t.Fatalf("correlation_id %q n'est pas un UUID : %v", id, l)
	}
	return id
}

func exigerChampsDeCorrelation(t *testing.T, l ligneJournal, service, correlation, trace string) {
	t.Helper()
	if texte(l, "service") != service {
		t.Errorf("service %q, %q attendu : %v", texte(l, "service"), service, l)
	}
	if texte(l, "correlation_id") != correlation {
		t.Errorf("correlation_id %q, %q attendu : %v", texte(l, "correlation_id"), correlation, l)
	}
	if texte(l, "trace_id") != trace {
		t.Errorf("trace_id %q, %q attendu : %v", texte(l, "trace_id"), trace, l)
	}
	exigerTraceValide(t, l)
}

type appelHTTP struct {
	traceparent string
	correlation string
	reponse     http.Header
	contexte    context.Context
}

func appelerAvec(t *testing.T, logger *slog.Logger, entetes map[string]string) appelHTTP {
	t.Helper()
	var appel appelHTTP
	traiteur := logging.Middleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appel.contexte = r.Context()
		logger.InfoContext(r.Context(), messageDuTraiteur)
		w.WriteHeader(http.StatusNoContent)
	}))
	requete := httptest.NewRequest(http.MethodGet, "/marche/v1/articles", nil)
	for nom, valeur := range entetes {
		requete.Header.Set(nom, valeur)
	}
	enregistreur := httptest.NewRecorder()
	traiteur.ServeHTTP(enregistreur, requete)
	appel.reponse = enregistreur.Result().Header
	appel.traceparent, _ = logging.TraceParent(appel.contexte)
	appel.correlation, _ = logging.CorrelationID(appel.contexte)
	return appel
}

func TestUS014AC1RequeteAvecTraceparentEtCorrelationValidesLesReprend(t *testing.T) {
	logger, sortie := journalJSON(t, serviceDeTest)

	appel := appelerAvec(t, logger, map[string]string{logging.TraceParentHeader: traceparentRecu, logging.CorrelationHeader: correlationRecue})

	ligne := ligneDuMessage(t, sortie, messageDuTraiteur)
	exigerChampsDeCorrelation(t, ligne, serviceDeTest, correlationRecue, traceRecue)
	if texte(ligne, "span_id") == spanRecu {
		t.Errorf("span_id identique au parent reçu %q : un nouveau span est attendu", spanRecu)
	}
	if appel.reponse.Get(logging.CorrelationHeader) != correlationRecue {
		t.Errorf("X-Correlation-Id renvoyé %q, %q attendu", appel.reponse.Get(logging.CorrelationHeader), correlationRecue)
	}
	morceaux := formatTraceparent.FindStringSubmatch(appel.traceparent)
	if morceaux == nil || morceaux[1] != traceRecue || morceaux[2] != texte(ligne, "span_id") || morceaux[3] != "01" {
		t.Errorf("traceparent du contexte %q : trace %s, span %s et drapeaux 01 attendus", appel.traceparent, traceRecue, texte(ligne, "span_id"))
	}
	for _, l := range lignesDe(t, sortie) {
		exigerChampsDeCorrelation(t, l, serviceDeTest, correlationRecue, traceRecue)
	}
}

func TestUS014AC1RequeteSansEnteteTireCorrelationEtTrace(t *testing.T) {
	logger, sortie := journalJSON(t, serviceDeTest)

	appel := appelerAvec(t, logger, nil)

	ligne := ligneDuMessage(t, sortie, messageDuTraiteur)
	correlation := exigerCorrelationUUID(t, ligne)
	exigerTraceValide(t, ligne)
	if appel.reponse.Get(logging.CorrelationHeader) != correlation {
		t.Errorf("X-Correlation-Id renvoyé %q, %q journalisé", appel.reponse.Get(logging.CorrelationHeader), correlation)
	}
	if !strings.Contains(appel.traceparent, texte(ligne, "trace_id")) {
		t.Errorf("traceparent du contexte %q sans la trace journalisée %s", appel.traceparent, texte(ligne, "trace_id"))
	}
	for _, l := range lignesDe(t, sortie) {
		exigerChampsDeCorrelation(t, l, serviceDeTest, correlation, texte(ligne, "trace_id"))
	}
}

func TestUS014AC1DeuxRequetesSansEnteteNePartagentNiCorrelationNiTrace(t *testing.T) {
	logger, _ := journalJSON(t, serviceDeTest)

	premiere := appelerAvec(t, logger, nil)
	seconde := appelerAvec(t, logger, nil)

	if premiere.correlation == seconde.correlation {
		t.Errorf("même correlation_id %q pour deux requêtes", premiere.correlation)
	}
	if premiere.traceparent[3:35] == seconde.traceparent[3:35] {
		t.Errorf("même trace %q pour deux requêtes", premiere.traceparent)
	}
}

func TestUS014AC1TraceparentInvalideRemplaceParUneTraceTiree(t *testing.T) {
	cas := map[string]string{
		"vide":                     "",
		"illisible":                "pas-un-traceparent",
		"version ff interdite":     "ff-" + traceRecue + "-" + spanRecu + "-01",
		"trace nulle":              "00-" + strings.Repeat("0", 32) + "-" + spanRecu + "-01",
		"span nul":                 "00-" + traceRecue + "-" + strings.Repeat("0", 16) + "-01",
		"trace trop courte":        "00-" + traceRecue[:31] + "-" + spanRecu + "-01",
		"span trop long":           "00-" + traceRecue + "-" + spanRecu + "0-01",
		"caractère non hexa":       "00-" + strings.Replace(traceRecue, "4", "g", 1) + "-" + spanRecu + "-01",
		"drapeaux sur un chiffre":  "00-" + traceRecue + "-" + spanRecu + "-1",
		"champ en trop version 00": traceparentRecu + "-suite",
		"séparateurs absents":      "00" + traceRecue + spanRecu + "01",
		"injection de ligne":       "00-" + traceRecue + "-" + spanRecu + "-01\n{\"audit\":true}",
	}
	for nom, recu := range cas {
		t.Run(nom, func(t *testing.T) {
			logger, sortie := journalJSON(t, serviceDeTest)

			appel := appelerAvec(t, logger, map[string]string{logging.TraceParentHeader: recu, logging.CorrelationHeader: correlationRecue})

			ligne := ligneDuMessage(t, sortie, messageDuTraiteur)
			exigerTraceValide(t, ligne)
			if texte(ligne, "trace_id") == traceRecue {
				t.Errorf("trace %s reprise d'un traceparent invalide %q", traceRecue, recu)
			}
			if texte(ligne, "correlation_id") != correlationRecue {
				t.Errorf("correlation_id %q, %q attendu malgré un traceparent invalide", texte(ligne, "correlation_id"), correlationRecue)
			}
			morceaux := formatTraceparent.FindStringSubmatch(appel.traceparent)
			if morceaux == nil || morceaux[1] != texte(ligne, "trace_id") {
				t.Errorf("traceparent du contexte %q sans la trace journalisée %s", appel.traceparent, texte(ligne, "trace_id"))
			}
		})
	}
}

func TestUS014AC1TraceparentHexaMajusculeObservation(t *testing.T) {
	logger, sortie := journalJSON(t, serviceDeTest)

	appelerAvec(t, logger, map[string]string{logging.TraceParentHeader: strings.ToUpper(traceparentRecu)})

	ligne := ligneDuMessage(t, sortie, messageDuTraiteur)
	exigerTraceValide(t, ligne)
	t.Logf("traceparent en majuscules : trace journalisée %s (reçue %s)", texte(ligne, "trace_id"), strings.ToUpper(traceRecue))
}

func TestUS014AC1CorrelationRecueMalFormeeRemplaceeParUnUUIDTire(t *testing.T) {
	cas := map[string]string{
		"texte libre":            "pas-un-uuid",
		"uuid entre accolades":   "{" + correlationRecue + "}",
		"uuid tronqué":           correlationRecue[:35],
		"uuid sans tirets":       strings.ReplaceAll(correlationRecue, "-", ""),
		"urn uuid":               "urn:uuid:" + correlationRecue,
		"uuid suivi d'un saut":   correlationRecue + "\n{\"audit\":true,\"service\":\"controle\"}",
		"injection json":         `","audit":true,"service":"controle`,
		"très long":              strings.Repeat("a", 4096),
		"espaces autour":         " " + correlationRecue + " ",
		"caractères de contrôle": "\x00\x1b[31m" + correlationRecue,
	}
	for nom, recu := range cas {
		t.Run(nom, func(t *testing.T) { exigerCorrelationRemplacee(t, recu) })
	}
}

func exigerCorrelationRemplacee(t *testing.T, recu string) {
	t.Helper()
	logger, sortie := journalJSON(t, serviceDeTest)

	appel := appelerAvec(t, logger, map[string]string{logging.CorrelationHeader: recu, logging.TraceParentHeader: traceparentRecu})

	ligne := ligneDuMessage(t, sortie, messageDuTraiteur)
	correlation := exigerCorrelationUUID(t, ligne)
	if correlation == recu {
		t.Errorf("correlation_id mal formé %q repris", recu)
	}
	if appel.reponse.Get(logging.CorrelationHeader) != correlation {
		t.Errorf("X-Correlation-Id renvoyé %q, %q journalisé", appel.reponse.Get(logging.CorrelationHeader), correlation)
	}
	if texte(ligne, "trace_id") != traceRecue {
		t.Errorf("trace %q, %q attendue malgré une corrélation mal formée", texte(ligne, "trace_id"), traceRecue)
	}
	for _, l := range lignesDe(t, sortie) {
		if l["audit"] == true {
			t.Errorf("ligne d'audit forgée par l'en-tête reçu : %v", l)
		}
	}
}

func TestUS014AC1PropagationSortanteRepriseParLeServiceAppele(t *testing.T) {
	loggerAppele, sortieAppele := journalJSON(t, "service-appele")
	appele := httptest.NewServer(logging.Middleware(loggerAppele, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loggerAppele.InfoContext(r.Context(), messageDuTraiteur)
		w.WriteHeader(http.StatusNoContent)
	})))
	t.Cleanup(appele.Close)
	loggerAppelant, sortieAppelant := journalJSON(t, "service-appelant")
	appelant := logging.Middleware(loggerAppelant, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loggerAppelant.InfoContext(r.Context(), messageDuTraiteur)
		sortante, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, appele.URL, nil)
		logging.Propagate(r.Context(), sortante.Header)
		reponse, err := appele.Client().Do(sortante)
		if err != nil {
			t.Errorf("appel sortant : %v", err)
			return
		}
		_ = reponse.Body.Close()
		w.WriteHeader(http.StatusNoContent)
	}))
	requete := httptest.NewRequest(http.MethodGet, "/", nil)
	requete.Header.Set(logging.TraceParentHeader, traceparentRecu)
	requete.Header.Set(logging.CorrelationHeader, correlationRecue)

	appelant.ServeHTTP(httptest.NewRecorder(), requete)

	cote1 := ligneDuMessage(t, sortieAppelant, messageDuTraiteur)
	cote2 := ligneDuMessage(t, sortieAppele, messageDuTraiteur)
	exigerChampsDeCorrelation(t, cote1, "service-appelant", correlationRecue, traceRecue)
	exigerChampsDeCorrelation(t, cote2, "service-appele", correlationRecue, traceRecue)
	if texte(cote1, "span_id") == texte(cote2, "span_id") {
		t.Errorf("même span_id %s des deux côtés d'un appel", texte(cote1, "span_id"))
	}
}

func TestUS014AC1PropagateSansContexteNePoseRienDeFaux(t *testing.T) {
	entetes := http.Header{}

	logging.Propagate(context.Background(), entetes)

	if valeur := entetes.Get(logging.TraceParentHeader); valeur != "" && !formatTraceparent.MatchString(valeur) {
		t.Errorf("traceparent propagé mal formé : %q", valeur)
	}
	if valeur := entetes.Get(logging.CorrelationHeader); valeur != "" {
		if _, err := uuid.Parse(valeur); err != nil {
			t.Errorf("X-Correlation-Id propagé mal formé : %q", valeur)
		}
	}
}

func TestUS014AC1MessageConsommeContinueLaTraceRecue(t *testing.T) {
	logger, sortie := journalJSON(t, "consommateur")
	ctx := logging.ContinueTrace(logging.WithReceivedCorrelationID(context.Background(), correlationRecue), traceparentRecu)

	logger.InfoContext(ctx, messageDuTraiteur)

	ligne := ligneDuMessage(t, sortie, messageDuTraiteur)
	exigerChampsDeCorrelation(t, ligne, "consommateur", correlationRecue, traceRecue)
	if texte(ligne, "span_id") == spanRecu {
		t.Errorf("span_id identique au parent reçu")
	}
	propage, _ := logging.TraceParent(ctx)
	morceaux := formatTraceparent.FindStringSubmatch(propage)
	if morceaux == nil || morceaux[1] != traceRecue || morceaux[2] != texte(ligne, "span_id") || morceaux[3] != "01" {
		t.Errorf("traceparent à propager %q : trace %s, span journalisé, drapeaux 01 attendus", propage, traceRecue)
	}
}

func TestUS014AC1MessageConsommeSansTraceValideTireUneTrace(t *testing.T) {
	for nom, recu := range map[string]string{"absent": "", "nul": "00-" + strings.Repeat("0", 32) + "-" + strings.Repeat("0", 16) + "-00", "illisible": "zzz"} {
		t.Run(nom, func(t *testing.T) {
			logger, sortie := journalJSON(t, "consommateur")
			ctx := logging.ContinueTrace(logging.WithReceivedCorrelationID(context.Background(), correlationRecue), recu)

			logger.InfoContext(ctx, messageDuTraiteur)

			ligne := ligneDuMessage(t, sortie, messageDuTraiteur)
			exigerTraceValide(t, ligne)
			if texte(ligne, "correlation_id") != correlationRecue {
				t.Errorf("correlation_id %q, %q attendu", texte(ligne, "correlation_id"), correlationRecue)
			}
		})
	}
}

func TestUS014AC1CorrelationDeMessageMalFormeeRemplacee(t *testing.T) {
	for nom, recu := range map[string]string{"vide": "", "texte": "pas-un-uuid", "injection": correlationRecue + "\n{\"audit\":true}"} {
		t.Run(nom, func(t *testing.T) {
			logger, sortie := journalJSON(t, "consommateur")
			ctx := logging.ContinueTrace(logging.WithReceivedCorrelationID(context.Background(), recu), traceparentRecu)

			logger.InfoContext(ctx, messageDuTraiteur)

			ligne := ligneDuMessage(t, sortie, messageDuTraiteur)
			if exigerCorrelationUUID(t, ligne) == recu {
				t.Errorf("correlation_id mal formé %q repris", recu)
			}
		})
	}
}

func TestUS014AC1CommandePonctuelleTireCorrelationEtTrace(t *testing.T) {
	logger, sortie := journalJSON(t, "commande")
	premiere := logging.StartOperation(context.Background())
	seconde := logging.StartOperation(context.Background())

	logger.InfoContext(premiere, "premiere")
	logger.InfoContext(seconde, "seconde")

	l1, l2 := ligneDuMessage(t, sortie, "premiere"), ligneDuMessage(t, sortie, "seconde")
	exigerCorrelationUUID(t, l1)
	exigerCorrelationUUID(t, l2)
	exigerTraceValide(t, l1)
	exigerTraceValide(t, l2)
	if texte(l1, "correlation_id") == texte(l2, "correlation_id") || texte(l1, "trace_id") == texte(l2, "trace_id") {
		t.Errorf("deux opérations partagent leurs identifiants : %v / %v", l1, l2)
	}
}

func TestUS014AC1CorrelationIDOrNewSansContexteDonneUnUUID(t *testing.T) {
	id := logging.CorrelationIDOrNew(context.Background())

	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("CorrelationIDOrNew %q : %v", id, err)
	}
	if repris := logging.CorrelationIDOrNew(logging.WithCorrelationID(context.Background(), correlationRecue)); repris != correlationRecue {
		t.Fatalf("CorrelationIDOrNew %q, %q attendu", repris, correlationRecue)
	}
}

func TestUS014AC1ChaqueLigneDeRequetePorteLeService(t *testing.T) {
	logger, sortie := journalJSON(t, serviceDeTest)

	appelerAvec(t, logger, map[string]string{logging.TraceParentHeader: traceparentRecu})
	logger.Info("hors requête")

	for _, l := range lignesDe(t, sortie) {
		if texte(l, "service") != serviceDeTest {
			t.Errorf("ligne sans service %q : %v", serviceDeTest, l)
		}
	}
}

func TestUS014AC4AuditHorodatePorteArticleRevisionIssueEtCorrelation(t *testing.T) {
	logger, sortie := journalJSON(t, "place-de-marche")
	ctx := logging.ContinueTrace(logging.WithReceivedCorrelationID(context.Background(), correlationRecue), traceparentRecu)
	avant := time.Now().Add(-time.Second)

	logging.Audit(ctx, logger, "résultat de contrôle appliqué", "controle.resultat_applique",
		slog.String("article_id", "0b7e9c1a-1111-4222-8333-944455556666"), slog.Int("revision", 2), slog.String("issue", "refuse"), slog.String("motif", "coordonnees_detectees"))
	logger.InfoContext(ctx, "journal ordinaire")

	audit := ligneDuMessage(t, sortie, "résultat de contrôle appliqué")
	if audit["audit"] != true || texte(audit, "event") != "controle.resultat_applique" {
		t.Errorf("événement d'audit sans audit=true ou event : %v", audit)
	}
	horodatage, err := time.Parse(time.RFC3339Nano, texte(audit, "time"))
	if err != nil || horodatage.Before(avant) || horodatage.After(time.Now().Add(time.Second)) {
		t.Errorf("horodatage %q invalide : %v", texte(audit, "time"), err)
	}
	if texte(audit, "article_id") != "0b7e9c1a-1111-4222-8333-944455556666" || audit["revision"] != float64(2) || texte(audit, "issue") != "refuse" || texte(audit, "motif") != "coordonnees_detectees" {
		t.Errorf("audit sans article, révision, issue ou motif : %v", audit)
	}
	exigerChampsDeCorrelation(t, audit, "place-de-marche", correlationRecue, traceRecue)
	if ordinaire := ligneDuMessage(t, sortie, "journal ordinaire"); ordinaire["audit"] != nil {
		t.Errorf("journal ordinaire marqué audit : %v", ordinaire)
	}
}

func TestUS014AC4AuditHorsDeToutContexteResteHorodateEtMarque(t *testing.T) {
	logger, sortie := journalJSON(t, "place-de-marche")

	logging.Audit(context.Background(), logger, "décision", "annonces.decision_admin", slog.String("article_id", "a"), slog.Int("revision", 1), slog.String("issue", "publie"))

	audit := ligneDuMessage(t, sortie, "décision")
	if audit["audit"] != true || texte(audit, "event") != "annonces.decision_admin" || texte(audit, "time") == "" || texte(audit, "service") != "place-de-marche" {
		t.Errorf("audit incomplet : %v", audit)
	}
}

func TestUS014AC6AuditNePorteAucunChampDOrigineDeConfiance(t *testing.T) {
	logger, sortie := journalJSON(t, "controle")
	ctx := logging.StartOperation(context.Background())

	logging.Audit(ctx, logger, "décision", "controle.resultat_applique", slog.String("article_id", "a"), slog.Int("revision", 1), slog.String("issue", "publie"))

	audit := ligneDuMessage(t, sortie, "décision")
	for _, cle := range []string{"kubernetes", "namespace", "pod", "pod_name", "container", "container_name", "origine", "origin", "source"} {
		if _, present := audit[cle]; present {
			t.Errorf("champ d'origine %q écrit par le service lui-même : %v", cle, audit)
		}
	}
}
