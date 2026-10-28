package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

const (
	cheminAdmin            = "/marche/v1/admin/articles"
	roleAdmin              = "admin"
	decisionPublier        = "publier"
	decisionRefuser        = "refuser"
	evenementDecisionAdmin = "annonces.decision_admin"
	typeRevisionPerimee    = "urn:collector:probleme:revision-perimee"
	typeNonAVerifier       = "urn:collector:probleme:article-non-a-verifier"
	typeSaisieInvalide     = "urn:collector:probleme:saisie-invalide"
	codeObligatoire        = "obligatoire"
)

var motifsDeRefus = []string{"annonce_incomplete", "coordonnees_detectees", "prix_abusif", "article_interdit", "vendeur_non_fiable"}

type articleAVerifierRecu struct {
	ID          string          `json:"id"`
	Revision    int             `json:"revision"`
	Titre       string          `json:"titre"`
	Description string          `json:"description"`
	Categorie   *categorieRecue `json:"categorie"`
	Prix        *montant        `json:"prix"`
	FraisDePort *montant        `json:"fraisDePort"`
	Motif       string          `json:"motif"`
	VendeurID   string          `json:"vendeurId"`
	ModifieLe   string          `json:"modifieLe"`
}

type pageAVerifier struct {
	Elements []articleAVerifierRecu `json:"elements"`
	Page     int                    `json:"page"`
	Taille   int                    `json:"taille"`
	Total    int                    `json:"total"`
}

type administrateur struct {
	sujet string
	jeton string
}

func (e *environnement) administrateur() administrateur {
	sujet := uuid.NewString()
	return administrateur{sujet: sujet, jeton: bearer(e.jetons.emettre(sujet, []string{roleAdmin}))}
}

func cheminDecision(id string) string {
	return cheminAdmin + "/" + id + "/decision"
}

func corpsDecision(revision int, decision string, motif *string) string {
	if motif == nil {
		return fmt.Sprintf(`{"revision":%d,"decision":%q}`, revision, decision)
	}
	return fmt.Sprintf(`{"revision":%d,"decision":%q,"motif":%q}`, revision, decision, *motif)
}

func (e *environnement) decider(t *testing.T, jeton, id, corps string) reponse {
	t.Helper()
	return e.envoyer(t, http.MethodPost, cheminDecision(id), jeton, corpsJSON(corps))
}

func (e *environnement) deciderAvecSucces(t *testing.T, jeton string, id string, revision int, decision string, motif *string) articleRecu {
	t.Helper()
	var tranche articleRecu
	e.decider(t, jeton, id, corpsDecision(revision, decision, motif)).exiger(t, http.StatusOK).lire(t, &tranche)
	return tranche
}

func (e *environnement) listeAVerifier(t *testing.T, jeton, requete string) pageAVerifier {
	t.Helper()
	var page pageAVerifier
	e.envoyer(t, http.MethodGet, cheminAdmin+requete, jeton).exiger(t, http.StatusOK).lire(t, &page)
	return page
}

func (e *environnement) idsAVerifier(t *testing.T, jeton string) []string {
	t.Helper()
	page := e.listeAVerifier(t, jeton, fmt.Sprintf("?taille=%d", tailleMaximale))
	ids := make([]string, 0, len(page.Elements))
	for _, a := range page.Elements {
		ids = append(ids, a.ID)
	}
	return ids
}

func (e *environnement) exigerDansLaListe(t *testing.T, jeton string, id string, attendu bool) {
	t.Helper()
	if present := slices.Contains(e.idsAVerifier(t, jeton), id); present != attendu {
		t.Fatalf("article %s présent dans la liste à vérifier : %t, attendu %t", id, present, attendu)
	}
}

func (b *bancEnMemoire) aVerifier(t *testing.T, v vendeurConnu, titre string) articleRecu {
	t.Helper()
	return b.articleDansLEtat(t, v.jeton, titre, categorieSure, etatAVerifier)
}

func (b *bancEnMemoire) exigerSansEffet(t *testing.T, admin administrateur, v vendeurConnu, avant articleRecu) {
	t.Helper()
	if apres := b.relire(t, v.jeton, avant.ID); !reflect.DeepEqual(apres, avant) {
		t.Fatalf("décision refusée avec effet : %+v devenu %+v", avant, apres)
	}
	if avant.Etat == etatAVerifier {
		b.exigerDansLaListe(t, admin.jeton, avant.ID, true)
	}
	if avant.Etat != etatPublie {
		exigerAbsents(t, b.toutLeCatalogue(t, ""), avant)
	}
}

func exigerTypeDeProbleme(t *testing.T, p probleme, attendu string) {
	t.Helper()
	if p.Type != attendu {
		t.Fatalf("problème de type %q, %q attendu", p.Type, attendu)
	}
}

func (j *journalPartage) evenements(t *testing.T, nom string) []map[string]any {
	t.Helper()
	j.mu.Lock()
	defer j.mu.Unlock()
	var trouves []map[string]any
	lecteur := bufio.NewScanner(strings.NewReader(j.tampon.String()))
	for lecteur.Scan() {
		var ligne map[string]any
		if err := json.Unmarshal(lecteur.Bytes(), &ligne); err != nil {
			t.Fatalf("ligne de journal non JSON : %v : %s", err, lecteur.Text())
		}
		if ligne["event"] == nom {
			trouves = append(trouves, ligne)
		}
	}
	return trouves
}

func feuilles(valeur any) []string {
	switch v := valeur.(type) {
	case map[string]any:
		var toutes []string
		for _, sous := range v {
			toutes = append(toutes, feuilles(sous)...)
		}
		return toutes
	case []any:
		var toutes []string
		for _, sous := range v {
			toutes = append(toutes, feuilles(sous)...)
		}
		return toutes
	default:
		return []string{fmt.Sprint(v)}
	}
}

func exigerEvenementDeDecision(t *testing.T, j *journalPartage, admin administrateur, article articleRecu, decision string, motif *string) {
	t.Helper()
	evenements := j.evenements(t, evenementDecisionAdmin)
	var portantSurLArticle []map[string]any
	for _, ev := range evenements {
		if slices.Contains(feuilles(ev), article.ID) {
			portantSurLArticle = append(portantSurLArticle, ev)
		}
	}
	if len(portantSurLArticle) != 1 {
		t.Fatalf("%d événement(s) %s pour l'article %s, 1 attendu : %v", len(portantSurLArticle), evenementDecisionAdmin, article.ID, evenements)
	}
	ev := portantSurLArticle[0]
	if ev["level"] != "INFO" {
		t.Errorf("événement de niveau %v, INFO attendu", ev["level"])
	}
	valeurs := feuilles(ev)
	attendues := []string{admin.sujet, article.ID, fmt.Sprint(article.Revision), decision}
	if motif != nil {
		attendues = append(attendues, *motif)
	}
	for _, attendue := range attendues {
		if !slices.Contains(valeurs, attendue) {
			t.Errorf("événement %v sans la valeur %q", ev, attendue)
		}
	}
}

func TestUS022AC1ListeLesSeulsArticlesAVerifierDuPlusAncienAuPlusRecent(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	admin := b.administrateur()
	v := b.nouveauVendeur()
	premier := b.aVerifier(t, v, "Premier à vérifier")
	for _, etat := range []string{etatBrouillon, etatEnControle, etatRefuse, etatPublie} {
		b.articleDansLEtat(t, v.jeton, "Hors liste "+etat, categorieSure, etat)
	}
	deuxieme := b.aVerifier(t, b.nouveauVendeur(), "Deuxième à vérifier")
	troisieme := b.aVerifier(t, v, "Troisième à vérifier")

	page := b.listeAVerifier(t, admin.jeton, "")

	attendus := []string{premier.ID, deuxieme.ID, troisieme.ID}
	if ids := b.idsAVerifier(t, admin.jeton); !slices.Equal(ids, attendus) || page.Total != 3 {
		t.Fatalf("liste %v (total %d), attendue %v du plus ancien au plus récent", ids, page.Total, attendus)
	}
}

func TestUS022AC1ChaqueArticleAvecContenuRevisionMotifEtVendeur(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	admin := b.administrateur()
	v := b.nouveauVendeur()
	soumis := b.soumettreAvecSucces(t, v.jeton, b.creer(t, v.jeton, corpsComplet("Figurine Spirou", categorieSure, 98000, 650)))
	b.trancher(t, soumis, issueAVerifier, motif(motifVendeurAVerifier))
	article := b.relire(t, v.jeton, soumis.ID)

	elements := b.listeAVerifier(t, admin.jeton, "").Elements

	if len(elements) != 1 {
		t.Fatalf("%d article(s) listé(s), 1 attendu", len(elements))
	}
	recu := elements[0]
	attendu := articleAVerifierRecu{
		ID: article.ID, Revision: article.Revision, Titre: article.Titre, Description: article.Description,
		Categorie: article.Categorie, Prix: article.Prix, FraisDePort: article.FraisDePort,
		Motif: motifVendeurAVerifier, VendeurID: v.sujet, ModifieLe: article.ModifieLe,
	}
	if !reflect.DeepEqual(recu, attendu) {
		t.Fatalf("article listé %+v, attendu %+v", recu, attendu)
	}
}

func TestUS022AC1AucunArticleAVerifierListeVide(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	b.publier(t, b.vendeur(), "Publié", categorieSure)

	reponse := b.envoyer(t, http.MethodGet, cheminAdmin, b.administrateur().jeton).exiger(t, http.StatusOK)

	if !strings.Contains(string(reponse.corps), `"elements":[]`) {
		t.Fatalf("liste vide attendue sous la forme d'un tableau vide : %s", reponse.corps)
	}
	var page pageAVerifier
	reponse.lire(t, &page)
	if page.Total != 0 || page.Page != 1 || page.Taille != tailleParDefaut {
		t.Fatalf("page %+v, attendue page 1, taille %d, total 0", page, tailleParDefaut)
	}
}

func TestUS022AC1PaginationDeLaListe(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	admin := b.administrateur()
	v := b.nouveauVendeur()
	var articles []articleRecu
	for i := range 3 {
		articles = append(articles, b.aVerifier(t, v, fmt.Sprintf("À vérifier %d", i)))
	}

	deuxieme := b.listeAVerifier(t, admin.jeton, "?page=2&taille=1")
	if len(deuxieme.Elements) != 1 || deuxieme.Elements[0].ID != articles[1].ID || deuxieme.Total != 3 || deuxieme.Page != 2 || deuxieme.Taille != 1 {
		t.Fatalf("page 2 de taille 1 : %+v, attendu l'article %s sur un total de 3", deuxieme, articles[1].ID)
	}
	if audela := b.listeAVerifier(t, admin.jeton, "?page=4&taille=1"); len(audela.Elements) != 0 || audela.Total != 3 {
		t.Fatalf("page au-delà du total : %+v, vide attendue avec un total de 3", audela)
	}
	for _, requete := range []string{"?taille=0", fmt.Sprintf("?taille=%d", tailleMaximale+1), "?page=0", "?page=abc"} {
		t.Run(requete, func(t *testing.T) {
			b.envoyer(t, http.MethodGet, cheminAdmin+requete, admin.jeton).probleme(t, http.StatusBadRequest)
		})
	}
}

func TestUS022AC1ListeMontreLaRevisionExamineeApresNouveauControle(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	admin := b.administrateur()
	v := b.nouveauVendeur()
	ancienne := b.aVerifier(t, v, titreInitial)
	b.emettreLesDemandes(t)
	modifie := b.modifierAvecSucces(t, v.jeton, ancienne, corpsComplet(titreModifie, categorieHors, prixInitial, fraisInitiaux))
	b.exigerDansLaListe(t, admin.jeton, ancienne.ID, false)

	exigerUnSeulResultatAcquitte(t, b.controler(t))

	elements := b.listeAVerifier(t, admin.jeton, "").Elements
	if len(elements) != 1 || elements[0].Revision != modifie.Revision || elements[0].Titre != titreModifie {
		t.Fatalf("liste %+v, attendu l'article en révision %d au titre %q", elements, modifie.Revision, titreModifie)
	}
}

func TestUS022AC2PublierFaitApparaitreAuCatalogue(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	admin := b.administrateur()
	v := b.nouveauVendeur()
	article := b.aVerifier(t, v, "Figurine à publier")
	exigerAbsents(t, b.toutLeCatalogue(t, ""), article)

	tranche := b.deciderAvecSucces(t, admin.jeton, article.ID, article.Revision, decisionPublier, nil)

	exigerEtat(t, tranche, etatPublie, "")
	if tranche.ID != article.ID || tranche.Titre != article.Titre {
		t.Fatalf("article tranché %+v, attendu %s au titre %q", tranche, article.ID, article.Titre)
	}
	exigerPresents(t, b.toutLeCatalogue(t, ""), article)
	if f := b.lireFiche(t, article.ID); f.Titre != article.Titre {
		t.Fatalf("fiche publique au titre %q, %q attendu", f.Titre, article.Titre)
	}
	exigerEtat(t, b.relire(t, v.jeton, article.ID), etatPublie, "")
	b.exigerDansLaListe(t, admin.jeton, article.ID, false)
}

func TestUS022AC2RefuserAvecChaqueMotifDeLaListeFermee(t *testing.T) {
	for _, code := range motifsDeRefus {
		t.Run(code, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			admin := b.administrateur()
			v := b.nouveauVendeur()
			article := b.aVerifier(t, v, "Figurine à refuser")

			tranche := b.deciderAvecSucces(t, admin.jeton, article.ID, article.Revision, decisionRefuser, motif(code))

			exigerEtat(t, tranche, etatRefuse, code)
			exigerEtat(t, b.relire(t, v.jeton, article.ID), etatRefuse, code)
			exigerEtat(t, b.dansMesArticles(t, v.jeton, article.ID), etatRefuse, code)
			exigerAbsents(t, b.toutLeCatalogue(t, ""), article)
			b.fiche(t, article.ID).probleme(t, http.StatusNotFound)
			b.exigerDansLaListe(t, admin.jeton, article.ID, false)
		})
	}
}

func TestUS022AC3DecisionSurRevisionPerimeeRefuseeSansEffet(t *testing.T) {
	cas := []struct {
		nom      string
		preparer func(t *testing.T, b *bancEnMemoire, v vendeurConnu, a articleRecu)
	}{
		{"article modifié, nouvelle révision en contrôle", func(t *testing.T, b *bancEnMemoire, v vendeurConnu, a articleRecu) {
			b.modifierAvecSucces(t, v.jeton, a, corpsComplet(titreModifie, categorieHors, prixInitial, fraisInitiaux))
		}},
		{"article modifié puis revenu à vérifier", func(t *testing.T, b *bancEnMemoire, v vendeurConnu, a articleRecu) {
			b.modifierAvecSucces(t, v.jeton, a, corpsComplet(titreModifie, categorieHors, prixInitial, fraisInitiaux))
			exigerUnSeulResultatAcquitte(t, b.controler(t))
		}},
	}
	decisions := []struct {
		decision string
		motif    *string
	}{{decisionPublier, nil}, {decisionRefuser, motif("prix_abusif")}}
	for _, c := range cas {
		for _, d := range decisions {
			t.Run(c.nom+" / "+d.decision, func(t *testing.T) {
				b := demarrerBancEnMemoire(t)
				admin := b.administrateur()
				v := b.nouveauVendeur()
				ancienne := b.aVerifier(t, v, titreInitial)
				b.emettreLesDemandes(t)
				c.preparer(t, b, v, ancienne)
				avant := b.relire(t, v.jeton, ancienne.ID)
				if avant.Revision != ancienne.Revision+1 {
					t.Fatalf("révision %d après modification, %d attendue", avant.Revision, ancienne.Revision+1)
				}

				p := b.decider(t, admin.jeton, ancienne.ID, corpsDecision(ancienne.Revision, d.decision, d.motif)).probleme(t, http.StatusConflict)

				exigerTypeDeProbleme(t, p, typeRevisionPerimee)
				b.exigerSansEffet(t, admin, v, avant)
				if n := len(b.journalHTTP.evenements(t, evenementDecisionAdmin)); n != 0 {
					t.Fatalf("%d décision(s) journalisée(s) alors qu'aucune n'a été appliquée", n)
				}
			})
		}
	}
}

func TestUS022AC3RevisionFutureRefusee(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	admin := b.administrateur()
	v := b.nouveauVendeur()
	article := b.aVerifier(t, v, titreInitial)

	p := b.decider(t, admin.jeton, article.ID, corpsDecision(article.Revision+1, decisionPublier, nil)).probleme(t, http.StatusConflict)

	exigerTypeDeProbleme(t, p, typeRevisionPerimee)
	b.exigerSansEffet(t, admin, v, article)
}

func TestUS022AC3NouvelleRevisionTranchableApresRechargement(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	admin := b.administrateur()
	v := b.nouveauVendeur()
	ancienne := b.aVerifier(t, v, titreInitial)
	b.emettreLesDemandes(t)
	b.modifierAvecSucces(t, v.jeton, ancienne, corpsComplet(titreModifie, categorieHors, prixInitial, fraisInitiaux))
	exigerUnSeulResultatAcquitte(t, b.controler(t))
	b.decider(t, admin.jeton, ancienne.ID, corpsDecision(ancienne.Revision, decisionPublier, nil)).probleme(t, http.StatusConflict)

	recharge := b.listeAVerifier(t, admin.jeton, "").Elements[0]
	tranche := b.deciderAvecSucces(t, admin.jeton, recharge.ID, recharge.Revision, decisionPublier, nil)

	exigerEtat(t, tranche, etatPublie, "")
	if tranche.Titre != titreModifie {
		t.Fatalf("titre publié %q, %q attendu", tranche.Titre, titreModifie)
	}
	exigerPresents(t, b.toutLeCatalogue(t, ""), tranche)
}

func TestUS022AC3ArticleQuiNEstPlusAVerifierRefuse(t *testing.T) {
	for _, etat := range []string{etatBrouillon, etatEnControle, etatPublie, etatRefuse} {
		t.Run(etat, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			admin := b.administrateur()
			v := b.nouveauVendeur()
			article := b.amenerDansLEtat(t, v.jeton, b.creer(t, v.jeton, corpsInitial()), etat)

			p := b.decider(t, admin.jeton, article.ID, corpsDecision(article.Revision, decisionPublier, nil)).probleme(t, http.StatusConflict)

			exigerTypeDeProbleme(t, p, typeNonAVerifier)
			b.exigerSansEffet(t, admin, v, article)
		})
	}
}

func TestUS022AC3SecondeDecisionSurLaMemeRevisionRefusee(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	admin := b.administrateur()
	v := b.nouveauVendeur()
	article := b.aVerifier(t, v, titreInitial)
	refuse := b.deciderAvecSucces(t, admin.jeton, article.ID, article.Revision, decisionRefuser, motif("article_interdit"))

	p := b.decider(t, b.administrateur().jeton, article.ID, corpsDecision(article.Revision, decisionPublier, nil)).probleme(t, http.StatusConflict)

	exigerTypeDeProbleme(t, p, typeNonAVerifier)
	b.exigerSansEffet(t, admin, v, refuse)
}

func TestUS022DecisionSurArticleInexistantIntrouvable(t *testing.T) {
	b := demarrerBancEnMemoire(t)

	b.decider(t, b.administrateur().jeton, uuid.NewString(), corpsDecision(1, decisionPublier, nil)).probleme(t, http.StatusNotFound)
}

func TestUS022MotifInvalideRefuseEn422SansEffet(t *testing.T) {
	cas := []struct {
		nom   string
		corps func(revision int) string
		code  string
	}{
		{"refuser sans motif", func(r int) string { return corpsDecision(r, decisionRefuser, nil) }, codeObligatoire},
		{"publier avec un motif", func(r int) string { return corpsDecision(r, decisionPublier, motif("prix_abusif")) }, codeInvalide},
		{"motif du contrôle hors de la liste fermée", func(r int) string { return corpsDecision(r, decisionRefuser, motif(motifCategorieInconnu)) }, codeInvalide},
		{"motif inconnu", func(r int) string { return corpsDecision(r, decisionRefuser, motif("trop_cher")) }, codeInvalide},
		{"motif vide", func(r int) string { return corpsDecision(r, decisionRefuser, motif("")) }, codeInvalide},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			admin := b.administrateur()
			v := b.nouveauVendeur()
			article := b.aVerifier(t, v, titreInitial)

			p := b.decider(t, admin.jeton, article.ID, c.corps(article.Revision)).probleme(t, http.StatusUnprocessableEntity)

			exigerTypeDeProbleme(t, p, typeSaisieInvalide)
			exigerErreur(t, p, "motif", c.code)
			b.exigerSansEffet(t, admin, v, article)
		})
	}
}

func TestUS022SaisieInvalideRefuseeEn400SansEffet(t *testing.T) {
	cas := map[string]func(revision int) string{
		"révision absente":     func(int) string { return `{"decision":"publier"}` },
		"révision nulle":       func(int) string { return `{"revision":0,"decision":"publier"}` },
		"révision négative":    func(int) string { return `{"revision":-1,"decision":"publier"}` },
		"révision en texte":    func(r int) string { return fmt.Sprintf(`{"revision":"%d","decision":"publier"}`, r) },
		"décision absente":     func(r int) string { return fmt.Sprintf(`{"revision":%d}`, r) },
		"décision inconnue":    func(r int) string { return fmt.Sprintf(`{"revision":%d,"decision":"supprimer"}`, r) },
		"champ inconnu":        func(r int) string { return fmt.Sprintf(`{"revision":%d,"decision":"publier","etat":"publie"}`, r) },
		"clé dupliquée":        func(r int) string { return fmt.Sprintf(`{"revision":%d,"decision":"refuser","decision":"publier"}`, r) },
		"JSON mal formé":       func(r int) string { return fmt.Sprintf(`{"revision":%d,"decision":"publier"`, r) },
		"corps vide":           func(int) string { return `` },
		"tableau au lieu d'un": func(r int) string { return fmt.Sprintf(`[{"revision":%d,"decision":"publier"}]`, r) },
	}
	for nom, corps := range cas {
		t.Run(nom, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			admin := b.administrateur()
			v := b.nouveauVendeur()
			article := b.aVerifier(t, v, titreInitial)

			b.decider(t, admin.jeton, article.ID, corps(article.Revision)).probleme(t, http.StatusBadRequest)

			b.exigerSansEffet(t, admin, v, article)
		})
	}
}

func TestUS022CorpsHorsFormatRefuseSansEffet(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	admin := b.administrateur()
	v := b.nouveauVendeur()
	article := b.aVerifier(t, v, titreInitial)
	valide := corpsDecision(article.Revision, decisionPublier, nil)

	b.envoyer(t, http.MethodPost, cheminDecision(article.ID), admin.jeton, corpsJSON(valide), typeDeContenu("text/plain")).probleme(t, http.StatusUnsupportedMediaType)
	b.envoyer(t, http.MethodPost, cheminDecision(article.ID), admin.jeton, corpsBrut([]byte(valide+strings.Repeat(" ", limiteCorps)))).probleme(t, http.StatusRequestEntityTooLarge)

	b.exigerSansEffet(t, admin, v, article)
}

func TestUS022AC4DecisionAppliqueeJournalisee(t *testing.T) {
	decisions := []struct {
		decision string
		motif    *string
	}{{decisionPublier, nil}, {decisionRefuser, motif("coordonnees_detectees")}}
	for _, d := range decisions {
		t.Run(d.decision, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			admin := b.administrateur()
			article := b.aVerifier(t, b.nouveauVendeur(), titreInitial)

			b.deciderAvecSucces(t, admin.jeton, article.ID, article.Revision, d.decision, d.motif)

			exigerEvenementDeDecision(t, b.journalHTTP, admin, article, d.decision, d.motif)
		})
	}
}

func TestUS022AC5VendeurEtAcheteurInterditsSurLesDeuxRoutes(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	admin := b.administrateur()
	v := b.nouveauVendeur()
	article := b.aVerifier(t, v, titreInitial)
	sujet := uuid.NewString()
	jetons := map[string]string{
		"vendeur":               bearer(b.jetons.vendeur(sujet)),
		"vendeur de l'article":  v.jeton,
		"acheteur":              bearer(b.jetons.acheteur(sujet)),
		"vendeur et acheteur":   bearer(b.jetons.emettre(sujet, []string{roleVendeur, roleAcheteur})),
		"sans aucun rôle":       bearer(b.jetons.emettre(sujet, nil)),
		"rôle voisin d'admin":   bearer(b.jetons.emettre(sujet, []string{"admins", "Admin", "administrateur"})),
		"rôle admin hors realm": bearer(b.jetons.emettre(sujet, nil, avecRevendication("resource_access", map[string]any{clientAutorise: map[string]any{"roles": []string{roleAdmin}}}))),
	}
	for nom, jeton := range jetons {
		t.Run(nom+" / liste", func(t *testing.T) {
			b.envoyer(t, http.MethodGet, cheminAdmin, jeton).probleme(t, http.StatusForbidden)
		})
		t.Run(nom+" / décision", func(t *testing.T) {
			b.decider(t, jeton, article.ID, corpsDecision(article.Revision, decisionPublier, nil)).probleme(t, http.StatusForbidden)
		})
	}
	b.exigerSansEffet(t, admin, v, article)
	if n := len(b.journalHTTP.evenements(t, evenementDecisionAdmin)); n != 0 {
		t.Fatalf("%d décision(s) journalisée(s) pour des appels interdits", n)
	}
}

func TestUS022AC5SansJetonOuJetonInvalideNonAuthentifie(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	admin := b.administrateur()
	v := b.nouveauVendeur()
	article := b.aVerifier(t, v, titreInitial)
	sujet := uuid.NewString()
	jetons := map[string]string{
		"sans jeton":             "",
		"jeton admin expiré":     bearer(b.jetons.emettre(sujet, []string{roleAdmin}, avecRevendication("exp", 1))),
		"jeton admin d'un autre": bearer(b.jetons.emettre(sujet, []string{roleAdmin}, avecRevendication("iss", "https://intrus.test/realms/collector"))),
	}
	for nom, jeton := range jetons {
		for _, r := range []reponse{
			b.envoyer(t, http.MethodGet, cheminAdmin, jeton),
			b.decider(t, jeton, article.ID, corpsDecision(article.Revision, decisionPublier, nil)),
		} {
			t.Run(nom, func(t *testing.T) {
				r.probleme(t, http.StatusUnauthorized)
				if r.entetes.Get("WWW-Authenticate") == "" {
					t.Error("en-tête WWW-Authenticate absent")
				}
			})
		}
	}
	b.exigerSansEffet(t, admin, v, article)
}

func TestUS022AC5AdminNAccedePasAuxRoutesDuVendeur(t *testing.T) {
	b := demarrerBancEnMemoire(t)

	b.envoyer(t, http.MethodGet, cheminArticles, b.administrateur().jeton).probleme(t, http.StatusForbidden)
}

func TestUS022ReponsesAdministrationConformesAuContratOpenAPI(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	routeur := chargerContrat(t, b.adresse)
	jouerLesStatutsDeLAdministration(t, b)
	for _, ech := range b.historique() {
		verifierEchange(t, routeur, ech)
	}
}

func jouerLesStatutsDeLAdministration(t *testing.T, b *bancEnMemoire) {
	t.Helper()
	admin := b.administrateur()
	v := b.nouveauVendeur()
	aPublier := b.aVerifier(t, v, "Contrat à publier")
	aRefuser := b.aVerifier(t, v, "Contrat à refuser")
	b.listeAVerifier(t, admin.jeton, "")
	b.listeAVerifier(t, admin.jeton, "?page=9&taille=1")
	b.envoyer(t, http.MethodGet, cheminAdmin+"?taille=51", admin.jeton).exiger(t, http.StatusBadRequest)
	b.envoyer(t, http.MethodGet, cheminAdmin, "").exiger(t, http.StatusUnauthorized)
	b.envoyer(t, http.MethodGet, cheminAdmin, v.jeton).exiger(t, http.StatusForbidden)
	b.decider(t, admin.jeton, aRefuser.ID, corpsDecision(aRefuser.Revision, decisionRefuser, nil)).exiger(t, http.StatusUnprocessableEntity)
	b.decider(t, admin.jeton, aRefuser.ID, `{"revision":0,"decision":"publier"}`).exiger(t, http.StatusBadRequest)
	b.decider(t, admin.jeton, aPublier.ID, corpsDecision(aPublier.Revision+1, decisionPublier, nil)).exiger(t, http.StatusConflict)
	b.decider(t, admin.jeton, uuid.NewString(), corpsDecision(1, decisionPublier, nil)).exiger(t, http.StatusNotFound)
	b.decider(t, "", aPublier.ID, corpsDecision(aPublier.Revision, decisionPublier, nil)).exiger(t, http.StatusUnauthorized)
	b.decider(t, bearer(b.jetons.acheteur(uuid.NewString())), aPublier.ID, corpsDecision(aPublier.Revision, decisionPublier, nil)).exiger(t, http.StatusForbidden)
	b.envoyer(t, http.MethodPost, cheminDecision(aPublier.ID), admin.jeton, corpsJSON(`{}`), typeDeContenu("text/plain")).exiger(t, http.StatusUnsupportedMediaType)
	b.deciderAvecSucces(t, admin.jeton, aPublier.ID, aPublier.Revision, decisionPublier, nil)
	b.deciderAvecSucces(t, admin.jeton, aRefuser.ID, aRefuser.Revision, decisionRefuser, motif("vendeur_non_fiable"))
	b.decider(t, admin.jeton, aPublier.ID, corpsDecision(aPublier.Revision, decisionPublier, nil)).exiger(t, http.StatusConflict)
	b.relire(t, v.jeton, aRefuser.ID)
}
