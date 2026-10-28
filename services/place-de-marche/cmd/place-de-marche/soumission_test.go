package main

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"

	"collector/libs/platform/broker"
)

const (
	etatBrouillon  = "brouillon"
	etatEnControle = "en_controle"
	etatPublie     = "publie"
	etatRefuse     = "refuse"
	etatAVerifier  = "a_verifier"
	categorieSure  = "figurines"
	categorieHors  = "cartes-postales"
)

func cheminSoumission(id string) string {
	return cheminArticles + "/" + id + "/soumission"
}

func (e *environnement) soumettre(t *testing.T, jeton, id string, revision int) reponse {
	t.Helper()
	return e.envoyer(t, http.MethodPost, cheminSoumission(id), jeton, corpsJSON(fmt.Sprintf(`{"revision":%d}`, revision)))
}

func (e *environnement) soumettreAvecSucces(t *testing.T, jeton string, article articleRecu) articleRecu {
	t.Helper()
	var soumis articleRecu
	e.soumettre(t, jeton, article.ID, article.Revision).exiger(t, http.StatusAccepted).lire(t, &soumis)
	if soumis.Etat != etatEnControle || soumis.Revision != article.Revision+1 || soumis.Motif != nil {
		t.Fatalf("après soumission : état %q, révision %d, motif %v ; attendu en_controle, révision %d, sans motif",
			soumis.Etat, soumis.Revision, soumis.Motif, article.Revision+1)
	}
	return soumis
}

func exigerEtat(t *testing.T, article articleRecu, etat string, motif string) {
	t.Helper()
	if article.Etat != etat {
		t.Fatalf("article %s : état %q, attendu %q", article.ID, article.Etat, etat)
	}
	recu := ""
	if article.Motif != nil {
		recu = *article.Motif
	}
	if recu != motif {
		t.Fatalf("article %s : motif %q, attendu %q", article.ID, recu, motif)
	}
}

func (e *environnement) dansMesArticles(t *testing.T, jeton, id string) articleRecu {
	t.Helper()
	for _, a := range e.mesArticles(t, jeton, "?taille=50").Elements {
		if a.ID == id {
			return a
		}
	}
	t.Fatalf("article %s absent de « Mes articles »", id)
	return articleRecu{}
}

func corpsSansDescription(categorie string) string {
	return fmt.Sprintf(`{"titre":"Figurine Astérix","categorieId":%q,"prix":{"montantCentimes":2500,"devise":"EUR"}}`, categorie)
}

func exigerUnSeulResultatAcquitte(t *testing.T, livrees []livraison) {
	t.Helper()
	if len(livrees) != 1 {
		t.Fatalf("%d demande(s) de contrôle émise(s), 1 attendue", len(livrees))
	}
	if livrees[0].disposition != broker.Ack {
		t.Fatalf("résultat %s non acquitté", livrees[0].MessageID)
	}
}

func TestUS013AC1BrouillonCompletEnControlePuisPublie(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	article := b.creer(t, jeton, corpsComplet("Figurine Tintin", categorieSure, 4500, 600))
	b.soumettreAvecSucces(t, jeton, article)
	exigerEtat(t, b.relire(t, jeton, article.ID), etatEnControle, "")
	exigerUnSeulResultatAcquitte(t, b.controler(t))
	exigerEtat(t, b.relire(t, jeton, article.ID), etatPublie, "")
	exigerEtat(t, b.dansMesArticles(t, jeton, article.ID), etatPublie, "")
}

func TestUS013AC1SansFraisDePortResteComplet(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	article := b.creer(t, jeton, `{"titre":"Timbre","description":"Rare","categorieId":"timbres","prix":{"montantCentimes":100,"devise":"EUR"}}`)
	b.soumettreAvecSucces(t, jeton, article)
	b.controler(t)
	exigerEtat(t, b.relire(t, jeton, article.ID), etatPublie, "")
}

func TestUS013AC1SoumissionRefuseeSansEffet(t *testing.T) {
	cas := []struct {
		nom    string
		corps  func(revision int) string
		statut int
	}{
		{"révision future", func(r int) string { return fmt.Sprintf(`{"revision":%d}`, r+1) }, http.StatusConflict},
		{"révision absente", func(int) string { return `{}` }, http.StatusBadRequest},
		{"révision nulle", func(int) string { return `{"revision":0}` }, http.StatusBadRequest},
		{"révision en texte", func(int) string { return `{"revision":"1"}` }, http.StatusBadRequest},
		{"champ inconnu", func(r int) string { return fmt.Sprintf(`{"revision":%d,"etat":"publie"}`, r) }, http.StatusBadRequest},
		{"révision dupliquée", func(r int) string { return fmt.Sprintf(`{"revision":%d,"revision":%d}`, r, r) }, http.StatusBadRequest},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			jeton := b.vendeur()
			article := b.creer(t, jeton, corpsComplet("Figurine", categorieSure, 100, 0))
			b.envoyer(t, http.MethodPost, cheminSoumission(article.ID), jeton, corpsJSON(c.corps(article.Revision))).probleme(t, c.statut)
			relu := b.relire(t, jeton, article.ID)
			exigerEtat(t, relu, etatBrouillon, "")
			if relu.Revision != article.Revision || len(b.emettreLesDemandes(t)) != 0 {
				t.Fatalf("soumission refusée avec effet : révision %d, demandes émises", relu.Revision)
			}
		})
	}
}

func TestUS013AC1SoumissionAvecRevisionPerimeeRefusee(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	article := b.creer(t, jeton, corpsComplet("Figurine", categorieSure, 100, 0))
	b.modifier(t, jeton, article.ID, corpsRevision(article.Revision, corpsComplet("Figurine Milou", categorieSure, 100, 0))).exiger(t, http.StatusOK)
	b.soumettre(t, jeton, article.ID, article.Revision).probleme(t, http.StatusConflict)
	relu := b.relire(t, jeton, article.ID)
	exigerEtat(t, relu, etatBrouillon, "")
	if relu.Revision != article.Revision+1 || len(b.emettreLesDemandes(t)) != 0 {
		t.Fatalf("soumission périmée avec effet : révision %d", relu.Revision)
	}
}

func TestUS013AC1ArticleEnControleOuPublieNonSoumettable(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	article := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsComplet("Figurine", categorieSure, 100, 0)))
	b.soumettre(t, jeton, article.ID, article.Revision).probleme(t, http.StatusConflict)
	b.controler(t)
	publie := b.relire(t, jeton, article.ID)
	b.soumettre(t, jeton, article.ID, publie.Revision).probleme(t, http.StatusConflict)
	exigerEtat(t, b.relire(t, jeton, article.ID), etatPublie, "")
	if len(b.emettreLesDemandes(t)) != 0 {
		t.Fatal("une soumission refusée a émis une demande de contrôle")
	}
}

func TestUS013AC2BrouillonIncompletRefuseAvecMotif(t *testing.T) {
	cas := map[string]string{
		"sans description":       corpsSansDescription(categorieSure),
		"description en espaces": `{"titre":"Figurine","description":"   ","categorieId":"figurines","prix":{"montantCentimes":2500,"devise":"EUR"}}`,
		"sans titre":             `{"description":"Belle","categorieId":"figurines","prix":{"montantCentimes":2500,"devise":"EUR"}}`,
		"sans catégorie":         `{"titre":"Figurine","description":"Belle","prix":{"montantCentimes":2500,"devise":"EUR"}}`,
		"sans prix":              `{"titre":"Figurine","description":"Belle","categorieId":"figurines"}`,
		"brouillon vide":         `{}`,
	}
	for nom, corps := range cas {
		t.Run(nom, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			jeton := b.vendeur()
			article := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corps))
			exigerUnSeulResultatAcquitte(t, b.controler(t))
			exigerEtat(t, b.relire(t, jeton, article.ID), etatRefuse, motifIncomplet)
			exigerEtat(t, b.dansMesArticles(t, jeton, article.ID), etatRefuse, motifIncomplet)
		})
	}
}

func TestUS013AC3ArticleRefuseCorrigeEtResoumisPuisPublie(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	soumis := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsSansDescription(categorieSure)))
	b.controler(t)
	refuse := b.relire(t, jeton, soumis.ID)
	exigerEtat(t, refuse, etatRefuse, motifIncomplet)
	var corrige articleRecu
	b.modifier(t, jeton, refuse.ID, corpsRevision(refuse.Revision, corpsComplet("Figurine Astérix", categorieSure, 2500, 0))).exiger(t, http.StatusOK).lire(t, &corrige)
	resoumis := b.soumettreAvecSucces(t, jeton, corrige)
	exigerEtat(t, b.relire(t, jeton, resoumis.ID), etatEnControle, "")
	exigerUnSeulResultatAcquitte(t, b.controler(t))
	exigerEtat(t, b.relire(t, jeton, resoumis.ID), etatPublie, "")
}

func TestUS013AC3ArticleRefuseResoumisSansCorrectionRepasseEnControle(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	soumis := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsSansDescription(categorieSure)))
	b.controler(t)
	resoumis := b.soumettreAvecSucces(t, jeton, b.relire(t, jeton, soumis.ID))
	b.controler(t)
	exigerEtat(t, b.relire(t, jeton, resoumis.ID), etatRefuse, motifIncomplet)
}

func TestUS013AC4CategorieInconnueDuControleAVerifierJamaisPublie(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	article := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsComplet("Carte postale de Paris", categorieHors, 300, 0)))
	exigerUnSeulResultatAcquitte(t, b.controler(t))
	exigerEtat(t, b.relire(t, jeton, article.ID), etatAVerifier, motifCategorieInconnu)
	b.soumettre(t, jeton, article.ID, article.Revision).probleme(t, http.StatusConflict)
	exigerEtat(t, b.dansMesArticles(t, jeton, article.ID), etatAVerifier, motifCategorieInconnu)
}

func TestUS013AC5DemandePorteLArticleSaRevisionEtSonContenu(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	article := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsComplet("Figurine Tintin", categorieSure, 4500, 600)))
	demandes := b.emettreLesDemandes(t)
	if len(demandes) != 1 {
		t.Fatalf("%d demande(s) émise(s), 1 attendue", len(demandes))
	}
	d := demandes[0]
	if d.ArticleID != article.ID || d.Revision != article.Revision || d.Listing.Title != "Figurine Tintin" ||
		d.Listing.CategoryCode == nil || *d.Listing.CategoryCode != categorieSure || d.Listing.PriceCents == nil || *d.Listing.PriceCents != 4500 {
		t.Fatalf("demande %+v ne reflète pas l'article soumis %+v", d, article)
	}
}

func TestUS013AC6ResultatLivreDeuxFoisSansEffet(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	article := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsComplet("Figurine", categorieSure, 100, 0)))
	livree := b.controler(t)[0]
	avant := b.relire(t, jeton, article.ID)
	rejouee := livree.Delivery
	rejouee.Redelivered = true
	if b.livrer(t, rejouee).disposition != broker.Ack {
		t.Fatal("doublon non acquitté")
	}
	apres := b.relire(t, jeton, article.ID)
	if !reflect.DeepEqual(apres, avant) {
		t.Fatalf("doublon avec effet : %+v devenu %+v", avant, apres)
	}
}

func TestUS013AC6ResultatDUneRevisionAnterieureSansEffet(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	premier := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsSansDescription(categorieSure)))
	ancien := b.controler(t)[0]
	var corrige articleRecu
	b.modifier(t, jeton, premier.ID, corpsRevision(premier.Revision, corpsComplet("Figurine", categorieSure, 100, 0))).exiger(t, http.StatusOK).lire(t, &corrige)
	resoumis := b.soumettreAvecSucces(t, jeton, corrige)
	if b.livrer(t, ancien.Delivery).disposition != broker.Ack {
		t.Fatal("résultat périmé non acquitté")
	}
	exigerEtat(t, b.relire(t, jeton, resoumis.ID), etatEnControle, "")
	b.livrer(t, b.resultatSigne(t, resultatEmis{ArticleID: resoumis.ID, Revision: resoumis.Revision - 1, RulesVersion: 1, Issue: issuePublie}))
	exigerEtat(t, b.relire(t, jeton, resoumis.ID), etatEnControle, "")
	b.controler(t)
	exigerEtat(t, b.relire(t, jeton, resoumis.ID), etatPublie, "")
}

func TestUS013AC6ResultatPourUnArticleInconnuSansEffet(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	livree := b.livrer(t, b.resultatSigne(t, resultatEmis{ArticleID: uuid.NewString(), Revision: 2, RulesVersion: 1, Issue: issuePublie}))
	if livree.disposition != broker.Ack {
		t.Fatal("résultat sans article non acquitté")
	}
}

type exempleDeContrat struct {
	chemin string
	valide bool
}

func exemplesDeResultats(t *testing.T) []exempleDeContrat {
	t.Helper()
	racine := filepath.Join("..", "..", "..", "..", "contracts", "messages", "controle.resultat", "exemples", "v1")
	var exemples []exempleDeContrat
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		relatif, _ := filepath.Rel(racine, chemin)
		exemples = append(exemples, exempleDeContrat{chemin: chemin, valide: strings.HasPrefix(relatif, "valides")})
		return nil
	})
	if err != nil || len(exemples) == 0 {
		t.Fatalf("exemples du contrat introuvables : %v", err)
	}
	return exemples
}

func TestUS013AC8ConsommateurDeResultatsRespecteLeContratV1(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	for _, exemple := range exemplesDeResultats(t) {
		t.Run(filepath.Base(filepath.Dir(exemple.chemin))+"/"+filepath.Base(exemple.chemin), func(t *testing.T) {
			corps, err := os.ReadFile(exemple.chemin)
			if err != nil {
				t.Fatalf("lecture : %v", err)
			}
			id := uuid.NewString()
			livree := b.livrer(t, broker.Delivery{MessageID: id, UserID: identiteControle, Body: corps})
			attendue := broker.Reject
			if exemple.valide {
				attendue = broker.Ack
			}
			if livree.disposition != attendue {
				t.Fatalf("disposition %d, attendue %d", livree.disposition, attendue)
			}
			if !exemple.valide && !b.journal.contient(id) {
				t.Fatalf("message non conforme %s écarté sans être journalisé", id)
			}
		})
	}
}

func TestUS013AC8ResultatNonConformeEcarteSansEffet(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	article := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsComplet("Figurine", categorieSure, 100, 0)))
	b.emettreLesDemandes(t)
	corps := fmt.Sprintf(`{"id":%q,"type":"controle.resultat","version":1,"emitted_at":"2026-10-26T09:00:00Z","correlation_id":%q,"data":{"article_id":%q,"revision":%d,"rules_version":1,"issue":"publie","reason":"annonce_incomplete"}}`,
		uuid.NewString(), uuid.NewString(), article.ID, article.Revision)
	if b.livrer(t, broker.Delivery{MessageID: uuid.NewString(), UserID: identiteControle, Body: []byte(corps)}).disposition != broker.Reject {
		t.Fatal("résultat non conforme accepté")
	}
	exigerEtat(t, b.relire(t, jeton, article.ID), etatEnControle, "")
}

func TestUS013AC9SoumissionDeLArticleDUnAutreVendeurIntrouvable(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	vendeurA, vendeurB := b.vendeur(), b.vendeur()
	article := b.creer(t, vendeurA, corpsComplet("Figurine", categorieSure, 100, 0))
	b.soumettre(t, vendeurB, article.ID, article.Revision).probleme(t, http.StatusNotFound)
	b.soumettre(t, vendeurB, uuid.NewString(), 1).probleme(t, http.StatusNotFound)
	exigerEtat(t, b.relire(t, vendeurA, article.ID), etatBrouillon, "")
	if len(b.emettreLesDemandes(t)) != 0 {
		t.Fatal("la tentative du vendeur B a émis une demande de contrôle")
	}
}

func TestUS013AC9ReponseIdentiquePourArticleDAutruiEtInexistant(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	vendeurA, vendeurB := b.vendeur(), b.vendeur()
	article := b.creer(t, vendeurA, corpsComplet("Figurine", categorieSure, 100, 0))
	dAutrui := b.soumettre(t, vendeurB, article.ID, 1).probleme(t, http.StatusNotFound)
	inexistant := b.soumettre(t, vendeurB, uuid.NewString(), 1).probleme(t, http.StatusNotFound)
	if dAutrui.Type != inexistant.Type || dAutrui.Title != inexistant.Title || dAutrui.Detail != inexistant.Detail {
		t.Fatalf("réponses distinguables : %+v / %+v", dAutrui, inexistant)
	}
}

func TestUS013SoumissionReserveeAuVendeurAuthentifie(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	article := b.creer(t, b.vendeur(), corpsComplet("Figurine", categorieSure, 100, 0))
	b.soumettre(t, "", article.ID, 1).probleme(t, http.StatusUnauthorized)
	b.soumettre(t, bearer(b.jetons.acheteur(uuid.NewString())), article.ID, 1).probleme(t, http.StatusForbidden)
}

func TestUS013ReponsesDeSoumissionConformesAuContratOpenAPI(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	routeur := chargerContrat(t, b.adresse)
	jeton := b.vendeur()
	refuse := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsSansDescription(categorieSure)))
	b.soumettre(t, jeton, refuse.ID, refuse.Revision).exiger(t, http.StatusConflict)
	b.soumettre(t, b.vendeur(), refuse.ID, refuse.Revision).exiger(t, http.StatusNotFound)
	b.envoyer(t, http.MethodPost, cheminSoumission(refuse.ID), jeton, corpsJSON(`{}`)).exiger(t, http.StatusBadRequest)
	b.controler(t)
	b.relire(t, jeton, refuse.ID)
	b.mesArticles(t, jeton, "")
	for _, ech := range b.historique() {
		verifierEchange(t, routeur, ech)
	}
}
