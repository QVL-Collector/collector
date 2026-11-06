package main

import (
	"net/http"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

type ligneDecisionAdmin struct {
	adminID  string
	revision int
	decision string
	motif    *string
}

type bancAdministrationPostgreSQL struct {
	*bancEnMemoire
	base  basePostgreSQL
	admin administrateur
}

func demarrerBancAdministrationPostgreSQL(t *testing.T) *bancAdministrationPostgreSQL {
	t.Helper()
	base, _, service := servicePostgreSQL(t)
	b := monterBanc(t, service, nil)
	return &bancAdministrationPostgreSQL{bancEnMemoire: b, base: base, admin: b.administrateur()}
}

func (b *bancAdministrationPostgreSQL) decisionsDe(t *testing.T, id string) []ligneDecisionAdmin {
	t.Helper()
	lignes, err := b.base.maitre.Query(t.Context(), `SELECT admin_id::text, revision, decision, motif FROM decision_admin WHERE article_id = $1 ORDER BY decide_le`, id)
	if err != nil {
		t.Fatalf("lecture de decision_admin : %v", err)
	}
	decisions, err := pgx.CollectRows(lignes, func(r pgx.CollectableRow) (ligneDecisionAdmin, error) {
		var l ligneDecisionAdmin
		return l, r.Scan(&l.adminID, &l.revision, &l.decision, &l.motif)
	})
	if err != nil {
		t.Fatalf("lecture de decision_admin : %v", err)
	}
	return decisions
}

func (b *bancAdministrationPostgreSQL) exigerDecisionEnregistree(t *testing.T, article articleRecu, decision string, motif *string) {
	t.Helper()
	lignes := b.decisionsDe(t, article.ID)
	if len(lignes) != 1 {
		t.Fatalf("%d ligne(s) decision_admin pour %s, 1 attendue", len(lignes), article.ID)
	}
	l := lignes[0]
	if l.adminID != b.admin.sujet || l.revision != article.Revision || l.decision != decision || !motifsEgaux(l.motif, motif) {
		t.Fatalf("ligne decision_admin %+v (motif %v), attendue admin %s, révision %d, décision %s, motif %v",
			l, valeurDuMotif(l.motif), b.admin.sujet, article.Revision, decision, valeurDuMotif(motif))
	}
}

func motifsEgaux(a, b *string) bool {
	return valeurDuMotif(a) == valeurDuMotif(b)
}

func valeurDuMotif(m *string) string {
	if m == nil {
		return "<null>"
	}
	return *m
}

func (b *bancAdministrationPostgreSQL) exigerAucuneDecision(t *testing.T, id string) {
	t.Helper()
	if lignes := b.decisionsDe(t, id); len(lignes) != 0 {
		t.Fatalf("%d ligne(s) decision_admin pour %s alors qu'aucune décision n'a été appliquée : %+v", len(lignes), id, lignes)
	}
}

func TestUS022IntegrationPostgreSQL(t *testing.T) {
	b := demarrerBancAdministrationPostgreSQL(t)
	t.Run("AC1 seuls les articles à vérifier, du plus ancien au plus récent, avec vendeur et motif", b.ac1ListeSurPostgreSQL)
	t.Run("AC2 AC4 publier : au catalogue, ligne decision_admin exacte et événement journalisé", b.ac2ac4PublierSurPostgreSQL)
	t.Run("AC2 AC4 refuser : le vendeur lit refusé et le motif, ligne decision_admin exacte", b.ac2ac4RefuserSurPostgreSQL)
	t.Run("AC3 décision sur la révision N après passage en N+1 : 409, rien d'appliqué ni d'enregistré", b.ac3RevisionPerimeeSurPostgreSQL)
	t.Run("AC3 décisions concurrentes sur la même révision : une seule appliquée et enregistrée", b.ac3DecisionsConcurrentesSurPostgreSQL)
	t.Run("décisions invalides ou interdites sans ligne decision_admin", b.decisionsRefuseesSansLigneSurPostgreSQL)
	t.Run("réponses de l'administration conformes au contrat OpenAPI sur PostgreSQL", func(t *testing.T) {
		jouerLesStatutsDeLAdministration(t, b.bancEnMemoire)
		routeur := chargerContrat(t, b.adresse)
		for _, ech := range b.historique() {
			verifierEchange(t, routeur, ech)
		}
	})
}

func (b *bancAdministrationPostgreSQL) ac1ListeSurPostgreSQL(t *testing.T) {
	v := b.nouveauVendeur()
	premier := b.aVerifier(t, v, "PG premier à vérifier")
	for _, etat := range []string{etatBrouillon, etatEnControle, etatRefuse, etatPublie} {
		b.articleDansLEtat(t, v.jeton, "PG hors liste "+etat, categorieSure, etat)
	}
	autre := b.nouveauVendeur()
	second := b.aVerifier(t, autre, "PG second à vérifier")

	elements := b.listeAVerifier(t, b.admin.jeton, "?taille=50").Elements

	var vus []articleAVerifierRecu
	for _, e := range elements {
		if e.ID == premier.ID || e.ID == second.ID {
			vus = append(vus, e)
		}
		if e.Revision < 1 || e.Motif == "" {
			t.Errorf("article listé incomplet : %+v", e)
		}
	}
	if len(vus) != 2 || vus[0].ID != premier.ID || vus[1].ID != second.ID {
		t.Fatalf("articles listés %+v, attendus %s puis %s", vus, premier.ID, second.ID)
	}
	if vus[0].VendeurID != v.sujet || vus[1].VendeurID != autre.sujet || vus[0].Titre != premier.Titre || vus[0].Motif != motifCategorieInconnu {
		t.Fatalf("articles listés %+v, vendeurs %s et %s attendus avec le motif %s", vus, v.sujet, autre.sujet, motifCategorieInconnu)
	}
	for _, e := range elements {
		if b.relire(t, bearer(b.jetons.vendeur(e.VendeurID)), e.ID).Etat != etatAVerifier {
			t.Errorf("article %s listé sans être à vérifier", e.ID)
		}
	}
}

func (b *bancAdministrationPostgreSQL) ac2ac4PublierSurPostgreSQL(t *testing.T) {
	v := b.nouveauVendeur()
	article := b.aVerifier(t, v, "PG à publier")

	b.deciderAvecSucces(t, b.admin.jeton, article.ID, article.Revision, decisionPublier, nil)

	exigerEtat(t, b.relire(t, v.jeton, article.ID), etatPublie, "")
	exigerPresents(t, b.toutLeCatalogue(t, ""), article)
	b.exigerDecisionEnregistree(t, article, decisionPublier, nil)
	exigerEvenementDeDecision(t, b.journalHTTP, b.admin, article, decisionPublier, nil)
}

func (b *bancAdministrationPostgreSQL) ac2ac4RefuserSurPostgreSQL(t *testing.T) {
	v := b.nouveauVendeur()
	article := b.aVerifier(t, v, "PG à refuser")
	code := motif("prix_abusif")

	b.deciderAvecSucces(t, b.admin.jeton, article.ID, article.Revision, decisionRefuser, code)

	exigerEtat(t, b.relire(t, v.jeton, article.ID), etatRefuse, *code)
	exigerEtat(t, b.dansMesArticles(t, v.jeton, article.ID), etatRefuse, *code)
	exigerAbsents(t, b.toutLeCatalogue(t, ""), article)
	b.exigerDecisionEnregistree(t, article, decisionRefuser, code)
	exigerEvenementDeDecision(t, b.journalHTTP, b.admin, article, decisionRefuser, code)
}

func (b *bancAdministrationPostgreSQL) ac3RevisionPerimeeSurPostgreSQL(t *testing.T) {
	v := b.nouveauVendeur()
	ancienne := b.aVerifier(t, v, titreInitial)
	modifie := b.modifierAvecSucces(t, v.jeton, ancienne, corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux))
	b.trancher(t, modifie, issueAVerifier, motif(motifCategorieInconnu))
	avant := b.relire(t, v.jeton, ancienne.ID)
	if avant.Revision != ancienne.Revision+1 || avant.Etat != etatAVerifier {
		t.Fatalf("article %+v, attendu à vérifier en révision %d", avant, ancienne.Revision+1)
	}

	for _, corps := range []string{
		corpsDecision(ancienne.Revision, decisionPublier, nil),
		corpsDecision(ancienne.Revision, decisionRefuser, motif("article_interdit")),
	} {
		p := b.decider(t, b.admin.jeton, ancienne.ID, corps).probleme(t, http.StatusConflict)
		exigerTypeDeProbleme(t, p, typeRevisionPerimee)
	}

	b.exigerSansEffet(t, b.admin, v, avant)
	b.exigerAucuneDecision(t, ancienne.ID)
	b.deciderAvecSucces(t, b.admin.jeton, avant.ID, avant.Revision, decisionPublier, nil)
	b.exigerDecisionEnregistree(t, avant, decisionPublier, nil)
}

func (b *bancAdministrationPostgreSQL) ac3DecisionsConcurrentesSurPostgreSQL(t *testing.T) {
	v := b.nouveauVendeur()
	article := b.aVerifier(t, v, "PG décisions concurrentes")
	corps := []string{
		corpsDecision(article.Revision, decisionPublier, nil),
		corpsDecision(article.Revision, decisionRefuser, motif("vendeur_non_fiable")),
		corpsDecision(article.Revision, decisionPublier, nil),
		corpsDecision(article.Revision, decisionRefuser, motif("annonce_incomplete")),
	}
	statuts := make([]int, len(corps))
	var groupe sync.WaitGroup
	for i, c := range corps {
		groupe.Go(func() {
			statuts[i] = b.decider(t, b.admin.jeton, article.ID, c).statut
		})
	}
	groupe.Wait()

	acceptees := 0
	for _, s := range statuts {
		switch s {
		case http.StatusOK:
			acceptees++
		case http.StatusConflict:
		default:
			t.Fatalf("statuts %v : seuls 200 et 409 attendus", statuts)
		}
	}
	if acceptees != 1 {
		t.Fatalf("statuts %v : une seule décision appliquée attendue", statuts)
	}
	lignes := b.decisionsDe(t, article.ID)
	if len(lignes) != 1 {
		t.Fatalf("%d ligne(s) decision_admin, 1 attendue", len(lignes))
	}
	attendu := etatPublie
	if lignes[0].decision == decisionRefuser {
		attendu = etatRefuse
	}
	motifAttendu := ""
	if lignes[0].motif != nil {
		motifAttendu = *lignes[0].motif
	}
	exigerEtat(t, b.relire(t, v.jeton, article.ID), attendu, motifAttendu)
}

func (b *bancAdministrationPostgreSQL) decisionsRefuseesSansLigneSurPostgreSQL(t *testing.T) {
	v := b.nouveauVendeur()
	article := b.aVerifier(t, v, "PG décisions refusées")
	b.decider(t, b.admin.jeton, article.ID, corpsDecision(article.Revision, decisionRefuser, nil)).exiger(t, http.StatusUnprocessableEntity)
	b.decider(t, b.admin.jeton, article.ID, corpsDecision(article.Revision, decisionRefuser, motif("hors_liste"))).exiger(t, http.StatusUnprocessableEntity)
	b.decider(t, b.admin.jeton, article.ID, `{"revision":0,"decision":"publier"}`).exiger(t, http.StatusBadRequest)
	b.decider(t, v.jeton, article.ID, corpsDecision(article.Revision, decisionPublier, nil)).exiger(t, http.StatusForbidden)
	b.decider(t, "", article.ID, corpsDecision(article.Revision, decisionPublier, nil)).exiger(t, http.StatusUnauthorized)

	b.exigerSansEffet(t, b.admin, v, article)
	b.exigerAucuneDecision(t, article.ID)
}
