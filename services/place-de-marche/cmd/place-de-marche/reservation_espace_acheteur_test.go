package main

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const (
	commandesAmorcees      = 25
	tailleDeListeParDefaut = 20
)

type pageDeCommandes struct {
	Elements []commandeRecue `json:"elements"`
	Page     int             `json:"page"`
	Taille   int             `json:"taille"`
	Total    int             `json:"total"`
}

type commandeAmorcee struct {
	id     string
	creeLe time.Time
}

func TestUS032EspaceAcheteur(t *testing.T) {
	b := demarrerBancReservationAvec(t, dureeCourte, dureeCourte)
	t.Run("AC1 l'acheteur voit pour chaque réservation l'article, le montant total, l'échéance et l'état, la plus récente d'abord", b.us032ListeDeLAcheteur)
	t.Run("AC1 titre figé à la réservation : la modification ultérieure de l'annonce ne le change pas", b.us032TitreFige)
	t.Run("AC2 échéance passée pendant la consultation : la liste rafraîchie montre l'état expiree", b.us032ExpireePendantLaConsultation)
	t.Run("AC3 B ne voit que ses réservations, le détail d'une réservation de A répond 404", b.us032CloisonnementEntreAcheteurs)
	t.Run("AC3 liste sans acheteur authentifié refusée : sans jeton, jeton illisible, jeton de service", b.us032SansAcheteurAuthentifie)
	t.Run("Pagination et tri sur des commandes amorcées en base", b.us032PaginationEtTri)
	t.Run("Pagination hors bornes : 400 saisie-invalide", b.us032PaginationHorsBornes)
	t.Run("Réserver (port interne) renvoie le titre de l'article", b.us032ReserverRenvoieLeTitre)
	t.Run("Contrats OpenAPI : liste, création et détail de Transactions, réserver interne", b.us032Contrats)
}

func (b *bancReservation) listerCommandes(t *testing.T, jeton, requete string) reponse {
	t.Helper()
	return b.appeler(t, "http://"+b.tx.api, http.MethodGet, cheminCommandes+requete, nil, avecJeton(jeton))
}

func (b *bancReservation) mesCommandes(t *testing.T, jeton, requete string) pageDeCommandes {
	t.Helper()
	var page pageDeCommandes
	b.listerCommandes(t, jeton, requete).exiger(t, http.StatusOK).lire(t, &page)
	return page
}

func identifiantsDe(page pageDeCommandes) []string {
	ids := make([]string, 0, len(page.Elements))
	for _, c := range page.Elements {
		ids = append(ids, c.ID)
	}
	return ids
}

func exigerElementDeListe(t *testing.T, element commandeRecue, origine commandeRecue, titre, statut string) {
	t.Helper()
	if element.ID != origine.ID || element.ArticleID != origine.ArticleID {
		t.Errorf("élément %s (article %s), commande %s (article %s) attendue", element.ID, element.ArticleID, origine.ID, origine.ArticleID)
	}
	if element.Titre != titre {
		t.Errorf("commande %s : titre %q, %q attendu", element.ID, element.Titre, titre)
	}
	if element.Statut != statut {
		t.Errorf("commande %s : état %q, %q attendu", element.ID, element.Statut, statut)
	}
	exigerMemeIdEtEcheance(t, "élément de la liste", element, origine)
	exigerMontant(t, "montant total listé", element.TotalPaye, prixDeReference+fraisDeReference)
	exigerMontant(t, "prix listé", element.Prix, prixDeReference)
}

func (b *bancReservation) us032ListeDeLAcheteur(t *testing.T) {
	articles := b.publierPlusieurs(t, b.kc.creerVendeur(t), 2)
	acheteur := b.connecterAcheteur(t)
	vide := b.mesCommandes(t, acheteur.jeton, "")
	if vide.Total != 0 || len(vide.Elements) != 0 || vide.Page != 1 || vide.Taille != tailleDeListeParDefaut {
		t.Fatalf("liste d'un acheteur sans réservation : %+v", vide)
	}
	premiere := b.commanderAvecSucces(t, acheteur.jeton, articles[0].ID, http.StatusCreated)
	active := exigerAvantLEcheance(t, premiere.Echeance)
	page := b.mesCommandes(t, acheteur.jeton, "")
	active("première liste")
	if page.Total != 1 || len(page.Elements) != 1 {
		t.Fatalf("liste après une réservation : %+v", page)
	}
	exigerElementDeListe(t, page.Elements[0], premiere, articles[0].Titre, statutEnAttente)
	attendreLEcheance(t, premiere.Echeance)
	seconde := b.commanderAvecSucces(t, acheteur.jeton, articles[1].ID, http.StatusCreated)
	page = b.mesCommandes(t, acheteur.jeton, "")
	exigerAvantLEcheance(t, seconde.Echeance)("seconde liste")
	if page.Total != 2 || !slices.Equal(identifiantsDe(page), []string{seconde.ID, premiere.ID}) {
		t.Fatalf("liste %v (total %d), [%s %s] attendue (plus récente d'abord)", identifiantsDe(page), page.Total, seconde.ID, premiere.ID)
	}
	exigerElementDeListe(t, page.Elements[0], seconde, articles[1].Titre, statutEnAttente)
	exigerElementDeListe(t, page.Elements[1], premiere, articles[0].Titre, statutExpiree)
	t.Logf("liste de l'acheteur : %s (%s, %s) puis %s (%s, %s)", page.Elements[0].Titre, page.Elements[0].Statut, page.Elements[0].Echeance, page.Elements[1].Titre, page.Elements[1].Statut, page.Elements[1].Echeance)
}

func (b *bancReservation) us032TitreFige(t *testing.T) {
	vendeur := b.kc.creerVendeur(t)
	article := b.publierPlusieurs(t, vendeur, 1)[0]
	acheteur := b.connecterAcheteur(t)
	commande := b.commanderAvecSucces(t, acheteur.jeton, article.ID, http.StatusCreated)
	if commande.Titre != article.Titre {
		t.Errorf("titre à la création %q, %q attendu", commande.Titre, article.Titre)
	}
	if ligne := b.commandeEnBase(t, commande.ID); ligne["titre"] != article.Titre {
		t.Errorf("titre en base %v, %q attendu", ligne["titre"], article.Titre)
	}
	attendreLEcheance(t, commande.Echeance)
	nouveauTitre := "Titre réécrit " + texteAleatoire()[:6]
	b.modifier(t, bearer(b.kc.seConnecter(t, vendeur)), article.ID, fmt.Sprintf(`{"revision":%d,"titre":%q}`, article.Revision, nouveauTitre)).exiger(t, http.StatusOK)
	if detail := b.lireCommandeRecue(t, acheteur.jeton, commande.ID); detail.Titre != article.Titre {
		t.Errorf("détail après modification de l'annonce : titre %q, %q attendu", detail.Titre, article.Titre)
	}
	page := b.mesCommandes(t, acheteur.jeton, "")
	if len(page.Elements) != 1 || page.Elements[0].Titre != article.Titre {
		t.Errorf("liste après modification de l'annonce : %+v, titre %q attendu", page.Elements, article.Titre)
	}
}

func (b *bancReservation) us032ExpireePendantLaConsultation(t *testing.T) {
	article := b.publierPlusieurs(t, b.kc.creerVendeur(t), 1)[0]
	acheteur := b.connecterAcheteur(t)
	commande := b.commanderAvecSucces(t, acheteur.jeton, article.ID, http.StatusCreated)
	avant := b.mesCommandes(t, acheteur.jeton, "")
	exigerAvantLEcheance(t, commande.Echeance)("liste avant l'échéance")
	if len(avant.Elements) != 1 || avant.Elements[0].Statut != statutEnAttente {
		t.Fatalf("avant l'échéance : %+v, %s attendu", avant.Elements, statutEnAttente)
	}
	attendreLEcheance(t, commande.Echeance)
	apres := b.mesCommandes(t, acheteur.jeton, "")
	if len(apres.Elements) != 1 {
		t.Fatalf("après l'échéance : %+v", apres.Elements)
	}
	exigerElementDeListe(t, apres.Elements[0], commande, article.Titre, statutExpiree)
}

func (b *bancReservation) us032CloisonnementEntreAcheteurs(t *testing.T) {
	articles := b.publierPlusieurs(t, b.kc.creerVendeur(t), 2)
	a, acheteurB := b.connecterAcheteur(t), b.connecterAcheteur(t)
	deA := b.commanderAvecSucces(t, a.jeton, articles[0].ID, http.StatusCreated)
	if page := b.mesCommandes(t, acheteurB.jeton, ""); page.Total != 0 || len(page.Elements) != 0 {
		t.Errorf("B sans réservation voit %+v", page)
	}
	if page := b.mesCommandes(t, acheteurB.jeton, "?acheteurId="+a.compte.sujet); page.Total != 0 || len(page.Elements) != 0 {
		t.Errorf("B qui désigne A dans la requête voit %+v", page)
	}
	deB := b.commanderAvecSucces(t, acheteurB.jeton, articles[1].ID, http.StatusCreated)
	if page := b.mesCommandes(t, acheteurB.jeton, "?taille=50"); page.Total != 1 || !slices.Equal(identifiantsDe(page), []string{deB.ID}) {
		t.Errorf("B voit %v (total %d), seulement %s attendu", identifiantsDe(page), page.Total, deB.ID)
	}
	if page := b.mesCommandes(t, a.jeton, "?taille=50"); page.Total != 1 || !slices.Equal(identifiantsDe(page), []string{deA.ID}) {
		t.Errorf("A voit %v (total %d), seulement %s attendu", identifiantsDe(page), page.Total, deA.ID)
	}
	detailDeA := b.lireCommande(t, acheteurB.jeton, deA.ID).exiger(t, http.StatusNotFound)
	inexistante := b.lireCommande(t, acheteurB.jeton, uuid.NewString()).exiger(t, http.StatusNotFound)
	if !bytes.Equal(detailDeA.corps, inexistante.corps) {
		t.Errorf("404 du détail de A différent d'une commande inexistante :\n%s\n%s", detailDeA.corps, inexistante.corps)
	}
	for _, secret := range []string{articles[0].ID, articles[0].Titre, a.compte.sujet, deA.Echeance} {
		if strings.Contains(string(detailDeA.corps), secret) {
			t.Errorf("404 qui révèle %q : %s", secret, detailDeA.corps)
		}
	}
	if lue := b.lireCommandeRecue(t, a.jeton, deA.ID); lue.ID != deA.ID {
		t.Errorf("A ne relit pas sa commande : %+v", lue)
	}
	exigerAvantLEcheance(t, deA.Echeance)("lectures croisées")
}

func (b *bancReservation) us032SansAcheteurAuthentifie(t *testing.T) {
	service := b.kc.jetonDeTransactions(t)
	b.listerCommandes(t, "", "").exiger(t, http.StatusUnauthorized)
	b.listerCommandes(t, "pas-un-jeton", "").exiger(t, http.StatusUnauthorized)
	exigerRefusDAutorisation(t, "GET commandes avec le jeton de service", b.listerCommandes(t, service, ""))
}

func (b *bancReservation) amorcerCommandes(t *testing.T, acheteur string, nombre int) []commandeAmorcee {
	t.Helper()
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	amorcees := make([]commandeAmorcee, 0, nombre)
	for i := range nombre {
		c := commandeAmorcee{id: uuid.NewString(), creeLe: base.Add(time.Duration(i/3) * time.Second)}
		echeance := time.Now().Add(time.Hour)
		if i%2 == 0 {
			echeance = time.Now().Add(-time.Minute)
		}
		_, err := b.transactions.maitre.Exec(t.Context(), `INSERT INTO commande (id, reservation_id, article_id, acheteur_id, statut, devise, prix_centimes, frais_port_centimes, total_paye_centimes, commission_centimes, montant_vendeur_centimes, echeance, cree_le, titre)
			VALUES ($1, $2, $3, $4, 'en_attente_paiement', 'EUR', 8000, 500, 8500, 425, 8075, $5, $6, $7)`,
			c.id, uuid.NewString(), uuid.NewString(), acheteur, echeance, c.creeLe, fmt.Sprintf("Amorcée %02d", i))
		if err != nil {
			t.Fatalf("amorçage de la commande %d : %v", i, err)
		}
		amorcees = append(amorcees, c)
	}
	slices.SortFunc(amorcees, func(x, y commandeAmorcee) int {
		if c := y.creeLe.Compare(x.creeLe); c != 0 {
			return c
		}
		return strings.Compare(y.id, x.id)
	})
	return amorcees
}

func idsAmorces(amorcees []commandeAmorcee) []string {
	ids := make([]string, 0, len(amorcees))
	for _, c := range amorcees {
		ids = append(ids, c.id)
	}
	return ids
}

func (b *bancReservation) exigerPage(t *testing.T, jeton, requete string, numero, taille int, attendus []string) pageDeCommandes {
	t.Helper()
	lue := b.mesCommandes(t, jeton, requete)
	if lue.Page != numero || lue.Taille != taille || lue.Total != commandesAmorcees || !slices.Equal(identifiantsDe(lue), attendus) {
		t.Errorf("%q : page %d, taille %d, total %d, %v ; attendu : %d, %d, %d, %v", requete, lue.Page, lue.Taille, lue.Total, identifiantsDe(lue), numero, taille, commandesAmorcees, attendus)
	}
	return lue
}

func (b *bancReservation) parcourirUneParUne(t *testing.T, jeton string) []string {
	t.Helper()
	var ids []string
	for page := 1; page <= commandesAmorcees; page++ {
		lue := b.mesCommandes(t, jeton, fmt.Sprintf("?page=%d&taille=1", page))
		if len(lue.Elements) != 1 {
			t.Fatalf("page %d de taille 1 : %+v", page, lue)
		}
		ids = append(ids, lue.Elements[0].ID)
	}
	return ids
}

func exigerEtatsDerivesDeLEcheance(t *testing.T, page pageDeCommandes) {
	t.Helper()
	for _, c := range page.Elements {
		attendu := statutEnAttente
		if !instantRFC3339(t, c.Echeance).After(time.Now()) {
			attendu = statutExpiree
		}
		if c.Statut != attendu || !strings.HasPrefix(c.Titre, "Amorcée ") {
			t.Errorf("commande amorcée %s : état %q titre %q, état %q attendu (échéance %s)", c.ID, c.Statut, c.Titre, attendu, c.Echeance)
		}
		exigerMontant(t, "total amorcé", c.TotalPaye, 8500)
	}
}

func (b *bancReservation) us032PaginationEtTri(t *testing.T) {
	acheteur := b.connecterAcheteur(t)
	attendus := idsAmorces(b.amorcerCommandes(t, acheteur.compte.sujet, commandesAmorcees))
	b.amorcerCommandes(t, b.kc.creerAcheteur(t).sujet, 3)
	b.exigerPage(t, acheteur.jeton, "", 1, tailleDeListeParDefaut, attendus[:tailleDeListeParDefaut])
	b.exigerPage(t, acheteur.jeton, "?page=2", 2, tailleDeListeParDefaut, attendus[tailleDeListeParDefaut:])
	b.exigerPage(t, acheteur.jeton, "?page=3", 3, tailleDeListeParDefaut, []string{})
	b.exigerPage(t, acheteur.jeton, "?page=10000&taille=50", 10000, 50, []string{})
	toutes := b.exigerPage(t, acheteur.jeton, "?taille=50", 1, 50, attendus)
	if uneParUne := b.parcourirUneParUne(t, acheteur.jeton); !slices.Equal(uneParUne, attendus) {
		t.Errorf("parcours une par une : %v, %v attendu", uneParUne, attendus)
	}
	exigerEtatsDerivesDeLEcheance(t, toutes)
	t.Logf("%d commandes amorcées (créations par groupes de 3 à la même seconde), tri creeLe puis id décroissants vérifié page par page", commandesAmorcees)
}

func (b *bancReservation) us032PaginationHorsBornes(t *testing.T) {
	acheteur := b.connecterAcheteur(t)
	for _, requete := range []string{"?page=0", "?page=10001", "?taille=0", "?taille=51", "?page=abc", "?taille=", "?page=1&page=2"} {
		t.Run(requete, func(t *testing.T) {
			exigerProbleme(t, b.listerCommandes(t, acheteur.jeton, requete), http.StatusBadRequest, typeSaisieInvalideTx)
		})
	}
}

func (b *bancReservation) us032ReserverRenvoieLeTitre(t *testing.T) {
	article := b.publierPlusieurs(t, b.kc.creerVendeur(t), 1)[0]
	acheteur := b.connecterAcheteur(t)
	var reservation reservationRecue
	b.reserverEnDirect(t, article.ID, entetesDeService(b.kc.jetonDeTransactions(t), acheteur.jeton)).exiger(t, http.StatusCreated).lire(t, &reservation)
	if reservation.Titre != article.Titre {
		t.Errorf("titre renvoyé par réserver %q, %q attendu", reservation.Titre, article.Titre)
	}
	var reprise reservationRecue
	b.reserverEnDirect(t, article.ID, entetesDeService(b.kc.jetonDeTransactions(t), acheteur.jeton)).exiger(t, http.StatusOK).lire(t, &reprise)
	if reprise.Titre != article.Titre {
		t.Errorf("titre renvoyé à la reprise %q, %q attendu", reprise.Titre, article.Titre)
	}
	commande := b.commanderAvecSucces(t, acheteur.jeton, article.ID, http.StatusCreated)
	if commande.Titre != article.Titre {
		t.Errorf("titre de la commande %q, %q attendu", commande.Titre, article.Titre)
	}
}

func (b *bancReservation) us032Contrats(t *testing.T) {
	b.mu.Lock()
	tx, internes := append([]echange(nil), b.echangesTx...), append([]echange(nil), b.echangesInt...)
	b.mu.Unlock()
	routeurTx := chargerContratDe(t, "http://"+b.tx.api, filepath.Join("transactions", "v1.openapi.yaml"), "/transactions/v1")
	listes := 0
	for _, e := range tx {
		if e.requete.Method == http.MethodGet && e.requete.URL.Path == cheminCommandes {
			listes++
		}
		verifierEchangeContrat(t, routeurTx, e)
	}
	routeurInterne := chargerContratDe(t, b.interne, filepath.Join("place-de-marche", "interne-v1.openapi.yaml"), "/interne/v1")
	for _, e := range internes {
		verifierEchangeContrat(t, routeurInterne, e)
	}
	if listes == 0 {
		t.Fatalf("aucun échange de liste vérifié")
	}
	t.Logf("Transactions : %d échanges vérifiés dont %d listes, statuts %v ; réserver interne : %d échanges, statuts %v", len(tx), listes, statutsDe(tx), len(internes), statutsDe(internes))
}
