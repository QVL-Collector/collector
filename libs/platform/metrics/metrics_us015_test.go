package metrics_test

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	"collector/libs/platform/httpserver"
	"collector/libs/platform/logging"
	"collector/libs/platform/metrics"
)

const (
	nomDuService      = "service-de-test"
	histogrammeHTTP   = "http_server_request_duration_seconds"
	routeArticle      = "/essai/articles/{id}"
	routeMethodes     = "/essai/methodes"
	routePanique      = "/essai/panique"
	routeIndisponible = "/essai/indisponible/{id}"
	methodeInconnue   = "BREW"
)

var exposition http.Handler

type serie struct {
	nom        string
	etiquettes map[string]string
	valeur     float64
}

type releve struct {
	brut   string
	series []serie
}

var (
	ligneDeSerie = regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*)(?:\{(.*)\})?\s+(\S+)`)
	etiquette    = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)="((?:[^"\\]|\\.)*)"`)
)

func TestMain(m *testing.M) {
	gestionnaire, err := metrics.Install(nomDuService, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "installation des métriques :", err)
		os.Exit(1)
	}
	exposition = gestionnaire
	os.Exit(m.Run())
}

func journalMuet() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func repondre(statut int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(statut) }
}

func demarrerAPIInstrumentee(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+routeArticle, repondre(http.StatusOK))
	mux.HandleFunc(routeMethodes, repondre(http.StatusOK))
	mux.HandleFunc("GET "+routePanique, func(http.ResponseWriter, *http.Request) { panic("panique provoquée par le test") })
	mux.HandleFunc("GET "+routeIndisponible, repondre(http.StatusServiceUnavailable))
	journal := journalMuet()
	serveur := httptest.NewServer(logging.Middleware(journal, metrics.Middleware(httpserver.Recover(journal, mux))))
	t.Cleanup(serveur.Close)
	return serveur
}

func appeler(t *testing.T, serveur *httptest.Server, methode, chemin string) int {
	t.Helper()
	requete, err := http.NewRequestWithContext(t.Context(), methode, serveur.URL+chemin, nil)
	if err != nil {
		t.Fatalf("requête %s %s : %v", methode, chemin, err)
	}
	reponse, err := serveur.Client().Do(requete)
	if err != nil {
		t.Fatalf("appel %s %s : %v", methode, chemin, err)
	}
	_, _ = io.Copy(io.Discard, reponse.Body)
	_ = reponse.Body.Close()
	return reponse.StatusCode
}

func interroger(t *testing.T, methode, chemin string) *httptest.ResponseRecorder {
	t.Helper()
	enregistreur := httptest.NewRecorder()
	exposition.ServeHTTP(enregistreur, httptest.NewRequestWithContext(t.Context(), methode, chemin, nil))
	return enregistreur
}

func relever(t *testing.T) releve {
	t.Helper()
	reponse := interroger(t, http.MethodGet, "/metrics")
	if reponse.Code != http.StatusOK {
		t.Fatalf("GET /metrics : statut %d", reponse.Code)
	}
	return analyser(t, reponse.Body.String())
}

func analyser(t *testing.T, brut string) releve {
	t.Helper()
	r := releve{brut: brut}
	for _, ligne := range strings.Split(brut, "\n") {
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		morceaux := ligneDeSerie.FindStringSubmatch(ligne)
		if morceaux == nil {
			t.Fatalf("ligne d'exposition illisible : %q", ligne)
		}
		valeur, err := strconv.ParseFloat(morceaux[3], 64)
		if err != nil {
			t.Fatalf("valeur illisible : %q", ligne)
		}
		s := serie{nom: morceaux[1], etiquettes: map[string]string{}, valeur: valeur}
		for _, e := range etiquette.FindAllStringSubmatch(morceaux[2], -1) {
			s.etiquettes[e[1]] = e[2]
		}
		r.series = append(r.series, s)
	}
	return r
}

func (r releve) filtrer(nom string, attendues map[string]string) []serie {
	var retenues []serie
	for _, s := range r.series {
		if s.nom == nom && correspond(s, attendues) {
			retenues = append(retenues, s)
		}
	}
	return retenues
}

func correspond(s serie, attendues map[string]string) bool {
	for cle, valeur := range attendues {
		if s.etiquettes[cle] != valeur {
			return false
		}
	}
	return true
}

func (r releve) somme(nom string, attendues map[string]string) float64 {
	total := 0.0
	for _, s := range r.filtrer(nom, attendues) {
		total += s.valeur
	}
	return total
}

func requetes(methode, route, code string) map[string]string {
	return map[string]string{"http_request_method": methode, "http_route": route, "http_response_status_code": code}
}

func exigerIncrement(t *testing.T, avant, apres releve, attendues map[string]string, increment float64) {
	t.Helper()
	ecart := apres.somme(histogrammeHTTP+"_count", attendues) - avant.somme(histogrammeHTTP+"_count", attendues)
	if ecart != increment {
		t.Fatalf("%s_count%v : +%v, +%v attendu\n%s", histogrammeHTTP, attendues, ecart, increment, apres.brut)
	}
}

func exigerAbsent(t *testing.T, r releve, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if strings.Contains(r.brut, fragment) {
			t.Fatalf("%q exposé dans les métriques", fragment)
		}
	}
}

func TestUS015AC1RequeteCompteeSousLaRouteParametreeEtNonLeCheminBrut(t *testing.T) {
	api := demarrerAPIInstrumentee(t)
	avant := relever(t)
	identifiants := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	for _, id := range identifiants {
		if statut := appeler(t, api, http.MethodGet, "/essai/articles/"+id); statut != http.StatusOK {
			t.Fatalf("GET article : %d", statut)
		}
	}
	apres := relever(t)
	exigerIncrement(t, avant, apres, requetes(http.MethodGet, routeArticle, "200"), 3)
	exigerAbsent(t, apres, identifiants...)
}

func TestUS015AC1CardinaliteBorneeQuelQueSoitLeNombreDIdentifiants(t *testing.T) {
	api := demarrerAPIInstrumentee(t)
	for range 50 {
		appeler(t, api, http.MethodGet, "/essai/articles/"+uuid.NewString())
	}
	series := relever(t).filtrer(histogrammeHTTP+"_count", map[string]string{"http_request_method": http.MethodGet, "http_response_status_code": "200"})
	routes := map[string]bool{}
	for _, s := range series {
		routes[s.etiquettes["http_route"]] = true
	}
	for route := range routes {
		if strings.HasPrefix(route, "/essai/articles/") && route != routeArticle {
			t.Fatalf("série par identifiant : http_route=%q", route)
		}
	}
	if !routes[routeArticle] {
		t.Fatalf("aucune série pour %s parmi %v", routeArticle, routes)
	}
}

func TestUS015AC1MethodeHorsListeHTTPRegroupeeSousOther(t *testing.T) {
	api := demarrerAPIInstrumentee(t)
	avant := relever(t)
	appeler(t, api, methodeInconnue, routeMethodes)
	apres := relever(t)
	exigerIncrement(t, avant, apres, requetes("_OTHER", routeMethodes, "200"), 1)
	if len(apres.filtrer(histogrammeHTTP+"_count", map[string]string{"http_request_method": methodeInconnue})) != 0 {
		t.Fatalf("méthode %s exposée telle quelle", methodeInconnue)
	}
}

func TestUS015AC1MethodeConnueConserveeTelleQuelle(t *testing.T) {
	api := demarrerAPIInstrumentee(t)
	avant := relever(t)
	appeler(t, api, http.MethodDelete, routeMethodes)
	exigerIncrement(t, avant, relever(t), requetes(http.MethodDelete, routeMethodes, "200"), 1)
}

func TestUS015AC1PaniqueInterceptéeCompteeEnErreur500(t *testing.T) {
	api := demarrerAPIInstrumentee(t)
	avant := relever(t)
	if statut := appeler(t, api, http.MethodGet, routePanique); statut != http.StatusInternalServerError {
		t.Fatalf("panique : statut %d, 500 attendu", statut)
	}
	exigerIncrement(t, avant, relever(t), requetes(http.MethodGet, routePanique, "500"), 1)
}

func TestUS015AC1ErreurServeurCompteeParRouteEtCode(t *testing.T) {
	api := demarrerAPIInstrumentee(t)
	avant := relever(t)
	id := uuid.NewString()
	appeler(t, api, http.MethodGet, "/essai/indisponible/"+id)
	appeler(t, api, http.MethodGet, "/essai/indisponible/"+uuid.NewString())
	apres := relever(t)
	exigerIncrement(t, avant, apres, requetes(http.MethodGet, routeIndisponible, "503"), 2)
	exigerAbsent(t, apres, id)
}

func TestUS015AC1CheminSansRouteNExposePasLeCheminBrut(t *testing.T) {
	api := demarrerAPIInstrumentee(t)
	secret := "client-" + uuid.NewString()
	if statut := appeler(t, api, http.MethodGet, "/inexistant/"+secret); statut != http.StatusNotFound {
		t.Fatalf("chemin inconnu : statut %d, 404 attendu", statut)
	}
	apres := relever(t)
	exigerAbsent(t, apres, secret, "/inexistant/")
	if apres.somme(histogrammeHTTP+"_count", map[string]string{"http_response_status_code": "404"}) < 1 {
		t.Fatalf("404 sur chemin inconnu non compté\n%s", apres.brut)
	}
}

func TestUS015AC1LatenceEnHistogrammePermettantLeP95AutourDe500ms(t *testing.T) {
	api := demarrerAPIInstrumentee(t)
	appeler(t, api, http.MethodGet, "/essai/articles/"+uuid.NewString())
	r := relever(t)
	attendues := requetes(http.MethodGet, routeArticle, "200")
	limites := map[string]bool{}
	for _, s := range r.filtrer(histogrammeHTTP+"_bucket", attendues) {
		limites[s.etiquettes["le"]] = true
	}
	for _, limite := range []string{"0.5", "+Inf"} {
		if !limites[limite] {
			t.Fatalf("limite le=%q absente de l'histogramme : %v", limite, limites)
		}
	}
	if len(limites) < 5 {
		t.Fatalf("histogramme trop grossier pour un p95 : %v", limites)
	}
	if len(r.filtrer(histogrammeHTTP+"_sum", attendues)) != 1 {
		t.Fatalf("%s_sum absent", histogrammeHTTP)
	}
}

func TestUS015AC1EtiquettesHTTPLimiteesAMethodeRouteEtCode(t *testing.T) {
	api := demarrerAPIInstrumentee(t)
	appeler(t, api, http.MethodGet, "/essai/articles/"+uuid.NewString())
	autorisees := map[string]bool{"http_request_method": true, "http_route": true, "http_response_status_code": true, "otel_scope_name": true, "otel_scope_version": true, "otel_scope_schema_url": true, "le": true}
	for _, s := range relever(t).filtrer(histogrammeHTTP+"_bucket", nil) {
		for cle := range s.etiquettes {
			if !autorisees[cle] {
				t.Fatalf("étiquette inattendue %q=%q", cle, s.etiquettes[cle])
			}
		}
	}
}

func TestUS015AC1CompteurCreeSurLeFournisseurGlobalExpose(t *testing.T) {
	compteur := metrics.Counter(otel.Meter("essai.us015"), "essai_us015_evenements", "{evenement}", "Compteur d'essai")
	avant := relever(t).somme("essai_us015_evenements_total", nil)
	compteur.Add(t.Context(), 2)
	if ecart := relever(t).somme("essai_us015_evenements_total", nil) - avant; ecart != 2 {
		t.Fatalf("compteur : +%v, +2 attendu", ecart)
	}
}

func TestUS015AC1IdentiteDuServiceExposee(t *testing.T) {
	if len(relever(t).filtrer("target_info", map[string]string{"service_name": nomDuService})) != 1 {
		t.Fatal("target_info sans service_name")
	}
}

func TestUS015AC5ExpositionRepond404HorsDeGETMetrics(t *testing.T) {
	for _, chemin := range []string{"/", "/sante", "/disponibilite", "/metrics/", "/metrics/x", "/metricsx", "/essai/articles/1"} {
		if code := interroger(t, http.MethodGet, chemin).Code; code != http.StatusNotFound {
			t.Errorf("GET %s : statut %d, 404 attendu", chemin, code)
		}
	}
}

func TestUS015AC5ExpositionRefuseLesMethodesAutresQueGET(t *testing.T) {
	for _, methode := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		reponse := interroger(t, methode, "/metrics")
		if reponse.Code == http.StatusOK || strings.Contains(reponse.Body.String(), histogrammeHTTP) {
			t.Errorf("%s /metrics : statut %d, métriques servies", methode, reponse.Code)
		}
	}
}
