package main

import (
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	histogrammeDesRequetes = "http_server_request_duration_seconds"
	comptageDesRequetes    = histogrammeDesRequetes + "_count"
	messagesConsommes      = "canal_messages_consommes_total"
	messagesEcartes        = "canal_messages_ecartes_total"
	delaiDeMesure          = 20 * time.Second
)

var (
	ligneExposee         = regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*)(?:\{(.*)\})?\s+(\S+)`)
	etiquetteExposee     = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)="((?:[^"\\]|\\.)*)"`)
	identifiantUniversel = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
)

type serieExposee struct {
	nom        string
	etiquettes map[string]string
	valeur     float64
}

type releveDeMetriques struct {
	brut   string
	series []serieExposee
}

func lireEnGET(t *testing.T, adresse string) (int, string) {
	t.Helper()
	requete, err := http.NewRequestWithContext(t.Context(), http.MethodGet, adresse, nil)
	if err != nil {
		t.Fatalf("requête %s : %v", adresse, err)
	}
	reponse, err := (&http.Client{Timeout: 5 * time.Second}).Do(requete)
	if err != nil {
		t.Fatalf("GET %s : %v", adresse, err)
	}
	defer func() { _ = reponse.Body.Close() }()
	corps, err := io.ReadAll(reponse.Body)
	if err != nil {
		t.Fatalf("lecture de %s : %v", adresse, err)
	}
	return reponse.StatusCode, string(corps)
}

func releverLesMetriques(t *testing.T, adresse string) releveDeMetriques {
	t.Helper()
	statut, corps := lireEnGET(t, "http://"+adresse+"/metrics")
	if statut != http.StatusOK {
		t.Fatalf("GET /metrics sur %s : statut %d", adresse, statut)
	}
	return analyserLExposition(t, corps)
}

func analyserLExposition(t *testing.T, brut string) releveDeMetriques {
	t.Helper()
	r := releveDeMetriques{brut: brut}
	for _, ligne := range strings.Split(brut, "\n") {
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		morceaux := ligneExposee.FindStringSubmatch(ligne)
		if morceaux == nil {
			t.Fatalf("ligne d'exposition illisible : %q", ligne)
		}
		valeur, err := strconv.ParseFloat(morceaux[3], 64)
		if err != nil {
			t.Fatalf("valeur illisible : %q", ligne)
		}
		s := serieExposee{nom: morceaux[1], etiquettes: map[string]string{}, valeur: valeur}
		for _, e := range etiquetteExposee.FindAllStringSubmatch(morceaux[2], -1) {
			s.etiquettes[e[1]] = e[2]
		}
		r.series = append(r.series, s)
	}
	return r
}

func (r releveDeMetriques) retenir(nom string, attendues map[string]string) []serieExposee {
	var retenues []serieExposee
	for _, s := range r.series {
		if s.nom == nom && porteLesEtiquettes(s, attendues) {
			retenues = append(retenues, s)
		}
	}
	return retenues
}

func porteLesEtiquettes(s serieExposee, attendues map[string]string) bool {
	for cle, valeur := range attendues {
		if s.etiquettes[cle] != valeur {
			return false
		}
	}
	return true
}

func (r releveDeMetriques) total(nom string, attendues map[string]string) float64 {
	somme := 0.0
	for _, s := range r.retenir(nom, attendues) {
		somme += s.valeur
	}
	return somme
}

func (r releveDeMetriques) requetesSousLaRoute(methode, prefixe, code string) (float64, []string) {
	somme := 0.0
	var routes []string
	for _, s := range r.retenir(comptageDesRequetes, map[string]string{"http_request_method": methode, "http_response_status_code": code}) {
		if strings.HasPrefix(s.etiquettes["http_route"], prefixe) {
			somme += s.valeur
			routes = append(routes, s.etiquettes["http_route"])
		}
	}
	return somme, routes
}

func exigerAucuneDonneeIdentifiante(t *testing.T, r releveDeMetriques, valeurs ...string) {
	t.Helper()
	for _, valeur := range valeurs {
		if strings.Contains(r.brut, valeur) {
			t.Fatalf("%q exposé dans les métriques", valeur)
		}
	}
	for _, s := range r.series {
		for cle, valeur := range s.etiquettes {
			if identifiantUniversel.MatchString(valeur) {
				t.Fatalf("identifiant en étiquette : %s{%s=%q}", s.nom, cle, valeur)
			}
		}
	}
}

func attendreLaMesure(t *testing.T, adresse, quoi string, condition func(releveDeMetriques) bool) releveDeMetriques {
	t.Helper()
	var dernier releveDeMetriques
	echeance := time.Now().Add(delaiDeMesure)
	for time.Now().Before(echeance) {
		dernier = releverLesMetriques(t, adresse)
		if condition(dernier) {
			return dernier
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("%s : non mesuré en %s\n%s", quoi, delaiDeMesure, dernier.brut)
	return dernier
}

func exigerMetriquesAbsentes(t *testing.T, adresse string) {
	t.Helper()
	statut, corps := lireEnGET(t, adresse+"/metrics")
	if statut == http.StatusOK || strings.Contains(corps, histogrammeDesRequetes) || strings.Contains(corps, "target_info") {
		t.Fatalf("GET %s/metrics : statut %d, métriques servies", adresse, statut)
	}
}

func exigerSeulementMetricsSurLePortDeMesure(t *testing.T, adresse string, chemins ...string) {
	t.Helper()
	for _, chemin := range chemins {
		if statut, _ := lireEnGET(t, "http://"+adresse+chemin); statut != http.StatusNotFound {
			t.Errorf("GET %s sur le port de métriques : statut %d, 404 attendu", chemin, statut)
		}
	}
}
