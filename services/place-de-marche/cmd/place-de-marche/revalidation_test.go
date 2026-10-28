package main

import (
	"net/http"
	"reflect"
	"testing"

	"collector/libs/platform/broker"
)

const (
	titreInitial  = "Figurine Tintin"
	titreModifie  = "Figurine Tintin, appelez-moi"
	prixInitial   = 1500
	fraisInitiaux = 400
)

func corpsInitial() string {
	return corpsComplet(titreInitial, categorieSure, prixInitial, fraisInitiaux)
}

func (b *bancEnMemoire) depuisLEtat(t *testing.T, jeton, etat string) articleRecu {
	t.Helper()
	article := b.articleDansLEtat(t, jeton, titreInitial, categorieSure, etat)
	b.emettreLesDemandes(t)
	return article
}

func (b *bancEnMemoire) modifierAvecSucces(t *testing.T, jeton string, article articleRecu, contenu string) articleRecu {
	t.Helper()
	var modifie articleRecu
	b.modifier(t, jeton, article.ID, corpsRevision(article.Revision, contenu)).exiger(t, http.StatusOK).lire(t, &modifie)
	return modifie
}

func exigerNouvelleRevisionEnControle(t *testing.T, modifie articleRecu, avant articleRecu) {
	t.Helper()
	exigerEtat(t, modifie, etatEnControle, "")
	if modifie.Revision != avant.Revision+1 {
		t.Fatalf("révision %d après modification, %d attendue", modifie.Revision, avant.Revision+1)
	}
}

func (b *bancEnMemoire) exigerUneDemandePour(t *testing.T, article articleRecu) demandeRecue {
	t.Helper()
	demandes := b.emettreLesDemandes(t)
	if len(demandes) != 1 {
		t.Fatalf("%d demande(s) de contrôle émise(s), 1 attendue", len(demandes))
	}
	d := demandes[0]
	if d.ArticleID != article.ID || d.Revision != article.Revision || d.Listing.Title != article.Titre {
		t.Fatalf("demande %+v, attendue pour %s en révision %d avec le titre %q", d, article.ID, article.Revision, article.Titre)
	}
	return d
}

func (b *bancEnMemoire) exigerHorsCatalogue(t *testing.T, article articleRecu) {
	t.Helper()
	exigerAbsents(t, b.toutLeCatalogue(t, ""), article)
	b.fiche(t, article.ID).probleme(t, http.StatusNotFound)
}

func TestUS021AC1SeulTitreModifieSurUnPublieRetireDuCatalogueEtRecontrole(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	publie := b.depuisLEtat(t, jeton, etatPublie)
	exigerPresents(t, b.toutLeCatalogue(t, ""), publie)

	modifie := b.modifierAvecSucces(t, jeton, publie, corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux))

	exigerNouvelleRevisionEnControle(t, modifie, publie)
	if modifie.Titre != titreModifie {
		t.Fatalf("titre %q après modification, %q attendu", modifie.Titre, titreModifie)
	}
	relu := b.relire(t, jeton, publie.ID)
	exigerNouvelleRevisionEnControle(t, relu, publie)
	exigerEtat(t, b.dansMesArticles(t, jeton, publie.ID), etatEnControle, "")
	b.exigerHorsCatalogue(t, publie)
	b.exigerUneDemandePour(t, relu)
}

func TestUS021AC1ToutChampModifieSansSeuilRecontrole(t *testing.T) {
	cas := map[string]string{
		"titre":                   corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux),
		"prix d'un centime":       corpsComplet(titreInitial, categorieSure, prixInitial+1, fraisInitiaux),
		"frais de port":           corpsComplet(titreInitial, categorieSure, prixInitial, fraisInitiaux+1),
		"catégorie":               corpsComplet(titreInitial, "timbres", prixInitial, fraisInitiaux),
		"description":             `{"titre":"Figurine Tintin","description":"Belle pièce","categorieId":"figurines","prix":{"montantCentimes":1500,"devise":"EUR"},"fraisDePort":{"montantCentimes":400,"devise":"EUR"}}`,
		"contenu inchangé":        corpsInitial(),
		"frais de port supprimés": corpsSansFrais(),
	}
	for nom, corps := range cas {
		t.Run(nom, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			jeton := b.vendeur()
			publie := b.depuisLEtat(t, jeton, etatPublie)

			modifie := b.modifierAvecSucces(t, jeton, publie, corps)

			exigerNouvelleRevisionEnControle(t, modifie, publie)
			b.exigerHorsCatalogue(t, publie)
			b.exigerUneDemandePour(t, modifie)
		})
	}
}

func corpsSansFrais() string {
	return `{"titre":"Figurine Tintin","description":"Belle pièce\nen bon état","categorieId":"figurines","prix":{"montantCentimes":1500,"devise":"EUR"}}`
}

func TestUS021AC1ModificationRefuseeDUnPublieSansEffet(t *testing.T) {
	cas := []struct {
		nom    string
		corps  func(publie articleRecu) string
		statut int
	}{
		{"révision périmée", func(p articleRecu) string { return corpsRevision(p.Revision-1, corpsInitial()) }, http.StatusConflict},
		{"révision future", func(p articleRecu) string { return corpsRevision(p.Revision+1, corpsInitial()) }, http.StatusConflict},
		{"saisie invalide", func(p articleRecu) string {
			return corpsRevision(p.Revision, corpsComplet(titreModifie, categorieSure, -1, 0))
		}, http.StatusBadRequest},
		{"catégorie hors jeu", func(p articleRecu) string {
			return corpsRevision(p.Revision, corpsComplet(titreModifie, "inexistante", prixInitial, 0))
		}, http.StatusBadRequest},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			jeton := b.vendeur()
			publie := b.depuisLEtat(t, jeton, etatPublie)

			b.modifier(t, jeton, publie.ID, c.corps(publie)).probleme(t, c.statut)

			if relu := b.relire(t, jeton, publie.ID); !reflect.DeepEqual(relu, publie) {
				t.Fatalf("modification refusée avec effet : %+v devenu %+v", publie, relu)
			}
			exigerPresents(t, b.toutLeCatalogue(t, ""), publie)
			if n := len(b.emettreLesDemandes(t)); n != 0 {
				t.Fatalf("%d demande(s) émise(s) par une modification refusée", n)
			}
		})
	}
}

func TestUS021AC1PublieDUnAutreVendeurNonModifiable(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	vendeurA, vendeurB := b.vendeur(), b.vendeur()
	publie := b.depuisLEtat(t, vendeurA, etatPublie)

	b.modifier(t, vendeurB, publie.ID, corpsRevision(publie.Revision, corpsComplet(titreModifie, categorieSure, 1, 0))).probleme(t, http.StatusNotFound)

	exigerEtat(t, b.relire(t, vendeurA, publie.ID), etatPublie, "")
	exigerPresents(t, b.toutLeCatalogue(t, ""), publie)
	if n := len(b.emettreLesDemandes(t)); n != 0 {
		t.Fatalf("%d demande(s) émise(s) par la tentative d'un autre vendeur", n)
	}
}

func TestUS021BrouillonEtRefuseModifiesSansControle(t *testing.T) {
	for _, etat := range []string{etatBrouillon, etatRefuse} {
		t.Run(etat, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			jeton := b.vendeur()
			article := b.depuisLEtat(t, jeton, etat)

			modifie := b.modifierAvecSucces(t, jeton, article, corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux))

			if modifie.Etat != etat || modifie.Revision != article.Revision+1 {
				t.Fatalf("après modification : état %q, révision %d ; attendu %q, révision %d", modifie.Etat, modifie.Revision, etat, article.Revision+1)
			}
			if n := len(b.emettreLesDemandes(t)); n != 0 {
				t.Fatalf("%d demande(s) émise(s) par la modification d'un %s", n, etat)
			}
		})
	}
}

func TestUS021AC2ModificationEnControleOuAVerifierSoumetLaNouvelleRevision(t *testing.T) {
	for _, etat := range []string{etatEnControle, etatAVerifier} {
		t.Run(etat, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			jeton := b.vendeur()
			article := b.depuisLEtat(t, jeton, etat)

			modifie := b.modifierAvecSucces(t, jeton, article, corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux))

			exigerNouvelleRevisionEnControle(t, modifie, article)
			exigerEtat(t, b.relire(t, jeton, article.ID), etatEnControle, "")
			b.exigerUneDemandePour(t, modifie)
			b.exigerHorsCatalogue(t, modifie)
		})
	}
}

func TestUS021AC2ResultatSurLAncienneRevisionSansEffet(t *testing.T) {
	issues := []struct {
		issue string
		motif *string
	}{
		{issuePublie, nil},
		{issueRefuse, motif(motifIncomplet)},
		{issueAVerifier, motif(motifCategorieInconnu)},
	}
	for _, etat := range []string{etatEnControle, etatAVerifier} {
		for _, i := range issues {
			t.Run(etat+" puis "+i.issue, func(t *testing.T) {
				b := demarrerBancEnMemoire(t)
				jeton := b.vendeur()
				ancienne := b.depuisLEtat(t, jeton, etat)
				modifie := b.modifierAvecSucces(t, jeton, ancienne, corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux))
				avant := b.relire(t, jeton, modifie.ID)

				livree := b.livrer(t, b.resultatSigne(t, resultatEmis{ArticleID: ancienne.ID, Revision: ancienne.Revision, RulesVersion: versionDesRegles, Issue: i.issue, Reason: i.motif}))

				if livree.disposition != broker.Ack {
					t.Fatal("résultat de l'ancienne révision non acquitté")
				}
				if apres := b.relire(t, jeton, modifie.ID); !reflect.DeepEqual(apres, avant) {
					t.Fatalf("résultat de l'ancienne révision avec effet : %+v devenu %+v", avant, apres)
				}
				b.exigerHorsCatalogue(t, modifie)
			})
		}
	}
}

func TestUS021AC2ResultatSurLaRevisionSoumiseAvantModificationSansEffet(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	soumis := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsInitial()))
	b.emettreLesDemandes(t)
	modifie := b.modifierAvecSucces(t, jeton, soumis, corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux))

	b.trancher(t, soumis, issuePublie, nil)

	exigerEtat(t, b.relire(t, jeton, modifie.ID), etatEnControle, "")
	b.exigerHorsCatalogue(t, modifie)
	exigerUnSeulResultatAcquitte(t, b.controler(t))
	exigerEtat(t, b.relire(t, jeton, modifie.ID), etatPublie, "")
}

func TestUS021AC3NouvelleRevisionAccepteeRevientAuCatalogueModifiee(t *testing.T) {
	for _, etat := range []string{etatPublie, etatEnControle, etatAVerifier} {
		t.Run(etat, func(t *testing.T) {
			b := demarrerBancEnMemoire(t)
			jeton := b.vendeur()
			article := b.depuisLEtat(t, jeton, etat)
			modifie := b.modifierAvecSucces(t, jeton, article, corpsComplet(titreModifie, categorieSure, prixInitial+100, fraisInitiaux))

			exigerUnSeulResultatAcquitte(t, b.controler(t))

			relu := b.relire(t, jeton, article.ID)
			exigerEtat(t, relu, etatPublie, "")
			if relu.Revision != modifie.Revision {
				t.Fatalf("révision %d publiée, %d attendue", relu.Revision, modifie.Revision)
			}
			exigerContenuPublieModifie(t, b.environnement, article.ID)
		})
	}
}

func exigerContenuPublieModifie(t *testing.T, e *environnement, id string) {
	t.Helper()
	var entree *entreeCatalogue
	for _, c := range e.toutLeCatalogue(t, "") {
		if c.ID == id {
			entree = &c
		}
	}
	if entree == nil || entree.Titre != titreModifie || entree.Prix == nil || entree.Prix.MontantCentimes != prixInitial+100 {
		t.Fatalf("entrée du catalogue %+v, attendue avec le titre %q et le prix %d", entree, titreModifie, prixInitial+100)
	}
	if f := e.lireFiche(t, id); f.Titre != titreModifie {
		t.Fatalf("fiche publique au titre %q, %q attendu", f.Titre, titreModifie)
	}
}

func TestUS021AC5ResultatRedelivreApresNouvelleRevisionSansEffet(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	jeton := b.vendeur()
	soumis := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corpsInitial()))
	premier := b.controler(t)[0]
	publie := b.relire(t, jeton, soumis.ID)
	exigerEtat(t, publie, etatPublie, "")
	modifie := b.modifierAvecSucces(t, jeton, publie, corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux))
	avant := b.relire(t, jeton, modifie.ID)

	redelivre := premier.Delivery
	redelivre.Redelivered = true
	if b.livrer(t, redelivre).disposition != broker.Ack {
		t.Fatal("résultat redélivré non acquitté")
	}
	b.livrer(t, premier.Delivery)

	if apres := b.relire(t, jeton, modifie.ID); !reflect.DeepEqual(apres, avant) {
		t.Fatalf("résultat redélivré avec effet : %+v devenu %+v", avant, apres)
	}
	b.exigerHorsCatalogue(t, modifie)
}

func TestUS021ReponsesDeModificationConformesAuContratOpenAPI(t *testing.T) {
	b := demarrerBancEnMemoire(t)
	routeur := chargerContrat(t, b.adresse)
	jeton := b.vendeur()
	publie := b.depuisLEtat(t, jeton, etatPublie)
	modifie := b.modifierAvecSucces(t, jeton, publie, corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux))
	b.modifier(t, jeton, modifie.ID, corpsRevision(publie.Revision, corpsInitial())).exiger(t, http.StatusConflict)
	b.relire(t, jeton, modifie.ID)
	for _, ech := range b.historique() {
		verifierEchange(t, routeur, ech)
	}
}
