package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"collector/libs/platform/broker"
)

const (
	cheminCatalogue      = "/marche/v1/articles"
	tailleMaximale       = 50
	tailleParDefaut      = 20
	codeInvalide         = "invalide"
	codeHorsLimites      = "hors_limites"
	codeCategorieInconnu = "categorie_inconnue"
)

var (
	clesDUneEntree  = []string{"id", "titre", "prix", "categorie"}
	clesDUneFiche   = []string{"id", "titre", "prix", "categorie", "description", "fraisDePort"}
	etatsNonPublies = []string{etatBrouillon, etatEnControle, etatRefuse, etatAVerifier}
)

type entreeCatalogue struct {
	ID        string          `json:"id"`
	Titre     string          `json:"titre"`
	Prix      *montant        `json:"prix"`
	Categorie *categorieRecue `json:"categorie"`
}

type pageCatalogue struct {
	Elements []entreeCatalogue `json:"elements"`
	Page     int               `json:"page"`
	Taille   int               `json:"taille"`
	Total    int               `json:"total"`
}

type fichePublique struct {
	ID          string          `json:"id"`
	Titre       string          `json:"titre"`
	Description string          `json:"description"`
	Prix        *montant        `json:"prix"`
	FraisDePort *montant        `json:"fraisDePort"`
	Categorie   *categorieRecue `json:"categorie"`
}

type categoriesDuBanc struct {
	filtree    string
	autre      string
	sansPublie string
}

func (e *environnement) categoriesDuBanc(t *testing.T) categoriesDuBanc {
	t.Helper()
	proposees := e.categories(t)
	if len(proposees) < 3 {
		t.Fatalf("%d catégorie(s) proposée(s), 3 au moins attendues", len(proposees))
	}
	return categoriesDuBanc{filtree: proposees[0].ID, autre: proposees[1].ID, sansPublie: proposees[2].ID}
}

type vendeurConnu struct {
	sujet string
	jeton string
}

type scenarioCatalogue struct {
	nom   string
	jouer func(*testing.T, *bancEnMemoire)
}

func (b *bancEnMemoire) nouveauVendeur() vendeurConnu {
	sujet := uuid.NewString()
	return vendeurConnu{sujet: sujet, jeton: bearer(b.jetons.vendeur(sujet))}
}

func (b *bancEnMemoire) trancher(t *testing.T, soumis articleRecu, issue string, motif *string) {
	t.Helper()
	livree := b.livrer(t, b.resultatSigne(t, resultatEmis{ArticleID: soumis.ID, Revision: soumis.Revision, RulesVersion: versionDesRegles, Issue: issue, Reason: motif}))
	if livree.disposition != broker.Ack {
		t.Fatalf("issue %s de l'article %s non acquittée", issue, soumis.ID)
	}
}

func motif(code string) *string {
	return &code
}

func (b *bancEnMemoire) amenerDansLEtat(t *testing.T, jeton string, article articleRecu, etat string) articleRecu {
	t.Helper()
	if etat == etatBrouillon {
		return article
	}
	soumis := b.soumettreAvecSucces(t, jeton, article)
	switch etat {
	case etatPublie:
		b.trancher(t, soumis, issuePublie, nil)
	case etatRefuse:
		b.trancher(t, soumis, issueRefuse, motif(motifIncomplet))
	case etatAVerifier:
		b.trancher(t, soumis, issueAVerifier, motif(motifCategorieInconnu))
	}
	relu := b.relire(t, jeton, article.ID)
	if relu.Etat != etat {
		t.Fatalf("article %s en état %q, %q attendu", article.ID, relu.Etat, etat)
	}
	return relu
}

func (b *bancEnMemoire) articleDansLEtat(t *testing.T, jeton, titre, categorie, etat string) articleRecu {
	t.Helper()
	return b.amenerDansLEtat(t, jeton, b.creer(t, jeton, corpsComplet(titre, categorie, 1500, 400)), etat)
}

func (b *bancEnMemoire) publier(t *testing.T, jeton, titre, categorie string) articleRecu {
	t.Helper()
	return b.articleDansLEtat(t, jeton, titre, categorie, etatPublie)
}

func (b *bancEnMemoire) nonPublies(t *testing.T, jeton, categorie string) []articleRecu {
	t.Helper()
	var articles []articleRecu
	for _, etat := range etatsNonPublies {
		articles = append(articles, b.articleDansLEtat(t, jeton, "Non publié "+etat, categorie, etat))
	}
	return articles
}

func (e *environnement) consulterCatalogue(t *testing.T, requete string) reponse {
	t.Helper()
	return e.envoyer(t, http.MethodGet, cheminCatalogue+requete, "")
}

func (e *environnement) pageDuCatalogue(t *testing.T, requete string) pageCatalogue {
	t.Helper()
	var page pageCatalogue
	e.consulterCatalogue(t, requete).exiger(t, http.StatusOK).lire(t, &page)
	return page
}

func filtre(categorie string) string {
	if categorie == "" {
		return ""
	}
	return "&categorie=" + categorie
}

func (e *environnement) toutLeCatalogue(t *testing.T, categorie string) []entreeCatalogue {
	t.Helper()
	premiere := e.pageDuCatalogue(t, fmt.Sprintf("?page=1&taille=%d%s", tailleMaximale, filtre(categorie)))
	entrees := premiere.Elements
	for numero := 2; len(entrees) < premiere.Total; numero++ {
		page := e.pageDuCatalogue(t, fmt.Sprintf("?page=%d&taille=%d%s", numero, tailleMaximale, filtre(categorie)))
		if page.Total != premiere.Total || len(page.Elements) == 0 {
			t.Fatalf("page %d : total %d et %d élément(s), total %d annoncé en page 1", numero, page.Total, len(page.Elements), premiere.Total)
		}
		entrees = append(entrees, page.Elements...)
	}
	if len(entrees) != premiere.Total {
		t.Fatalf("%d entrée(s) parcourue(s), total annoncé %d", len(entrees), premiere.Total)
	}
	return entrees
}

func identifiants(entrees []entreeCatalogue) []string {
	ids := make([]string, 0, len(entrees))
	for _, e := range entrees {
		ids = append(ids, e.ID)
	}
	return ids
}

func exigerPresents(t *testing.T, entrees []entreeCatalogue, articles ...articleRecu) {
	t.Helper()
	ids := identifiants(entrees)
	for _, a := range articles {
		if !slices.Contains(ids, a.ID) {
			t.Errorf("article publié %s (%q) absent du catalogue", a.ID, a.Titre)
		}
	}
}

func exigerAbsents(t *testing.T, entrees []entreeCatalogue, articles ...articleRecu) {
	t.Helper()
	ids := identifiants(entrees)
	for _, a := range articles {
		if slices.Contains(ids, a.ID) {
			t.Errorf("article %s en état %q visible au catalogue", a.ID, a.Etat)
		}
	}
}

func exigerSansDoublon(t *testing.T, entrees []entreeCatalogue) {
	t.Helper()
	ids := identifiants(entrees)
	slices.Sort(ids)
	if len(slices.Compact(ids)) != len(entrees) {
		t.Fatalf("doublons dans le catalogue : %v", identifiants(entrees))
	}
}

func (e *environnement) fiche(t *testing.T, id string) reponse {
	t.Helper()
	return e.envoyer(t, http.MethodGet, cheminCatalogue+"/"+id, "")
}

func (e *environnement) lireFiche(t *testing.T, id string) fichePublique {
	t.Helper()
	var f fichePublique
	e.fiche(t, id).exiger(t, http.StatusOK).lire(t, &f)
	return f
}

func scenariosDuCatalogue() []scenarioCatalogue {
	return []scenarioCatalogue{
		{"US-023-AC1 seuls les articles publiés de la catégorie choisie", us023Ac1FiltreParCategorie},
		{"US-023-AC1 catégorie sans article publié : liste vide", us023Ac1CategorieSansArticlePublie},
		{"US-023-AC1 du plus récemment publié au plus ancien", us023Ac1TriDuPlusRecemmentPublie},
		{"US-023-AC1 pagination cohérente avec le total", us023Ac1PaginationCoherente},
		{"US-023-AC1 page au-delà du total vide", us023Ac1PageAuDelaDuTotalVide},
		{"US-023-AC1 page 1 de 20 articles par défaut", us023Ac1ValeursParDefaut},
		{"US-023-AC1 bornes de pagination acceptées", us023Ac1BornesAcceptees},
		{"US-023-AC1 paramètres invalides refusés en 400", us023Ac1ParametresRefuses},
		{"US-023-AC2 fiche publique conforme au contenu publié", us023Ac2FicheConformeAuContenu},
		{"US-023-AC2 fiche sans frais de port", us023Ac2FicheSansFraisDePort},
		{"US-023-AC3 article non publié, inexistant ou mal formé : même 404", us023Ac3NonPublieIntrouvable},
		{"US-010-AC2 aucun article non publié au catalogue", us010Ac2AucunNonPublieAuCatalogue},
		{"US-010-AC2 article publié après refus : absent puis visible", us010Ac2VisibleSeulementUneFoisPublie},
		{"US-010-AC2 aucune donnée du vendeur exposée", us010Ac2AucuneDonneeDuVendeur},
	}
}

func us023Ac1FiltreParCategorie(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	avantFiltree := len(b.toutLeCatalogue(t, c.filtree))
	avantAutre := len(b.toutLeCatalogue(t, c.autre))
	v := b.nouveauVendeur()
	f1 := b.publier(t, v.jeton, "Figurine Tintin", c.filtree)
	f2 := b.publier(t, v.jeton, "Figurine Milou", c.filtree)
	t1 := b.publier(t, v.jeton, "Timbre Marianne", c.autre)
	caches := b.nonPublies(t, v.jeton, c.filtree)

	filtrees := b.toutLeCatalogue(t, c.filtree)
	if len(filtrees) != avantFiltree+2 {
		t.Errorf("%d article(s) en %s, %d attendu(s)", len(filtrees), c.filtree, avantFiltree+2)
	}
	for _, entree := range filtrees {
		if entree.Categorie == nil || entree.Categorie.ID != c.filtree {
			t.Errorf("entrée %s hors de la catégorie %s : %+v", entree.ID, c.filtree, entree.Categorie)
		}
	}
	exigerPresents(t, filtrees, f1, f2)
	exigerAbsents(t, filtrees, append(caches, t1)...)

	autres := b.toutLeCatalogue(t, c.autre)
	if len(autres) != avantAutre+1 {
		t.Errorf("%d article(s) en %s, %d attendu(s)", len(autres), c.autre, avantAutre+1)
	}
	exigerPresents(t, autres, t1)
	exigerAbsents(t, autres, f1, f2)
}

func us023Ac1CategorieSansArticlePublie(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	b.nonPublies(t, b.vendeur(), c.sansPublie)
	page := b.pageDuCatalogue(t, "?categorie="+c.sansPublie)
	if page.Total != 0 || len(page.Elements) != 0 || page.Page != 1 {
		t.Fatalf("catégorie sans article publié : %+v, page 1 vide attendue", page)
	}
}

func position(t *testing.T, entrees []entreeCatalogue, a articleRecu) int {
	t.Helper()
	i := slices.Index(identifiants(entrees), a.ID)
	if i < 0 {
		t.Fatalf("article %s absent du catalogue", a.ID)
	}
	return i
}

func us023Ac1TriDuPlusRecemmentPublie(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	jeton := b.vendeur()
	creeEnPremier := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsComplet("Créé en premier", c.filtree, 100, 0)))
	creeEnSecond := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsComplet("Créé en second", c.filtree, 100, 0)))
	b.trancher(t, creeEnSecond, issuePublie, nil)
	b.trancher(t, creeEnPremier, issuePublie, nil)
	dernier := b.publier(t, jeton, "Publié en dernier", c.autre)

	for _, categorie := range []string{"", c.filtree} {
		entrees := b.toutLeCatalogue(t, categorie)
		if position(t, entrees, creeEnPremier) > position(t, entrees, creeEnSecond) {
			t.Errorf("filtre %q : l'article publié en dernier parmi les deux articles de la catégorie filtrée n'est pas listé avant l'autre", categorie)
		}
	}
	tous := b.toutLeCatalogue(t, "")
	if position(t, tous, dernier) != 0 {
		t.Errorf("le dernier article publié est en position %d, 0 attendue", position(t, tous, dernier))
	}
}

func us023Ac1PaginationCoherente(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	jeton := b.vendeur()
	for i := range 3 {
		b.publier(t, jeton, fmt.Sprintf("Timbre %d", i), c.autre)
	}
	reference := b.toutLeCatalogue(t, c.autre)
	exigerSansDoublon(t, reference)
	var parcourus []entreeCatalogue
	for numero := 1; numero <= len(reference); numero++ {
		page := b.pageDuCatalogue(t, fmt.Sprintf("?categorie=%s&page=%d&taille=1", c.autre, numero))
		if page.Page != numero || page.Taille != 1 || page.Total != len(reference) || len(page.Elements) != 1 {
			t.Fatalf("page %d de taille 1 : %+v, total %d attendu", numero, page, len(reference))
		}
		parcourus = append(parcourus, page.Elements...)
	}
	if !slices.Equal(identifiants(parcourus), identifiants(reference)) {
		t.Fatalf("pages de taille 1 %v différentes de la liste complète %v", identifiants(parcourus), identifiants(reference))
	}
	deux := b.pageDuCatalogue(t, fmt.Sprintf("?categorie=%s&page=1&taille=2", c.autre))
	if len(deux.Elements) != 2 || deux.Total != len(reference) {
		t.Fatalf("page de taille 2 : %+v", deux)
	}
}

func us023Ac1PageAuDelaDuTotalVide(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	b.publier(t, b.vendeur(), "Seul timbre de la page", c.autre)
	total := len(b.toutLeCatalogue(t, c.autre))
	for _, numero := range []int{total + 1, 10000} {
		page := b.pageDuCatalogue(t, fmt.Sprintf("?categorie=%s&page=%d&taille=1", c.autre, numero))
		if len(page.Elements) != 0 || page.Total != total || page.Page != numero {
			t.Errorf("page %d au-delà du total %d : %+v, page vide attendue", numero, total, page)
		}
	}
}

func us023Ac1ValeursParDefaut(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	b.publier(t, b.vendeur(), "Figurine par défaut", c.filtree)
	page := b.pageDuCatalogue(t, "")
	if page.Page != 1 || page.Taille != tailleParDefaut || len(page.Elements) == 0 || len(page.Elements) > tailleParDefaut {
		t.Fatalf("catalogue sans paramètre : page %d, taille %d, %d élément(s)", page.Page, page.Taille, len(page.Elements))
	}
}

func us023Ac1BornesAcceptees(t *testing.T, b *bancEnMemoire) {
	for _, requete := range []string{"?page=1&taille=1", "?page=10000", "?taille=50", "?categorie=figurines&page=1&taille=50"} {
		t.Run(requete, func(t *testing.T) {
			b.consulterCatalogue(t, requete).exiger(t, http.StatusOK)
		})
	}
}

func us023Ac1ParametresRefuses(t *testing.T, b *bancEnMemoire) {
	cas := []struct {
		requete string
		code    string
	}{
		{"?categorie=inconnue", codeCategorieInconnu},
		{"?categorie=", codeInvalide},
		{"?categorie=figurines&categorie=timbres", codeInvalide},
		{"?categorie=figurines&categorie=figurines", codeInvalide},
		{"?page=0", codeHorsLimites},
		{"?page=-1", codeHorsLimites},
		{"?page=10001", codeHorsLimites},
		{"?taille=0", codeHorsLimites},
		{"?taille=51", codeHorsLimites},
		{"?page=", codeInvalide},
		{"?taille=", codeInvalide},
		{"?page=abc", codeInvalide},
		{"?taille=1.5", codeInvalide},
		{"?page=1&page=2", codeInvalide},
		{"?taille=10&taille=10", codeInvalide},
	}
	for _, c := range cas {
		t.Run(c.requete, func(t *testing.T) {
			p := b.consulterCatalogue(t, c.requete).probleme(t, http.StatusBadRequest)
			if !slices.ContainsFunc(p.Erreurs, func(e erreurDeChamp) bool { return e.Code == c.code }) {
				t.Errorf("code %q attendu dans %+v", c.code, p.Erreurs)
			}
		})
	}
	t.Run("catégorie hors format", func(t *testing.T) {
		b.consulterCatalogue(t, "?categorie=FIGURINES").probleme(t, http.StatusBadRequest)
	})
}

func us023Ac2FicheConformeAuContenu(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	jeton := b.vendeur()
	publie := b.publier(t, jeton, "Figurine Haddock", c.filtree)
	f := b.lireFiche(t, publie.ID)
	attendue := fichePublique{
		ID:          publie.ID,
		Titre:       publie.Titre,
		Description: publie.Description,
		Prix:        publie.Prix,
		FraisDePort: publie.FraisDePort,
		Categorie:   publie.Categorie,
	}
	if attendue.FraisDePort == nil || !reflect.DeepEqual(f, attendue) {
		t.Fatalf("fiche %+v, attendue %+v", f, attendue)
	}
}

func us023Ac2FicheSansFraisDePort(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	jeton := b.vendeur()
	sansFrais := b.creer(t, jeton, fmt.Sprintf(`{"titre":"Sans frais","description":"Rare","categorieId":%q,"prix":{"montantCentimes":100,"devise":"EUR"}}`, c.autre))
	publie := b.amenerDansLEtat(t, jeton, sansFrais, etatPublie)
	var brute map[string]json.RawMessage
	b.fiche(t, publie.ID).exiger(t, http.StatusOK).lire(t, &brute)
	if string(brute["fraisDePort"]) != "null" {
		t.Fatalf("fraisDePort %s, null attendu", brute["fraisDePort"])
	}
}

func us023Ac3NonPublieIntrouvable(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	jeton := b.vendeur()
	reference := b.fiche(t, uuid.NewString()).probleme(t, http.StatusNotFound)
	cibles := map[string]string{"mal formé": "pas-un-identifiant", "identifiant presque valide": strings.Repeat("0", 35)}
	for _, a := range b.nonPublies(t, jeton, c.filtree) {
		cibles[a.Etat] = a.ID
	}
	for nom, id := range cibles {
		t.Run(nom, func(t *testing.T) {
			p := b.fiche(t, id).probleme(t, http.StatusNotFound)
			if p.Type != reference.Type || p.Title != reference.Title || p.Detail != reference.Detail {
				t.Fatalf("404 distinguable de celui d'un article inexistant : %+v / %+v", p, reference)
			}
		})
	}
	b.fiche(t, b.publier(t, jeton, "Témoin publié", c.filtree).ID).exiger(t, http.StatusOK)
}

func us010Ac2AucunNonPublieAuCatalogue(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	avant := len(b.toutLeCatalogue(t, ""))
	caches := b.nonPublies(t, b.vendeur(), c.filtree)
	for _, categorie := range []string{"", c.filtree} {
		entrees := b.toutLeCatalogue(t, categorie)
		exigerAbsents(t, entrees, caches...)
	}
	if apres := len(b.toutLeCatalogue(t, "")); apres != avant {
		t.Fatalf("total du catalogue passé de %d à %d sans publication", avant, apres)
	}
	for _, a := range caches {
		b.fiche(t, a.ID).probleme(t, http.StatusNotFound)
	}
}

func us010Ac2VisibleSeulementUneFoisPublie(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	jeton := b.vendeur()
	refuse := b.articleDansLEtat(t, jeton, "Refusé puis publié", c.filtree, etatRefuse)
	b.fiche(t, refuse.ID).probleme(t, http.StatusNotFound)
	exigerAbsents(t, b.toutLeCatalogue(t, c.filtree), refuse)
	publie := b.amenerDansLEtat(t, jeton, refuse, etatPublie)
	b.fiche(t, publie.ID).exiger(t, http.StatusOK)
	exigerPresents(t, b.toutLeCatalogue(t, c.filtree), publie)
}

func exigerClesAutorisees(t *testing.T, objet map[string]json.RawMessage, autorisees []string, quoi string) {
	t.Helper()
	for cle := range objet {
		if !slices.Contains(autorisees, cle) {
			t.Errorf("%s expose le champ %q", quoi, cle)
		}
	}
}

func sansTraceDuVendeur(t *testing.T, corps []byte, v vendeurConnu, quoi string) {
	t.Helper()
	texte := strings.ToLower(string(corps))
	for _, interdit := range []string{strings.ToLower(v.sujet), "vendeur", "seller", "etat", "revision", "motif"} {
		if strings.Contains(texte, interdit) {
			t.Errorf("%s contient %q : %s", quoi, interdit, corps)
		}
	}
}

func us010Ac2AucuneDonneeDuVendeur(t *testing.T, b *bancEnMemoire) {
	c := b.categoriesDuBanc(t)
	v := b.nouveauVendeur()
	publie := b.publier(t, v.jeton, "Figurine anonyme", c.filtree)

	liste := b.consulterCatalogue(t, "?categorie="+c.filtree+"&taille=50").exiger(t, http.StatusOK)
	sansTraceDuVendeur(t, liste.corps, v, "la liste du catalogue")
	var page struct {
		Elements []map[string]json.RawMessage `json:"elements"`
	}
	liste.lire(t, &page)
	for _, entree := range page.Elements {
		exigerClesAutorisees(t, entree, clesDUneEntree, "une entrée du catalogue")
	}

	fiche := b.fiche(t, publie.ID).exiger(t, http.StatusOK)
	sansTraceDuVendeur(t, fiche.corps, v, "la fiche")
	var brute map[string]json.RawMessage
	fiche.lire(t, &brute)
	exigerClesAutorisees(t, brute, clesDUneFiche, "la fiche")
}

func TestUS023CatalogueEnMemoire(t *testing.T) {
	for _, s := range scenariosDuCatalogue() {
		t.Run(s.nom, func(t *testing.T) {
			s.jouer(t, demarrerBancEnMemoire(t))
		})
	}
}

func TestUS023ReponsesDuCatalogueConformesAuContratOpenAPI(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	for _, s := range scenariosDuCatalogue() {
		t.Run(s.nom, func(t *testing.T) {
			s.jouer(t, b)
		})
	}
	routeur := chargerContrat(t, b.adresse)
	for _, ech := range b.historique() {
		if strings.HasPrefix(ech.requete.URL.Path, cheminCatalogue) {
			verifierEchange(t, routeur, ech)
		}
	}
}
