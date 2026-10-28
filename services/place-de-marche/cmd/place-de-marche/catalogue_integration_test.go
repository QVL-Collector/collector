package main

import (
	"slices"
	"strings"
	"testing"

	"collector/services/place-de-marche/internal/migrations"
)

func exigerIndexPartielsDuCatalogue(t *testing.T, base basePostgreSQL) {
	t.Helper()
	var definitions []string
	lignes, err := base.maitre.Query(t.Context(), `
		SELECT indexdef FROM pg_indexes
		WHERE schemaname = 'public' AND tablename = 'article' AND indexdef ILIKE '% WHERE %publie%'`)
	if err != nil {
		t.Fatalf("inventaire des index : %v", err)
	}
	defer lignes.Close()
	for lignes.Next() {
		var definition string
		if err := lignes.Scan(&definition); err != nil {
			t.Fatalf("lecture d'un index : %v", err)
		}
		definitions = append(definitions, definition)
	}
	if len(definitions) == 0 {
		t.Fatal("aucun index partiel restreint aux articles publiés sur la table article")
	}
	t.Logf("index partiels des publiés : %v", definitions)
}

func TestUS023IntegrationPostgreSQL(t *testing.T) {
	base, pool, service := servicePostgreSQL(t)
	b := monterBanc(t, service, nil)
	for _, s := range scenariosDuCatalogue() {
		t.Run(s.nom, func(t *testing.T) {
			s.jouer(t, b)
		})
	}
	t.Run("réponses du catalogue conformes au contrat OpenAPI sur PostgreSQL", func(t *testing.T) {
		routeur := chargerContrat(t, b.adresse)
		for _, ech := range b.historique() {
			if strings.HasPrefix(ech.requete.URL.Path, cheminCatalogue) {
				verifierEchange(t, routeur, ech)
			}
		}
	})
	t.Run("index partiels du catalogue posés par les migrations", func(t *testing.T) {
		exigerIndexPartielsDuCatalogue(t, base)
	})
	t.Run("migrations rejouées sur une base peuplée sans effet sur le catalogue", func(t *testing.T) {
		avant := b.toutLeCatalogue(t, "")
		if err := migrations.Apply(t.Context(), journalMuet(), base.migration); err != nil {
			t.Fatalf("migrations rejouées : %v", err)
		}
		if apres := b.toutLeCatalogue(t, ""); !slices.Equal(identifiants(apres), identifiants(avant)) {
			t.Fatalf("catalogue modifié par le rejeu des migrations : %d puis %d article(s)", len(avant), len(apres))
		}
		exigerIndexPartielsDuCatalogue(t, base)
	})
	t.Run("rôle d'exécution limité au DML de ses tables", func(t *testing.T) {
		verifierDroitsMinimaux(t, base, pool)
	})
}
