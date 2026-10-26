package main

import (
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

type scenario struct {
	nom   string
	jouer func(*testing.T, *environnement)
}

func scenariosMetier() []scenario {
	return []scenario{
		{"US-009-AC1 brouillon créé en révision 1 et listé", ac1BrouillonCreeEnRevision1EtListe},
		{"US-009-AC1 brouillon vide accepté", ac1BrouillonVideAccepte},
		{"US-009-AC1 bornes maximales acceptées", ac1BornesAcceptees},
		{"US-009-AC1 liste paginée du vendeur", ac1ListePagineeDuVendeur},
		{"US-009-AC2 modification enregistrée et révision augmentée", ac2ModificationAugmenteLaRevision},
		{"US-009-AC2 remplacement complet du contenu", ac2RemplacementComplet},
		{"US-009-AC2 révision périmée refusée en 409", ac2RevisionPerimeeRefusee},
		{"US-009-AC2 modifications concurrentes : une seule gagnante", ac2ModificationsConcurrentes},
		{"US-009-AC2 révision absente refusée", ac2RevisionAbsenteRefusee},
		{"US-009-AC2 modification invalide sans effet", ac2ModificationInvalideSansEffet},
		{"US-009-AC2 article inexistant introuvable", ac2ArticleInexistantIntrouvable},
		{"US-009-AC3 catégories publiques et bien formées", ac3CategoriesPubliques},
		{"US-009-AC3 chaque catégorie proposée est acceptée", ac3ChaqueCategorieProposeeAcceptee},
		{"US-009-AC3 catégorie hors jeu de données refusée", ac3CategorieHorsJeuRefusee},
		{"US-009-AC4 saisies invalides refusées en 400", ac4SaisiesInvalidesRefusees},
		{"US-009-AC4 plusieurs erreurs signalées ensemble", ac4PlusieursErreursSignalees},
		{"US-009-AC7 article d'un autre vendeur introuvable", ac7ArticleDUnAutreVendeurIntrouvable},
		{"US-009-AC7 liste limitée aux articles du vendeur", ac7ListeLimiteeAuVendeur},
	}
}

func corpsComplet(titre, categorie string, prix, frais int64) string {
	return fmt.Sprintf(`{"titre":%q,"description":"Belle pièce\nen bon état","categorieId":%q,"prix":{"montantCentimes":%d,"devise":"EUR"},"fraisDePort":{"montantCentimes":%d,"devise":"EUR"}}`, titre, categorie, prix, frais)
}

func corpsRevision(revision int, contenu string) string {
	return fmt.Sprintf(`{"revision":%d,%s`, revision, strings.TrimPrefix(contenu, "{"))
}

func premiereCategorie(t *testing.T, e *environnement) categorieRecue {
	t.Helper()
	return e.categories(t)[0]
}

func ac1BrouillonCreeEnRevision1EtListe(t *testing.T, e *environnement) {
	jeton := e.vendeur()
	categorie := premiereCategorie(t, e)
	reponseCreation := e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsJSON(corpsComplet("Pièce de 5 francs 1960", categorie.ID, 2500, 450)))
	var cree articleRecu
	reponseCreation.exiger(t, http.StatusCreated).lire(t, &cree)
	if cree.Etat != "brouillon" || cree.Revision != 1 {
		t.Fatalf("état %q révision %d, attendu brouillon révision 1", cree.Etat, cree.Revision)
	}
	verifierContenuComplet(t, cree, categorie)
	if !strings.HasSuffix(reponseCreation.entetes.Get("Location"), "/vendeur/articles/"+cree.ID) {
		t.Errorf("Location %q ne désigne pas le brouillon", reponseCreation.entetes.Get("Location"))
	}
	if relu := e.relire(t, jeton, cree.ID); !reflect.DeepEqual(relu, cree) {
		t.Errorf("relecture %+v différente de la création %+v", relu, cree)
	}
	page := e.mesArticles(t, jeton, "")
	if page.Total != 1 || len(page.Elements) != 1 || page.Elements[0].ID != cree.ID {
		t.Fatalf("« Mes articles » ne contient pas le brouillon : %+v", page)
	}
}

func verifierContenuComplet(t *testing.T, a articleRecu, categorie categorieRecue) {
	t.Helper()
	if a.Titre != "Pièce de 5 francs 1960" || a.Description != "Belle pièce\nen bon état" {
		t.Errorf("titre ou description non enregistrés : %+v", a)
	}
	if a.Categorie == nil || *a.Categorie != categorie {
		t.Errorf("catégorie %+v, attendue %+v", a.Categorie, categorie)
	}
	if a.Prix == nil || *a.Prix != (montant{2500, "EUR"}) || a.FraisDePort == nil || *a.FraisDePort != (montant{450, "EUR"}) {
		t.Errorf("prix ou frais de port non enregistrés : %+v %+v", a.Prix, a.FraisDePort)
	}
	if a.CreeLe == "" || a.ModifieLe == "" {
		t.Errorf("dates absentes : %+v", a)
	}
}

func ac1BrouillonVideAccepte(t *testing.T, e *environnement) {
	cree := e.creer(t, e.vendeur(), `{}`)
	if cree.Etat != "brouillon" || cree.Revision != 1 || cree.Titre != "" || cree.Prix != nil || cree.Categorie != nil {
		t.Fatalf("brouillon vide inattendu : %+v", cree)
	}
}

func ac1BornesAcceptees(t *testing.T, e *environnement) {
	jeton := e.vendeur()
	cas := map[string]string{
		"titre de 200 caractères":           fmt.Sprintf(`{"titre":%q}`, strings.Repeat("a", 200)),
		"titre de 200 caractères accentués": fmt.Sprintf(`{"titre":%q}`, strings.Repeat("é", 200)),
		"description de 5000 caractères":    fmt.Sprintf(`{"description":%q}`, strings.Repeat("d", 5000)),
		"prix nul":                          `{"prix":{"montantCentimes":0,"devise":"EUR"}}`,
		"prix au plafond":                   `{"prix":{"montantCentimes":100000000,"devise":"EUR"}}`,
		"frais de port nuls":                `{"fraisDePort":{"montantCentimes":0,"devise":"EUR"}}`,
		"frais de port au plafond":          `{"fraisDePort":{"montantCentimes":100000,"devise":"EUR"}}`,
	}
	for nom, corps := range cas {
		t.Run(nom, func(t *testing.T) {
			e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsJSON(corps)).exiger(t, http.StatusCreated)
		})
	}
}

func ac1ListePagineeDuVendeur(t *testing.T, e *environnement) {
	jeton := e.vendeur()
	for i := range 3 {
		e.creer(t, jeton, fmt.Sprintf(`{"titre":"Article %d"}`, i))
	}
	premiere := e.mesArticles(t, jeton, "?page=1&taille=2")
	seconde := e.mesArticles(t, jeton, "?page=2&taille=2")
	if premiere.Total != 3 || len(premiere.Elements) != 2 || seconde.Total != 3 || len(seconde.Elements) != 1 {
		t.Fatalf("pagination inattendue : %+v / %+v", premiere, seconde)
	}
	if premiere.Elements[0].Titre != "Article 2" {
		t.Errorf("le plus récemment modifié devrait venir en premier : %+v", premiere.Elements)
	}
}

func ac2ModificationAugmenteLaRevision(t *testing.T, e *environnement) {
	jeton := e.vendeur()
	categorie := premiereCategorie(t, e)
	cree := e.creer(t, jeton, `{"titre":"Avant"}`)
	var modifie articleRecu
	e.modifier(t, jeton, cree.ID, corpsRevision(1, corpsComplet("Pièce de 5 francs 1960", categorie.ID, 2500, 450))).exiger(t, http.StatusOK).lire(t, &modifie)
	if modifie.Revision != 2 || modifie.ID != cree.ID || modifie.Etat != "brouillon" {
		t.Fatalf("révision %d état %q après modification, attendu 2 brouillon", modifie.Revision, modifie.Etat)
	}
	verifierContenuComplet(t, e.relire(t, jeton, cree.ID), categorie)
	var encore articleRecu
	e.modifier(t, jeton, cree.ID, `{"revision":2,"titre":"Encore"}`).exiger(t, http.StatusOK).lire(t, &encore)
	if encore.Revision != 3 || encore.Titre != "Encore" {
		t.Fatalf("seconde modification : %+v", encore)
	}
}

func ac2RemplacementComplet(t *testing.T, e *environnement) {
	jeton := e.vendeur()
	cree := e.creer(t, jeton, corpsComplet("Complet", premiereCategorie(t, e).ID, 100, 10))
	var modifie articleRecu
	e.modifier(t, jeton, cree.ID, `{"revision":1,"titre":"Seul"}`).exiger(t, http.StatusOK).lire(t, &modifie)
	if modifie.Titre != "Seul" || modifie.Description != "" || modifie.Categorie != nil || modifie.Prix != nil || modifie.FraisDePort != nil {
		t.Fatalf("les champs absents devraient être vidés : %+v", modifie)
	}
}

func ac2RevisionPerimeeRefusee(t *testing.T, e *environnement) {
	jeton := e.vendeur()
	cree := e.creer(t, jeton, `{"titre":"Original"}`)
	e.modifier(t, jeton, cree.ID, `{"revision":1,"titre":"Gagnant"}`).exiger(t, http.StatusOK)
	e.modifier(t, jeton, cree.ID, `{"revision":1,"titre":"Perdant"}`).probleme(t, http.StatusConflict)
	e.modifier(t, jeton, cree.ID, `{"revision":7,"titre":"Futur"}`).probleme(t, http.StatusConflict)
	relu := e.relire(t, jeton, cree.ID)
	if relu.Revision != 2 || relu.Titre != "Gagnant" {
		t.Fatalf("la révision périmée a modifié l'article : %+v", relu)
	}
}

func ac2ModificationsConcurrentes(t *testing.T, e *environnement) {
	jeton := e.vendeur()
	cree := e.creer(t, jeton, `{"titre":"Disputé"}`)
	statuts := make([]int, 8)
	var attente sync.WaitGroup
	for i := range statuts {
		attente.Go(func() {
			statuts[i] = e.modifier(t, jeton, cree.ID, fmt.Sprintf(`{"revision":1,"titre":"Version %d"}`, i)).statut
		})
	}
	attente.Wait()
	gagnants := 0
	for _, s := range statuts {
		if s == http.StatusOK {
			gagnants++
		} else if s != http.StatusConflict {
			t.Errorf("statut %d inattendu", s)
		}
	}
	if gagnants != 1 || e.relire(t, jeton, cree.ID).Revision != 2 {
		t.Fatalf("une seule modification devait l'emporter : %v", statuts)
	}
}

func ac2RevisionAbsenteRefusee(t *testing.T, e *environnement) {
	jeton := e.vendeur()
	cree := e.creer(t, jeton, `{"titre":"Sans révision"}`)
	e.modifier(t, jeton, cree.ID, `{"titre":"Modifié"}`).probleme(t, http.StatusBadRequest)
	e.modifier(t, jeton, cree.ID, `{"revision":0,"titre":"Modifié"}`).probleme(t, http.StatusBadRequest)
}

func ac2ModificationInvalideSansEffet(t *testing.T, e *environnement) {
	jeton := e.vendeur()
	cree := e.creer(t, jeton, `{"titre":"Intact"}`)
	p := e.modifier(t, jeton, cree.ID, `{"revision":1,"titre":"Changé","prix":{"montantCentimes":-1,"devise":"EUR"}}`).probleme(t, http.StatusBadRequest)
	exigerErreur(t, p, "prix", "negatif")
	if relu := e.relire(t, jeton, cree.ID); relu.Revision != 1 || relu.Titre != "Intact" {
		t.Fatalf("une saisie invalide a modifié l'article : %+v", relu)
	}
}

func ac2ArticleInexistantIntrouvable(t *testing.T, e *environnement) {
	e.modifier(t, e.vendeur(), uuid.NewString(), `{"revision":1,"titre":"Fantôme"}`).probleme(t, http.StatusNotFound)
}

func ac3CategoriesPubliques(t *testing.T, e *environnement) {
	categories := e.categories(t)
	vus := map[string]bool{}
	for _, c := range categories {
		if c.ID == "" || c.Libelle == "" || vus[c.ID] || strings.ToLower(c.ID) != c.ID {
			t.Errorf("catégorie mal formée ou dupliquée : %+v", c)
		}
		vus[c.ID] = true
	}
}

func ac3ChaqueCategorieProposeeAcceptee(t *testing.T, e *environnement) {
	jeton := e.vendeur()
	for _, c := range e.categories(t) {
		cree := e.creer(t, jeton, fmt.Sprintf(`{"categorieId":%q}`, c.ID))
		if cree.Categorie == nil || *cree.Categorie != c {
			t.Errorf("catégorie %+v enregistrée en %+v", c, cree.Categorie)
		}
	}
}

func ac3CategorieHorsJeuRefusee(t *testing.T, e *environnement) {
	ids := []string{}
	for _, c := range e.categories(t) {
		ids = append(ids, c.ID)
	}
	if slices.Contains(ids, "categorie-absente-du-jeu") {
		t.Fatal("identifiant de test présent dans le jeu de données")
	}
	p := e.envoyer(t, http.MethodPost, cheminArticles, e.vendeur(), corpsJSON(`{"categorieId":"categorie-absente-du-jeu"}`)).probleme(t, http.StatusBadRequest)
	exigerErreur(t, p, "categorieId", "categorie_inconnue")
}

type saisieInvalide struct {
	nom   string
	corps string
	champ string
	code  string
}

func saisiesInvalides() []saisieInvalide {
	return []saisieInvalide{
		{"prix négatif", `{"prix":{"montantCentimes":-1,"devise":"EUR"}}`, "prix", "negatif"},
		{"prix au-delà du plafond", `{"prix":{"montantCentimes":100000001,"devise":"EUR"}}`, "prix", "au_dela_du_plafond"},
		{"titre au-delà de 200 caractères", fmt.Sprintf(`{"titre":%q}`, strings.Repeat("a", 201)), "titre", "trop_long"},
		{"titre accentué au-delà de 200 caractères", fmt.Sprintf(`{"titre":%q}`, strings.Repeat("é", 201)), "titre", "trop_long"},
		{"catégorie inexistante", `{"categorieId":"inexistante"}`, "categorieId", "categorie_inconnue"},
		{"catégorie mal formée", `{"categorieId":"Pas Valide"}`, "categorieId", ""},
		{"frais de port négatifs", `{"fraisDePort":{"montantCentimes":-5,"devise":"EUR"}}`, "fraisDePort", "negatif"},
		{"frais de port au-delà du plafond", `{"fraisDePort":{"montantCentimes":100001,"devise":"EUR"}}`, "fraisDePort", "au_dela_du_plafond"},
		{"devise non prise en charge", `{"prix":{"montantCentimes":100,"devise":"USD"}}`, "prix", "devise_non_prise_en_charge"},
		{"description au-delà de 5000 caractères", fmt.Sprintf(`{"description":%q}`, strings.Repeat("d", 5001)), "description", "trop_long"},
		{"titre avec caractère de contrôle", `{"titre":"cloche\u0007"}`, "titre", "caractere_interdit"},
	}
}

func ac4SaisiesInvalidesRefusees(t *testing.T, e *environnement) {
	jeton := e.vendeur()
	for _, cas := range saisiesInvalides() {
		t.Run(cas.nom, func(t *testing.T) {
			p := e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsJSON(cas.corps)).probleme(t, http.StatusBadRequest)
			exigerErreur(t, p, cas.champ, cas.code)
		})
	}
	if page := e.mesArticles(t, jeton, ""); page.Total != 0 {
		t.Fatalf("une saisie invalide a créé un article : %+v", page)
	}
}

func ac4PlusieursErreursSignalees(t *testing.T, e *environnement) {
	corps := fmt.Sprintf(`{"titre":%q,"prix":{"montantCentimes":-1,"devise":"EUR"}}`, strings.Repeat("a", 201))
	p := e.envoyer(t, http.MethodPost, cheminArticles, e.vendeur(), corpsJSON(corps)).probleme(t, http.StatusBadRequest)
	exigerErreur(t, p, "titre", "trop_long")
	exigerErreur(t, p, "prix", "negatif")
}

func exigerErreur(t *testing.T, p probleme, champ, code string) {
	t.Helper()
	for _, erreur := range p.Erreurs {
		if strings.HasPrefix(erreur.Champ, champ) && (code == "" || erreur.Code == code) {
			return
		}
	}
	t.Errorf("erreur %s/%s absente du Problem Details : %+v", champ, code, p.Erreurs)
}

func ac7ArticleDUnAutreVendeurIntrouvable(t *testing.T, e *environnement) {
	vendeurA, vendeurB := e.vendeur(), e.vendeur()
	article := e.creer(t, vendeurA, `{"titre":"Propriété de A"}`)
	inexistant := e.lireArticle(t, vendeurB, uuid.NewString()).probleme(t, http.StatusNotFound)
	lecture := e.lireArticle(t, vendeurB, article.ID).probleme(t, http.StatusNotFound)
	modification := e.modifier(t, vendeurB, article.ID, `{"revision":1,"titre":"Volé"}`).probleme(t, http.StatusNotFound)
	modificationInexistant := e.modifier(t, vendeurB, uuid.NewString(), `{"revision":1,"titre":"Volé"}`).probleme(t, http.StatusNotFound)
	if !reflect.DeepEqual(lecture, inexistant) || !reflect.DeepEqual(modification, modificationInexistant) {
		t.Errorf("la réponse révèle l'existence de l'article : %+v / %+v", lecture, modification)
	}
	if relu := e.relire(t, vendeurA, article.ID); relu.Revision != 1 || relu.Titre != "Propriété de A" {
		t.Fatalf("le vendeur B a modifié l'article de A : %+v", relu)
	}
}

func ac7ListeLimiteeAuVendeur(t *testing.T, e *environnement) {
	vendeurA, vendeurB := e.vendeur(), e.vendeur()
	e.creer(t, vendeurA, `{"titre":"A1"}`)
	e.creer(t, vendeurA, `{"titre":"A2"}`)
	e.creer(t, vendeurB, `{"titre":"B1"}`)
	pageB := e.mesArticles(t, vendeurB, "")
	if pageB.Total != 1 || len(pageB.Elements) != 1 || pageB.Elements[0].Titre != "B1" {
		t.Fatalf("la liste de B contient des articles de A : %+v", pageB)
	}
	if pageA := e.mesArticles(t, vendeurA, ""); pageA.Total != 2 {
		t.Fatalf("la liste de A est incomplète : %+v", pageA)
	}
}
