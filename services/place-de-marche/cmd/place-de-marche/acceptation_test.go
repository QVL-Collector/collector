package main

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUS009ScenariosMetierEnMemoire(t *testing.T) {
	for _, s := range scenariosMetier() {
		t.Run(s.nom, func(t *testing.T) {
			s.jouer(t, demarrerEnMemoire(t))
		})
	}
}

func TestUS009AC3SeulesLesCategoriesDuJeuFictifSontProposees(t *testing.T) {
	e := demarrerEnMemoire(t)
	proposees := e.categories(t)
	if len(proposees) != len(jeuDeCategoriesFictif) {
		t.Fatalf("%d catégories proposées, %d dans le jeu de données", len(proposees), len(jeuDeCategoriesFictif))
	}
	for i, c := range jeuDeCategoriesFictif {
		if proposees[i] != (categorieRecue{ID: string(c.ID), Libelle: c.Label}) {
			t.Errorf("catégorie %d : %+v, attendue %+v", i, proposees[i], c)
		}
	}
}

type variante struct {
	nom    string
	entete func(f *fabriqueDeJetons, sujet string) string
}

func variantesRefusees() []variante {
	horsRS256 := func(alg string) func(f *fabriqueDeJetons, sujet string) string {
		return func(f *fabriqueDeJetons, sujet string) string {
			entete := map[string]any{"alg": alg, "typ": "JWT", "kid": f.cle.kid}
			charge := map[string]any(revendicationsValides(sujet, roleVendeur))
			if alg == "none" {
				return bearer(f.nonSigne(entete, charge))
			}
			return bearer(f.signerHS256(entete, charge))
		}
	}
	modifie := func(m ...modification) func(f *fabriqueDeJetons, sujet string) string {
		return func(f *fabriqueDeJetons, sujet string) string {
			return bearer(f.emettre(sujet, []string{roleVendeur}, m...))
		}
	}
	fixe := func(valeur string) func(*fabriqueDeJetons, string) string {
		return func(*fabriqueDeJetons, string) string { return valeur }
	}
	return []variante{
		{"sans jeton", fixe("")},
		{"schéma Basic", fixe("Basic dmVuZGV1cjptb3RkZXBhc3Nl")},
		{"Bearer vide", fixe("Bearer ")},
		{"jeton mal formé", fixe("Bearer pas.un.jeton")},
		{"jeton expiré", modifie(avecRevendication("exp", time.Now().Add(-time.Hour).Unix()))},
		{"jeton sans expiration", modifie(sansRevendication("exp"))},
		{"signature d'une autre clé", func(f *fabriqueDeJetons, sujet string) string {
			return bearer(f.signerRS256(f.entete(), revendicationsValides(sujet, roleVendeur), f.intrus))
		}},
		{"signature altérée", func(f *fabriqueDeJetons, sujet string) string {
			jeton := f.vendeur(sujet)
			return bearer(jeton[:len(jeton)-4] + "AAAA")
		}},
		{"clé inconnue", modifie(avecEntete("kid", "cle-inconnue"))},
		{"algorithme none", horsRS256("none")},
		{"algorithme HS256", horsRS256("HS256")},
		{"autre émetteur", modifie(avecRevendication("iss", "https://auth.ailleurs.test/realms/collector"))},
		{"autre audience", modifie(avecRevendication("aud", []string{"transactions"}))},
		{"autre client autorisé", modifie(avecRevendication("azp", "transactions"))},
		{"jeton de type Refresh", modifie(avecRevendication("typ", "Refresh"))},
		{"jeton de type ID", modifie(avecRevendication("typ", "ID"))},
	}
}

type routeVendeur struct {
	nom     string
	methode string
	chemin  func(e *environnement, t *testing.T) string
	corps   string
}

func routesVendeur() []routeVendeur {
	liste := func(*environnement, *testing.T) string { return cheminArticles }
	article := func(e *environnement, t *testing.T) string {
		return cheminArticles + "/" + e.creer(t, e.vendeur(), `{"titre":"Cible"}`).ID
	}
	return []routeVendeur{
		{"lister mes articles", http.MethodGet, liste, ""},
		{"créer un brouillon", http.MethodPost, liste, `{"titre":"Nouveau"}`},
		{"lire un article", http.MethodGet, article, ""},
		{"modifier un brouillon", http.MethodPut, article, `{"revision":1,"titre":"Changé"}`},
	}
}

func (r routeVendeur) appeler(t *testing.T, e *environnement, jeton string) reponse {
	t.Helper()
	chemin := r.chemin(e, t)
	if r.corps == "" {
		return e.envoyer(t, r.methode, chemin, jeton)
	}
	return e.envoyer(t, r.methode, chemin, jeton, corpsJSON(r.corps))
}

func TestUS009AC6JetonsRefusesEn401(t *testing.T) {
	e := demarrerEnMemoire(t)
	for _, route := range routesVendeur() {
		for _, v := range variantesRefusees() {
			t.Run(route.nom+" / "+v.nom, func(t *testing.T) {
				r := route.appeler(t, e, v.entete(e.jetons, uuid.NewString()))
				r.probleme(t, http.StatusUnauthorized)
				if r.entetes.Get("WWW-Authenticate") == "" {
					t.Error("en-tête WWW-Authenticate absent")
				}
			})
		}
	}
}

func TestUS009AC6JetonValideDeVendeurAccepte(t *testing.T) {
	e := demarrerEnMemoire(t)
	attendus := map[string]int{http.MethodGet: http.StatusOK, http.MethodPost: http.StatusCreated, http.MethodPut: http.StatusOK}
	for _, route := range routesVendeur()[:2] {
		t.Run(route.nom, func(t *testing.T) {
			route.appeler(t, e, e.vendeur()).exiger(t, attendus[route.methode])
		})
	}
	t.Run("audience unique sous forme de chaîne", func(t *testing.T) {
		jeton := bearer(e.jetons.emettre(uuid.NewString(), []string{roleVendeur}, avecRevendication("aud", audienceAttendue)))
		e.envoyer(t, http.MethodGet, cheminArticles, jeton).exiger(t, http.StatusOK)
	})
	t.Run("lire et modifier son propre brouillon", func(t *testing.T) {
		jeton := e.vendeur()
		article := e.creer(t, jeton, `{"titre":"Mien"}`)
		e.lireArticle(t, jeton, article.ID).exiger(t, http.StatusOK)
		e.modifier(t, jeton, article.ID, `{"revision":1,"titre":"Toujours mien"}`).exiger(t, http.StatusOK)
	})
}

func TestUS009AC8AcheteurRefuseEn403(t *testing.T) {
	e := demarrerEnMemoire(t)
	sujet := uuid.NewString()
	jetons := map[string]string{
		"acheteur":                bearer(e.jetons.acheteur(sujet)),
		"sans aucun rôle":         bearer(e.jetons.emettre(sujet, nil)),
		"rôle vendeur hors realm": bearer(e.jetons.emettre(sujet, nil, avecRevendication("resource_access", map[string]any{clientAutorise: map[string]any{"roles": []string{roleVendeur}}}))),
		"rôle voisin de vendeur":  bearer(e.jetons.emettre(sujet, []string{"vendeurs", "Vendeur"})),
	}
	for nom, jeton := range jetons {
		for _, route := range routesVendeur() {
			t.Run(nom+" / "+route.nom, func(t *testing.T) {
				route.appeler(t, e, jeton).probleme(t, http.StatusForbidden)
			})
		}
	}
	if page := e.mesArticles(t, bearer(e.jetons.vendeur(sujet)), ""); page.Total != 0 {
		t.Fatalf("un acheteur a créé un article : %+v", page)
	}
}

func TestUS009AC8VisiteurRefuseEn401(t *testing.T) {
	e := demarrerEnMemoire(t)
	e.envoyer(t, http.MethodPost, cheminArticles, "", corpsJSON(`{"titre":"Anonyme"}`)).probleme(t, http.StatusUnauthorized)
}

func TestUS009AC8CategoriesAccessiblesAuVisiteur(t *testing.T) {
	e := demarrerEnMemoire(t)
	e.envoyer(t, http.MethodGet, cheminCategories, "").exiger(t, http.StatusOK)
	e.envoyer(t, http.MethodGet, cheminCategories, bearer(e.jetons.acheteur(uuid.NewString()))).exiger(t, http.StatusOK)
}

const limiteCorps = 16 * 1024

func corpsDeTaille(taille int) []byte {
	debut := []byte(`{"titre":"Rembourré"}`)
	return append(debut, bytes.Repeat([]byte(" "), taille-len(debut))...)
}

func corpsModificationDeTaille(taille int) []byte {
	debut := []byte(`{"revision":1,"titre":"Rembourré"}`)
	return append(debut, bytes.Repeat([]byte(" "), taille-len(debut))...)
}

func TestUS009AC9CorpsRefuses(t *testing.T) {
	e := demarrerEnMemoire(t)
	cas := map[string]string{
		"clé dupliquée":                        `{"titre":"a","titre":"b"}`,
		"clé dupliquée à la casse près":        `{"titre":"a","Titre":"b"}`,
		"clé dupliquée dans un objet imbriqué": `{"prix":{"montantCentimes":1,"montantCentimes":2,"devise":"EUR"}}`,
		"clé dupliquée échappée":               `{"titre":"a","titre":"b"}`,
		"champ inconnu":                        `{"titre":"a","statut":"publie"}`,
		"champ inconnu imbriqué":               `{"prix":{"montantCentimes":1,"devise":"EUR","taux":2}}`,
		"type inattendu":                       `{"prix":{"montantCentimes":"12","devise":"EUR"}}`,
		"racine tableau":                       `[{"titre":"a"}]`,
		"JSON mal formé":                       `{"titre":`,
		"corps vide":                           ``,
	}
	for nom, corps := range cas {
		t.Run(nom, func(t *testing.T) {
			e.envoyer(t, http.MethodPost, cheminArticles, e.vendeur(), corpsJSON(corps)).probleme(t, http.StatusBadRequest)
		})
	}
}

func TestUS009AC9CleDupliqueeRefuseeEnModification(t *testing.T) {
	e := demarrerEnMemoire(t)
	jeton := e.vendeur()
	article := e.creer(t, jeton, `{"titre":"Original"}`)
	e.modifier(t, jeton, article.ID, `{"revision":1,"revision":1,"titre":"x"}`).probleme(t, http.StatusBadRequest)
	e.modifier(t, jeton, article.ID, `{"revision":1,"titre":"x","TITRE":"y"}`).probleme(t, http.StatusBadRequest)
	if relu := e.relire(t, jeton, article.ID); relu.Revision != 1 {
		t.Fatalf("un corps refusé a modifié l'article : %+v", relu)
	}
}

func TestUS009AC9TailleMaximaleDuCorps(t *testing.T) {
	e := demarrerEnMemoire(t)
	jeton := e.vendeur()
	t.Run("exactement 16 Kio accepté", func(t *testing.T) {
		e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsBrut(corpsDeTaille(limiteCorps))).exiger(t, http.StatusCreated)
	})
	t.Run("16 Kio plus un octet refusé en 413", func(t *testing.T) {
		e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsBrut(corpsDeTaille(limiteCorps+1))).probleme(t, http.StatusRequestEntityTooLarge)
	})
	t.Run("1 Mio refusé en 413", func(t *testing.T) {
		e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsBrut(corpsDeTaille(1<<20))).probleme(t, http.StatusRequestEntityTooLarge)
	})
	t.Run("modification au-delà de 16 Kio refusée en 413", func(t *testing.T) {
		article := e.creer(t, jeton, `{"titre":"Original"}`)
		e.envoyer(t, http.MethodPut, cheminArticles+"/"+article.ID, jeton, corpsBrut(corpsModificationDeTaille(limiteCorps+1))).probleme(t, http.StatusRequestEntityTooLarge)
		if relu := e.relire(t, jeton, article.ID); relu.Revision != 1 {
			t.Fatalf("un corps trop volumineux a modifié l'article : %+v", relu)
		}
	})
	if page := e.mesArticles(t, jeton, ""); page.Total != 2 {
		t.Fatalf("seuls le corps de 16 Kio et l'article modifié devaient exister : %+v", page)
	}
}

func TestUS009AC9TypeDeContenu(t *testing.T) {
	e := demarrerEnMemoire(t)
	jeton := e.vendeur()
	refuses := []string{"", "text/plain", "application/x-www-form-urlencoded", "application/jsonp"}
	for _, valeur := range refuses {
		t.Run("refusé en 415 : "+strings.ReplaceAll(valeur, "/", " "), func(t *testing.T) {
			e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsJSON(`{"titre":"a"}`), typeDeContenu(valeur)).probleme(t, http.StatusUnsupportedMediaType)
		})
	}
	t.Run("application/json avec charset accepté", func(t *testing.T) {
		e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsJSON(`{"titre":"a"}`), typeDeContenu("application/json; charset=utf-8")).exiger(t, http.StatusCreated)
	})
}
