package main

import (
	"net/http"
	"testing"
)

type bancDeRevalidation struct {
	*bancComplet
	publie           articleRecu
	resultatDuPublie publication
	modifie          articleRecu
}

func (b *bancComplet) trancherParLeCanal(t *testing.T, article articleRecu, issue string, motif *string) publication {
	t.Helper()
	p := versLesResultats(identiteControle, resultatForge(article, issue, motif))
	if err := b.canal.publier(t, identiteControle, p); err != nil {
		t.Fatalf("publication du résultat de %s : %v", article.ID, err)
	}
	return p
}

func (b *bancComplet) rejouer(t *testing.T, p publication) {
	t.Helper()
	if err := b.canal.publier(t, identiteControle, p); err != nil {
		t.Fatalf("redélivraison de %s : %v", p.id, err)
	}
}

func (b *bancComplet) attendreUnTemoinTraite(t *testing.T) {
	t.Helper()
	temoin := b.creerEtSoumettre(t, corpsComplet("Figurine témoin", categorieSure, 100, 0))
	b.trancherParLeCanal(t, temoin, issuePublie, nil)
	b.attendreEtat(t, temoin.ID, etatPublie, delaiDeTraitement)
}

func (b *bancComplet) modifierAvecSucces(t *testing.T, article articleRecu, contenu string) articleRecu {
	t.Helper()
	var modifie articleRecu
	b.modifier(t, b.jeton, article.ID, corpsRevision(article.Revision, contenu)).exiger(t, http.StatusOK).lire(t, &modifie)
	return modifie
}

func (b *bancComplet) exigerDemandeEnOutbox(t *testing.T, article articleRecu, demandesAttendues int) {
	t.Helper()
	lignes := b.outboxDe(t, article.ID)
	if len(lignes) != demandesAttendues {
		t.Fatalf("%d demande(s) en outbox pour %s, %d attendue(s)", len(lignes), article.ID, demandesAttendues)
	}
	derniere := lignes[len(lignes)-1]
	var demande demandeRecue
	if _, err := b.validateur.Decode(derniere.corps, &demande); err != nil {
		t.Fatalf("demande de l'outbox non conforme au contrat v1 : %v", err)
	}
	if derniere.echange != echangeDemandes || derniere.typ != typeDemande || demande.Revision != article.Revision || demande.Listing.Title != article.Titre {
		t.Fatalf("dernière demande %+v (échange %q), attendue en révision %d avec le titre %q", demande, derniere.echange, article.Revision, article.Titre)
	}
	attendre(t, delaiDeTraitement, "publication de la demande de la nouvelle révision", func() bool {
		return b.outboxDe(t, article.ID)[demandesAttendues-1].publiee
	})
}

func (b *bancComplet) exigerHorsCatalogue(t *testing.T, article articleRecu) {
	t.Helper()
	exigerAbsents(t, b.toutLeCatalogue(t, ""), article)
	b.fiche(t, article.ID).probleme(t, http.StatusNotFound)
}

func (b *bancComplet) exigerInchange(t *testing.T, avant articleRecu) {
	t.Helper()
	apres := b.relire(t, b.jeton, avant.ID)
	if apres.Etat != avant.Etat || apres.Revision != avant.Revision || apres.ModifieLe != avant.ModifieLe || apres.Titre != avant.Titre {
		t.Fatalf("résultat périmé avec effet : %+v devenu %+v", avant, apres)
	}
}

func TestUS021BoutEnBoutRevalidationSurPostgreSQLEtRabbitMQ(t *testing.T) {
	b := &bancDeRevalidation{bancComplet: demarrerBancComplet(t)}
	soumis := b.creerEtSoumettre(t, corpsInitial())
	b.resultatDuPublie = b.trancherParLeCanal(t, soumis, issuePublie, nil)
	b.publie, _ = b.attendreEtat(t, soumis.ID, etatPublie, delaiDeTraitement)
	exigerPresents(t, b.toutLeCatalogue(t, ""), b.publie)
	t.Run("AC1 seul titre modifié sur un publié : nouvelle révision en contrôle, retirée du catalogue, demande en outbox", b.ac1SeulTitreModifie)
	t.Run("AC5 résultat de l'ancienne révision redélivré par le canal sans effet", b.ac5ResultatRedelivreSansEffet)
	t.Run("AC3 nouvelle révision acceptée revient au catalogue avec le titre modifié", b.ac3NouvelleRevisionAcceptee)
	t.Run("AC2 article à vérifier modifié : nouvelle demande, résultat de l'ancienne révision sans effet", b.ac2AVerifierModifie)
	t.Run("AC2 article en contrôle modifié : nouvelle demande, résultat de l'ancienne révision sans effet", b.ac2EnControleModifie)
}

func (b *bancDeRevalidation) ac1SeulTitreModifie(t *testing.T) {
	b.modifie = b.modifierAvecSucces(t, b.publie, corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux))
	exigerNouvelleRevisionEnControle(t, b.modifie, b.publie)
	relu := b.relire(t, b.jeton, b.publie.ID)
	exigerNouvelleRevisionEnControle(t, relu, b.publie)
	b.exigerHorsCatalogue(t, b.publie)
	b.exigerDemandeEnOutbox(t, relu, 2)
}

func (b *bancDeRevalidation) ac5ResultatRedelivreSansEffet(t *testing.T) {
	avant := b.relire(t, b.jeton, b.publie.ID)
	b.rejouer(t, b.resultatDuPublie)
	b.trancherParLeCanal(t, b.publie, issuePublie, nil)
	b.attendreUnTemoinTraite(t)
	b.exigerInchange(t, avant)
	b.exigerHorsCatalogue(t, b.publie)
}

func (b *bancDeRevalidation) ac3NouvelleRevisionAcceptee(t *testing.T) {
	b.trancherParLeCanal(t, b.modifie, issuePublie, nil)
	republie, _ := b.attendreEtat(t, b.modifie.ID, etatPublie, delaiDeTraitement)
	if republie.Revision != b.modifie.Revision || republie.Titre != titreModifie {
		t.Fatalf("article republié %+v, attendu en révision %d au titre %q", republie, b.modifie.Revision, titreModifie)
	}
	var entree *entreeCatalogue
	for _, e := range b.toutLeCatalogue(t, "") {
		if e.ID == b.modifie.ID {
			entree = &e
		}
	}
	if entree == nil || entree.Titre != titreModifie {
		t.Fatalf("entrée du catalogue %+v, titre %q attendu", entree, titreModifie)
	}
	if f := b.lireFiche(t, b.modifie.ID); f.Titre != titreModifie {
		t.Fatalf("fiche publique au titre %q, %q attendu", f.Titre, titreModifie)
	}
}

func (b *bancDeRevalidation) ac2AVerifierModifie(t *testing.T) {
	soumis := b.creerEtSoumettre(t, corpsInitial())
	resultatAncien := b.trancherParLeCanal(t, soumis, issueAVerifier, motif(motifCategorieInconnu))
	aVerifier, _ := b.attendreEtat(t, soumis.ID, etatAVerifier, delaiDeTraitement)
	modifie := b.modifierAvecSucces(t, aVerifier, corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux))
	exigerNouvelleRevisionEnControle(t, modifie, aVerifier)
	b.exigerDemandeEnOutbox(t, modifie, 2)
	avant := b.relire(t, b.jeton, modifie.ID)
	b.rejouer(t, resultatAncien)
	b.trancherParLeCanal(t, aVerifier, issuePublie, nil)
	b.attendreUnTemoinTraite(t)
	b.exigerInchange(t, avant)
	b.exigerHorsCatalogue(t, modifie)
}

func (b *bancDeRevalidation) ac2EnControleModifie(t *testing.T) {
	soumis := b.creerEtSoumettre(t, corpsInitial())
	modifie := b.modifierAvecSucces(t, soumis, corpsComplet(titreModifie, categorieSure, prixInitial, fraisInitiaux))
	exigerNouvelleRevisionEnControle(t, modifie, soumis)
	b.exigerDemandeEnOutbox(t, modifie, 2)
	avant := b.relire(t, b.jeton, modifie.ID)
	b.trancherParLeCanal(t, soumis, issuePublie, nil)
	b.attendreUnTemoinTraite(t)
	b.exigerInchange(t, avant)
	b.exigerHorsCatalogue(t, modifie)
}
