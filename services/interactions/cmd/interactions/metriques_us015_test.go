package main

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const (
	motifEmetteurInattendu     = "emetteur_inattendu"
	motifRefuseParLeTraitement = "refuse_par_le_traitement"
)

type bancMesureInteractions struct {
	*bancInteractions
	vendeur string
	traces  []string
}

func TestUS015MetriquesDInteractions(t *testing.T) {
	vendeur := uuid.NewString()
	b := &bancMesureInteractions{bancInteractions: demarrerBancInteractionsEnProcessus(t), vendeur: vendeur, traces: []string{vendeur}}
	t.Run("AC1 requêtes comptées sous leur route avec leur code", b.requetesComptees)
	t.Run("AC1 décision consommée comptée sur sa file", b.decisionConsommeeComptee)
	t.Run("AC1 décision d'un émetteur inattendu comptée en écartée", b.decisionEmetteurInattenduEcartee)
	t.Run("AC1 décision non conforme comptée en écartée", b.decisionNonConformeEcartee)
	t.Run("AC1 aucun identifiant ni donnée personnelle en étiquette", b.aucunIdentifiantEnEtiquette)
	t.Run("AC5 métriques absentes des ports public et de santé", b.metriquesAbsentesDesAutresPorts)
	t.Run("AC5 port de métriques limité à GET /metrics", b.portDeMetriquesLimite)
}

func demarrerBancInteractionsEnProcessus(t *testing.T) *bancInteractions {
	t.Helper()
	bases := demarrerPostgreSQL(t)
	canal := demarrerRabbitMQ(t)
	migrer(t, bases)
	return &bancInteractions{serviceInteractions: lancerLeProcessusInteractions(t, construireInteractions(t), bases, canal), bases: bases, canal: canal}
}

func construireInteractions(t *testing.T) string {
	t.Helper()
	repertoire := t.TempDir()
	binaire := filepath.Join(repertoire, serviceName)
	if runtime.GOOS == "windows" {
		binaire += ".exe"
	}
	construction := exec.Command("go", "install", ".")
	construction.Env = append(os.Environ(), "GOBIN="+repertoire)
	if sortie, err := construction.CombinedOutput(); err != nil {
		t.Fatalf("construction d'Interactions : %v\n%s", err, sortie)
	}
	return binaire
}

func lancerLeProcessusInteractions(t *testing.T, binaire string, bases basesPostgreSQL, canal *canalDeTest) *serviceInteractions {
	t.Helper()
	jetons := nouvelleFabriqueDeJetons(t)
	api, gestion, metriques := adresseLibre(t), adresseLibre(t), adresseLibre(t)
	variables := []string{
		"BASE_DE_DONNEES_URL=" + bases.adresse(bases.interactions.execution, baseInteractions),
		"BASE_DE_DONNEES_TLS_EXIGE=false",
		"OIDC_EMETTEUR=" + emetteurAttendu,
		"OIDC_JEU_DE_CLES_URL=" + jetons.jwks.URL,
		"CANAL_HOTE=" + canal.hote,
		"CANAL_PORT=" + canal.port,
		"CANAL_VHOST=" + vhostDuCanal,
		"CANAL_UTILISATEUR=" + identiteInteraction,
		"CANAL_MOT_DE_PASSE=" + canal.secrets[identiteInteraction],
		"ADRESSE_API=" + api,
		"ADRESSE_GESTION=" + gestion,
		"ADRESSE_METRIQUES=" + metriques,
	}
	journal := &journalPartage{}
	processus := exec.Command(binaire, "serve")
	processus.Env = append(os.Environ(), variables...)
	processus.Stdout, processus.Stderr = journal, journal
	if err := processus.Start(); err != nil {
		t.Fatalf("lancement d'Interactions : %v", err)
	}
	termine := make(chan error, 1)
	go func() { termine <- processus.Wait() }()
	t.Cleanup(func() {
		_ = processus.Process.Kill()
		select {
		case erreur := <-termine:
			t.Logf("arrêt d'Interactions : %v", erreur)
		case <-time.After(10 * time.Second):
			t.Error("Interactions ne s'est pas arrêté en 10 s")
		}
	})
	client := &http.Client{Timeout: 5 * time.Second}
	attendre(t, time.Minute, "démarrage d'Interactions", func() bool {
		select {
		case err := <-termine:
			t.Fatalf("Interactions s'est arrêté : %v\n%s", err, journal.tampon.String())
		default:
		}
		reponse, err := client.Get("http://" + gestion + "/disponibilite")
		if err != nil {
			return false
		}
		_ = reponse.Body.Close()
		return reponse.StatusCode == http.StatusOK
	})
	return &serviceInteractions{adresse: "http://" + api, gestion: "http://" + gestion, metriques: "http://" + metriques, client: client, journal: journal, jetons: jetons}
}

func (b *bancMesureInteractions) adresseDesMetriques() string {
	return strings.TrimPrefix(b.metriques, "http://")
}

func (b *bancMesureInteractions) releve(t *testing.T) releveDeMetriques {
	t.Helper()
	return releverLesMetriques(t, b.adresseDesMetriques())
}

func (b *bancMesureInteractions) attendreLaMesure(t *testing.T, quoi string, ecartes map[string]string, nom string, attendu float64) {
	t.Helper()
	attendreLaMesure(t, b.adresseDesMetriques(), quoi, func(r releveDeMetriques) bool {
		return r.total(nom, ecartes) == attendu
	})
}

func exigerIncrementDuCode(t *testing.T, avant, apres releveDeMetriques, code string) {
	t.Helper()
	avantCode, _ := avant.requetesSousLaRoute(http.MethodGet, cheminNotifications, code)
	apresCode, routes := apres.requetesSousLaRoute(http.MethodGet, cheminNotifications, code)
	if apresCode-avantCode != 1 {
		t.Fatalf("GET %s en %s : +%v, +1 attendu\n%s", cheminNotifications, code, apresCode-avantCode, apres.brut)
	}
	for _, route := range routes {
		if strings.ContainsAny(route, "?=") {
			t.Fatalf("http_route=%q contient la requête brute", route)
		}
	}
}

func (b *bancMesureInteractions) requetesComptees(t *testing.T) {
	avant := b.releve(t)
	if statut := b.lister(t, b.jetons.vendeur(b.vendeur), "?taille=50").statut; statut != http.StatusOK {
		t.Fatalf("liste du vendeur : %d", statut)
	}
	if statut := b.lister(t, "", "").statut; statut != http.StatusUnauthorized {
		t.Fatalf("liste sans jeton : %d", statut)
	}
	apres := b.releve(t)
	exigerIncrementDuCode(t, avant, apres, "200")
	exigerIncrementDuCode(t, avant, apres, "401")
}

func (b *bancMesureInteractions) decisionConsommeeComptee(t *testing.T) {
	file := map[string]string{"file": fileDecisions}
	avant := b.releve(t).total(messagesConsommes, file)
	d := decision(t, b.vendeur, "refuse", motif("annonce_incomplete"))
	b.traces = append(b.traces, d.id, d.article)
	b.diffuser(t, d.corps)
	b.attendreLaMesure(t, "consommation de la décision", file, messagesConsommes, avant+1)
}

func (b *bancMesureInteractions) decisionEmetteurInattenduEcartee(t *testing.T) {
	ecartes := map[string]string{"file": fileDecisions, "motif": motifEmetteurInattendu}
	avant := b.releve(t).total(messagesEcartes, ecartes)
	d := decision(t, b.vendeur, "publie", nil)
	p := versLesEvenements(d.corps)
	p.emetteur = identiteAdmin
	b.traces = append(b.traces, d.id, d.article, p.id)
	if err := b.canal.publier(t, identiteAdmin, p); err != nil {
		t.Fatalf("publication par %s : %v", identiteAdmin, err)
	}
	b.attendreLaMesure(t, "mise à l'écart de l'émetteur inattendu", ecartes, messagesEcartes, avant+1)
}

func (b *bancMesureInteractions) decisionNonConformeEcartee(t *testing.T) {
	ecartes := map[string]string{"file": fileDecisions, "motif": motifRefuseParLeTraitement}
	avant := b.releve(t).total(messagesEcartes, ecartes)
	b.diffuser(t, []byte(`{"id":"tronqué"`))
	b.attendreLaMesure(t, "mise à l'écart de la décision non conforme", ecartes, messagesEcartes, avant+1)
}

func (b *bancMesureInteractions) aucunIdentifiantEnEtiquette(t *testing.T) {
	exigerAucuneDonneeIdentifiante(t, b.releve(t), b.traces...)
}

func (b *bancMesureInteractions) metriquesAbsentesDesAutresPorts(t *testing.T) {
	exigerMetriquesAbsentes(t, b.adresse)
	exigerMetriquesAbsentes(t, b.gestion)
}

func (b *bancMesureInteractions) portDeMetriquesLimite(t *testing.T) {
	exigerSeulementMetricsSurLePortDeMesure(t, b.adresseDesMetriques(), "/", "/sante", "/disponibilite", cheminNotifications)
}
