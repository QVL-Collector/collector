package main

import (
	"net/http"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/services/place-de-marche/internal/migrations"
)

const (
	motifEmetteurInattendu     = "emetteur_inattendu"
	motifRefuseParLeTraitement = "refuse_par_le_traitement"
	observationDeLAge          = 3 * time.Second
)

type adressesDuService struct {
	api       string
	gestion   string
	interne   string
	metriques string
}

type bancMesure struct {
	*environnement
	canal    *canalDeTest
	adresses adressesDuService
	sujet    string
	jeton    string
	soumis   articleRecu
	traces   []string
}

func demarrerBancMesure(t *testing.T) *bancMesure {
	t.Helper()
	ctx := contexteBorne(t, 3*time.Minute)
	base := amorcer(ctx, t, demarrerPostgreSQL(t))
	if err := migrations.Apply(ctx, journalMuet(), base.migration); err != nil {
		t.Fatalf("migrations : %v", err)
	}
	canal := demarrerRabbitMQ(t)
	emetteur := nouvelEmetteurDeJetons(t)
	adresses := lancerLeProcessusPlaceDeMarche(t, construireBinaire(t, ".", serviceName), canal, base.execution, emetteur)
	e := emetteur.environnementSur(t, "http://"+adresses.api, &http.Client{Timeout: 5 * time.Second})
	sujet := uuid.NewString()
	return &bancMesure{environnement: e, canal: canal, adresses: adresses, sujet: sujet, jeton: bearer(e.jetons.vendeur(sujet)), traces: []string{sujet}}
}

func lancerLeProcessusPlaceDeMarche(t *testing.T, binaire string, canal *canalDeTest, adresseBase string, emetteur emetteurDeJetons) adressesDuService {
	t.Helper()
	adresses := adressesDuService{api: adresseLibre(t), gestion: adresseLibre(t), interne: adresseLibre(t), metriques: adresseLibre(t)}
	variables := canal.environnementDeService(identiteMarche)
	variables["BASE_DE_DONNEES_URL"] = adresseBase
	variables["BASE_DE_DONNEES_TLS_EXIGE"] = "false"
	variables["OIDC_EMETTEUR"] = emetteurAttendu
	variables["OIDC_JEU_DE_CLES_URL"] = emetteur.jwks.URL
	variables["CANAL_EMETTEUR_ATTENDU"] = identiteControle
	variables["ADRESSE_API"] = adresses.api
	variables["ADRESSE_GESTION"] = adresses.gestion
	variables["ADRESSE_INTERNE"] = adresses.interne
	variables["ADRESSE_METRIQUES"] = adresses.metriques
	variables["LIMITE_DEBIT_PAR_MINUTE"] = "6000000"
	variables["LIMITE_DEBIT_RAFALE"] = "100000"
	journal := &journalPartage{}
	processus := exec.Command(binaire, "serve")
	processus.Env = environnementDuProcessus(variables)
	processus.Stdout, processus.Stderr = journal, journal
	if err := processus.Start(); err != nil {
		t.Fatalf("lancement de la Place de marché : %v", err)
	}
	termine := make(chan error, 1)
	go func() { termine <- processus.Wait() }()
	t.Cleanup(func() {
		_ = processus.Process.Kill()
		select {
		case erreur := <-termine:
			t.Logf("arrêt de la Place de marché : %v", erreur)
		case <-time.After(10 * time.Second):
			t.Error("la Place de marché ne s'est pas arrêtée en 10 s")
		}
	})
	client := &http.Client{Timeout: 2 * time.Second}
	attendre(t, 30*time.Second, "démarrage de la Place de marché", func() bool {
		select {
		case err := <-termine:
			t.Fatalf("la Place de marché s'est arrêtée : %v\n%s", err, journal.tampon.String())
		default:
		}
		return repond(client, "http://"+adresses.gestion+"/disponibilite")
	})
	return adresses
}

func (b *bancMesure) metriques(t *testing.T) releveDeMetriques {
	t.Helper()
	return releverLesMetriques(t, b.adresses.metriques)
}

func (b *bancMesure) attendre(t *testing.T, quoi string, condition func(releveDeMetriques) bool) releveDeMetriques {
	t.Helper()
	return attendreLaMesure(t, b.adresses.metriques, quoi, condition)
}

func (b *bancMesure) creerUnArticle(t *testing.T, titre string) articleRecu {
	t.Helper()
	article := b.creer(t, b.jeton, corpsComplet(titre, categorieSure, prixCourant, 0))
	b.traces = append(b.traces, article.ID)
	return article
}

func (b *bancMesure) publierEtTracer(t *testing.T, identite string, p publication) {
	t.Helper()
	b.traces = append(b.traces, p.id)
	if err := b.canal.publier(t, identite, p); err != nil {
		t.Fatalf("publication en %s : %v", identite, err)
	}
}

func TestUS015MetriquesDeLaPlaceDeMarche(t *testing.T) {
	b := demarrerBancMesure(t)
	t.Run("AC1 requêtes publiques comptées sous leur route paramétrée", b.requetesPubliquesComptees)
	t.Run("AC1 erreurs comptées par route et par code", b.erreursCompteesParRoute)
	t.Run("AC1 API interne mesurée sous sa route paramétrée", b.apiInterneMesuree)
	t.Run("AC5 métriques absentes des ports public, interne et de santé", b.metriquesAbsentesDesAutresPorts)
	t.Run("AC5 port de métriques limité à GET /metrics", b.portDeMetriquesLimite)
	t.Run("AC1 demande publiée par le relais comptée", b.demandePublieeComptee)
	t.Run("AC1 outbox vide mesurée à zéro", b.outboxVideAZero)
	t.Run("AC1 résultat consommé compté sur sa file", b.resultatConsommeCompte)
	t.Run("AC1 résultat d'un émetteur inattendu compté en écarté", b.resultatEmetteurInattenduEcarte)
	t.Run("AC1 résultat non conforme compté en écarté", b.resultatNonConformeEcarte)
	t.Run("AC1 aucun identifiant ni donnée personnelle en étiquette", b.aucunIdentifiantEnEtiquette)
	t.Run("AC1 outbox non publiée : longueur et âge croissent relais arrêté", b.outboxBloqueeMesuree)
}

func (b *bancMesure) requetesPubliquesComptees(t *testing.T) {
	avant, _ := b.metriques(t).requetesSousLaRoute(http.MethodGet, cheminArticles+"/", "200")
	premier, second := b.creerUnArticle(t, "Figurine mesurée 1"), b.creerUnArticle(t, "Figurine mesurée 2")
	b.relire(t, b.jeton, premier.ID)
	b.relire(t, b.jeton, second.ID)
	r := b.metriques(t)
	apres, routes := r.requetesSousLaRoute(http.MethodGet, cheminArticles+"/", "200")
	if apres-avant != 2 {
		t.Fatalf("GET article : +%v requête(s) mesurée(s), +2 attendu\n%s", apres-avant, r.brut)
	}
	exigerRoutesParametrees(t, routes)
	if creations, _ := r.requetesSousLaRoute(http.MethodPost, cheminArticles, "201"); creations < 2 {
		t.Fatalf("POST article : %v création(s) mesurée(s), 2 attendues", creations)
	}
}

func (b *bancMesure) erreursCompteesParRoute(t *testing.T) {
	avant, _ := b.metriques(t).requetesSousLaRoute(http.MethodGet, cheminArticles+"/", "404")
	inconnu := uuid.NewString()
	b.traces = append(b.traces, inconnu)
	b.lireArticle(t, b.jeton, inconnu).exiger(t, http.StatusNotFound)
	r := b.metriques(t)
	apres, routes := r.requetesSousLaRoute(http.MethodGet, cheminArticles+"/", "404")
	if apres-avant != 1 {
		t.Fatalf("GET article inconnu : +%v erreur(s) mesurée(s), +1 attendue\n%s", apres-avant, r.brut)
	}
	exigerRoutesParametrees(t, routes)
}

func (b *bancMesure) apiInterneMesuree(t *testing.T) {
	article := uuid.NewString()
	b.traces = append(b.traces, article)
	requete, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://"+b.adresses.interne+cheminDeReservation(article), nil)
	if err != nil {
		t.Fatalf("requête interne : %v", err)
	}
	reponse, err := b.client.Do(requete)
	if err != nil {
		t.Fatalf("appel interne : %v", err)
	}
	_ = reponse.Body.Close()
	r := b.metriques(t)
	mesurees, routes := r.requetesSousLaRoute(http.MethodPost, "/interne/", strconv.Itoa(reponse.StatusCode))
	if mesurees < 1 {
		t.Fatalf("appel interne (%d) non mesuré\n%s", reponse.StatusCode, r.brut)
	}
	exigerRoutesParametrees(t, routes)
}

func (b *bancMesure) metriquesAbsentesDesAutresPorts(t *testing.T) {
	for _, adresse := range []string{b.adresses.api, b.adresses.interne, b.adresses.gestion} {
		exigerMetriquesAbsentes(t, "http://"+adresse)
	}
}

func (b *bancMesure) portDeMetriquesLimite(t *testing.T) {
	exigerSeulementMetricsSurLePortDeMesure(t, b.adresses.metriques, "/", "/sante", "/disponibilite", "/marche/v1/categories", cheminArticles)
}

func (b *bancMesure) demandePublieeComptee(t *testing.T) {
	avant := b.metriques(t).total(messagesPublies, nil)
	b.soumis = b.soumettreAvecSucces(t, b.jeton, b.creerUnArticle(t, "Figurine soumise"))
	r := b.attendre(t, "publication de la demande", func(r releveDeMetriques) bool {
		return r.total(messagesPublies, nil) > avant
	})
	if len(r.retenir(messagesPublies, map[string]string{"echange": echangeDemandes})) != 1 {
		t.Fatalf("aucune série %s{echange=%q}\n%s", messagesPublies, echangeDemandes, r.brut)
	}
}

func (b *bancMesure) outboxVideAZero(t *testing.T) {
	b.attendre(t, "outbox vidée", func(r releveDeMetriques) bool {
		return len(r.retenir(outboxNonPubliees, nil)) == 1 && r.total(outboxNonPubliees, nil) == 0 &&
			len(r.retenir(ageOutbox, nil)) == 1 && r.total(ageOutbox, nil) == 0
	})
}

func (b *bancMesure) resultatConsommeCompte(t *testing.T) {
	file := map[string]string{"file": fileResultats}
	avant := b.metriques(t).total(messagesConsommes, file)
	b.publierEtTracer(t, identiteControle, versLesResultats(identiteControle, resultatForge(b.soumis, issuePublie, nil)))
	b.attendre(t, "consommation du résultat", func(r releveDeMetriques) bool {
		return r.total(messagesConsommes, file) == avant+1
	})
}

func (b *bancMesure) resultatEmetteurInattenduEcarte(t *testing.T) {
	ecartes := map[string]string{"file": fileResultats, "motif": motifEmetteurInattendu}
	avant := b.metriques(t).total(messagesEcartes, ecartes)
	b.publierEtTracer(t, identiteAdmin, versLesResultats(identiteAdmin, resultatForge(b.soumis, issuePublie, nil)))
	b.attendre(t, "mise à l'écart de l'émetteur inattendu", func(r releveDeMetriques) bool {
		return r.total(messagesEcartes, ecartes) == avant+1
	})
}

func (b *bancMesure) resultatNonConformeEcarte(t *testing.T) {
	ecartes := map[string]string{"file": fileResultats, "motif": motifRefuseParLeTraitement}
	avant := b.metriques(t).total(messagesEcartes, ecartes)
	b.publierEtTracer(t, identiteControle, versLesResultats(identiteControle, []byte(`{"id":"tronqué"`)))
	b.attendre(t, "mise à l'écart du résultat non conforme", func(r releveDeMetriques) bool {
		return r.total(messagesEcartes, ecartes) == avant+1
	})
}

func (b *bancMesure) aucunIdentifiantEnEtiquette(t *testing.T) {
	exigerAucuneDonneeIdentifiante(t, b.metriques(t), append(b.traces, "Figurine mesurée", "Figurine soumise")...)
}

func (b *bancMesure) outboxBloqueeMesuree(t *testing.T) {
	arret := 10 * time.Second
	if err := b.canal.conteneur.Stop(contexteBorne(t, time.Minute), &arret); err != nil {
		t.Fatalf("arrêt du broker : %v", err)
	}
	b.soumettreAvecSucces(t, b.jeton, b.creerUnArticle(t, "Figurine bloquée"))
	premier := b.attendre(t, "demande en attente dans l'outbox", func(r releveDeMetriques) bool {
		return r.total(outboxNonPubliees, nil) >= 1 && r.total(ageOutbox, nil) > 0
	})
	time.Sleep(observationDeLAge)
	second := b.metriques(t)
	if second.total(outboxNonPubliees, nil) < 1 {
		t.Fatalf("outbox_non_publies retombé à %v relais arrêté", second.total(outboxNonPubliees, nil))
	}
	if croissance := second.total(ageOutbox, nil) - premier.total(ageOutbox, nil); croissance < observationDeLAge.Seconds()-1 {
		t.Fatalf("âge de l'outbox : +%.2f s en %s, croissance attendue", croissance, observationDeLAge)
	}
	t.Logf("relais arrêté : %s=%v, %s=%.2f s, %s=%v", outboxNonPubliees, second.total(outboxNonPubliees, nil), ageOutbox, second.total(ageOutbox, nil), relancesOutbox, second.total(relancesOutbox, nil))
}
