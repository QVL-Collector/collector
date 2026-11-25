package main

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

const fileDesDemandes = "controle.demandes"

func TestUS015MetriquesDuControle(t *testing.T) {
	canal := demarrerRabbitMQ(t)
	controle := controleDisponible(t, construireControle(t), canal)
	t.Run("AC5 métriques absentes du port de santé", func(t *testing.T) {
		exigerMetriquesAbsentes(t, "http://"+controle.gestion)
	})
	t.Run("AC5 port de métriques limité à GET /metrics", func(t *testing.T) {
		exigerSeulementMetricsSurLePortDeMesure(t, controle.metriques, "/", "/sante", "/disponibilite")
	})
	var traces []string
	t.Run("AC1 demande non conforme comptée en écartée sur sa file", func(t *testing.T) {
		ecartes := map[string]string{"file": fileDesDemandes, "motif": motifRefuseParLeTraitement}
		avant := releverLesMetriques(t, controle.metriques).total(messagesEcartes, ecartes)
		id, err := canal.publierUneDemande(t)
		if err != nil {
			t.Fatalf("publication de la demande : %v", err)
		}
		traces = append(traces, id)
		attendreLaMesure(t, controle.metriques, "mise à l'écart de la demande", func(r releveDeMetriques) bool {
			return r.total(messagesEcartes, ecartes) == avant+1
		})
	})
	t.Run("AC1 demande d'un émetteur inattendu comptée en écartée", func(t *testing.T) {
		ecartes := map[string]string{"file": fileDesDemandes, "motif": motifEmetteurInattendu}
		avant := releverLesMetriques(t, controle.metriques).total(messagesEcartes, ecartes)
		id := uuid.NewString()
		traces = append(traces, id)
		if err := canal.publier(t, identiteAdmin, publication{echange: echangeDemandes, cle: cleDeRoutageV1, id: id, emetteur: identiteAdmin, corps: []byte(`{}`)}); err != nil {
			t.Fatalf("publication de la demande usurpée : %v", err)
		}
		attendreLaMesure(t, controle.metriques, "mise à l'écart de l'émetteur inattendu", func(r releveDeMetriques) bool {
			return r.total(messagesEcartes, ecartes) == avant+1
		})
	})
	t.Run("AC1 aucun identifiant de message en étiquette", func(t *testing.T) {
		exigerAucuneDonneeIdentifiante(t, releverLesMetriques(t, controle.metriques), traces...)
	})
}

func TestUS015MetriquesDeTransactions(t *testing.T) {
	b := demarrerBancReservation(t)
	commandes := []string{uuid.NewString(), uuid.NewString()}
	t.Run("AC1 requêtes comptées sous la route paramétrée avec leur code", func(t *testing.T) {
		avant, _ := releverLesMetriques(t, b.tx.metriques).requetesSousLaRoute(http.MethodGet, cheminCommandes+"/", "401")
		for _, id := range commandes {
			b.lireCommande(t, "", id).exiger(t, http.StatusUnauthorized)
		}
		r := releverLesMetriques(t, b.tx.metriques)
		apres, routes := r.requetesSousLaRoute(http.MethodGet, cheminCommandes+"/", "401")
		if apres-avant != 2 {
			t.Fatalf("GET commande : +%v requête(s) mesurée(s), +2 attendu\n%s", apres-avant, r.brut)
		}
		exigerRoutesParametrees(t, routes)
		exigerAucuneDonneeIdentifiante(t, r, commandes...)
	})
	t.Run("AC5 métriques absentes des ports public et de santé", func(t *testing.T) {
		exigerMetriquesAbsentes(t, "http://"+b.tx.api)
		exigerMetriquesAbsentes(t, "http://"+b.tx.gestion)
	})
	t.Run("AC5 port de métriques limité à GET /metrics", func(t *testing.T) {
		exigerSeulementMetricsSurLePortDeMesure(t, b.tx.metriques, "/", "/sante", "/disponibilite", cheminCommandes)
	})
}
