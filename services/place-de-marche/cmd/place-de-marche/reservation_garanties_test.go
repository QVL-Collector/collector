package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	dureeCourte            = 6 * time.Second
	margeApresEcheance     = 700 * time.Millisecond
	typeArticleReserve     = "urn:collector:probleme:article-reserve"
	statutExpiree          = "expiree"
	rafaleDuMemeAcheteur   = 12
	toursDeRafale          = 3
	toursDeDeuxArticles    = 3
	requetesParArticle     = 6
	toursDeCourseVendeur   = 12
	toursDeCourseDirecte   = 12
	acheteursParCourse     = 2
	pasDeDecalageVendeur   = 8 * time.Millisecond
	pasDeDecalageDirect    = 2 * time.Millisecond
	prixDeReference        = 8000
	fraisDeReference       = 500
	corpsDeModificationVol = `"titre":"Prix cassé","description":"Modifiée pendant la réservation","categorieId":%q,"prix":{"montantCentimes":100,"devise":"EUR"},"fraisDePort":{"montantCentimes":0,"devise":"EUR"}`
)

var tailleDesToursDeConcurrence = []int{2, 8, 16, 16, 20, 20}

type resultatParallele struct {
	reponse reponse
	err     error
}

type acheteurConnecte struct {
	compte compteKeycloak
	jeton  string
}

type problemeRecu struct {
	Type   string `json:"type"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

func TestUS031GarantiesDeLaReservation(t *testing.T) {
	b := demarrerBancReservationAvec(t, dureeCourte, dureeCourte)
	t.Run("AC1 acheteurs concurrents par Transactions : une seule réservation, les autres 409 article-deja-reserve, une seule commande", b.ac1ConcurrenceParTransactions)
	t.Run("AC1 acheteurs concurrents directement sur réserver : une seule réservation active", b.ac1ConcurrenceSurReserver)
	t.Run("AC2 rafale parallèle du même acheteur : même réservation, même échéance, une seule commande", b.ac2RafaleDuMemeAcheteur)
	t.Run("AC3 réservation active sur A, réserver B en séquence : 409 reservation-deja-active", b.ac3SecondArticleEnSequence)
	t.Run("AC3 même acheteur sur deux articles en parallèle : un seul article réservé", b.ac3DeuxArticlesEnParallele)
	t.Run("AC3 limite : réservation expirée, plus active, un autre article est réservable", b.ac3ApresEcheanceAutreArticleReservable)
	t.Run("AC4 échéance passée : article disponible, commande expirée, nouvel essai = nouvelle réservation et nouvelle commande", b.ac4EcheancePasseeMemeAcheteur)
	t.Run("AC4 échéance passée : un autre acheteur réserve, l'ancienne commande reste expirée", b.ac4EcheancePasseeAutreAcheteur)
	t.Run("AC5 article réservé : modification et soumission du vendeur refusées en 409 article-reserve, prix figé", b.ac5ArticleReserveFige)
	t.Run("AC5 article vendu : modification et soumission refusées en 409 article-reserve", b.ac5ArticleVenduFige)
	t.Run("AC5 course vendeur qui modifie contre acheteurs qui réservent : jamais les deux succès", b.ac5CourseVendeurContreAcheteurs)
	t.Run("AC5 course vendeur qui modifie contre réserver en direct : jamais les deux succès", b.ac5CourseVendeurContreReserverDirect)
	t.Run("Contrats OpenAPI : Transactions (statut expiree), réserver, catalogue, routes du vendeur", b.contratsUS031)
}

func (b *bancReservation) connecterAcheteur(t *testing.T) acheteurConnecte {
	t.Helper()
	compte := b.kc.creerAcheteur(t)
	return acheteurConnecte{compte: compte, jeton: b.kc.seConnecter(t, compte)}
}

func (b *bancReservation) connecterAcheteurs(t *testing.T, nombre int) []acheteurConnecte {
	t.Helper()
	acheteurs := make([]acheteurConnecte, nombre)
	for i := range acheteurs {
		acheteurs[i] = b.connecterAcheteur(t)
	}
	return acheteurs
}

func (b *bancReservation) publierPlusieurs(t *testing.T, vendeur compteKeycloak, nombre int) []articleRecu {
	t.Helper()
	jeton := bearer(b.kc.seConnecter(t, vendeur))
	soumis := make([]articleRecu, nombre)
	for i := range soumis {
		corps := corpsComplet(fmt.Sprintf("Monnaie rare %d %s", i, texteAleatoire()[:6]), categorieDeLaReservaton, prixDeReference, fraisDeReference)
		soumis[i] = b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corps))
	}
	publies := make([]articleRecu, nombre)
	for i, article := range soumis {
		attendre(t, delaiDePublicationResa, "publication de "+article.Titre, func() bool {
			publies[i] = b.relire(t, jeton, article.ID)
			return publies[i].Etat == etatPublie
		})
	}
	return publies
}

func (b *bancReservation) requete(t *testing.T, base, methode, chemin string, corps []byte, entetes http.Header) *http.Request {
	t.Helper()
	requete, err := http.NewRequestWithContext(t.Context(), methode, base+chemin, bytes.NewReader(corps))
	if err != nil {
		t.Fatalf("requête %s %s : %v", methode, chemin, err)
	}
	requete.Header = entetes.Clone()
	if corps != nil {
		requete.Header.Set("Content-Type", typeJSON)
	}
	return requete
}

func (b *bancReservation) requeteDeCommande(t *testing.T, jeton, articleID string) *http.Request {
	t.Helper()
	return b.requete(t, "http://"+b.tx.api, http.MethodPost, cheminCommandes, []byte(corpsDeCommande(articleID)), avecJeton(jeton))
}

func (b *bancReservation) requeteDeReservationDirecte(t *testing.T, jetonService, jetonAcheteur, articleID string) *http.Request {
	t.Helper()
	return b.requete(t, b.interne, http.MethodPost, cheminDeReservation(articleID), nil, entetesDeService(jetonService, jetonAcheteur))
}

func (b *bancReservation) requeteDeModification(t *testing.T, jetonVendeur, articleID string, revision int) *http.Request {
	t.Helper()
	corps := fmt.Sprintf("{\"revision\":%d,"+corpsDeModificationVol+"}", revision, categorieDeLaReservaton)
	return b.requete(t, b.public, http.MethodPut, cheminArticles+"/"+articleID, []byte(corps), http.Header{"Authorization": {jetonVendeur}})
}

func lireCorps(requete *http.Request) []byte {
	if requete.GetBody == nil {
		return nil
	}
	lecteur, err := requete.GetBody()
	if err != nil {
		return nil
	}
	contenu, _ := io.ReadAll(lecteur)
	return contenu
}

func executerSansT(client *http.Client, requete *http.Request) resultatParallele {
	brute, err := client.Do(requete)
	if err != nil {
		return resultatParallele{err: err}
	}
	defer func() { _ = brute.Body.Close() }()
	corps, err := io.ReadAll(brute.Body)
	return resultatParallele{reponse: reponse{statut: brute.StatusCode, entetes: brute.Header, corps: corps}, err: err}
}

func (b *bancReservation) enParallele(t *testing.T, requetes []*http.Request) []reponse {
	t.Helper()
	return b.enParalleleDecale(t, requetes, make([]time.Duration, len(requetes)))
}

func (b *bancReservation) enParalleleDecale(t *testing.T, requetes []*http.Request, decalages []time.Duration) []reponse {
	t.Helper()
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: len(requetes)}}
	t.Cleanup(client.CloseIdleConnections)
	corps := make([][]byte, len(requetes))
	for i, requete := range requetes {
		corps[i] = lireCorps(requete)
	}
	resultats := make([]resultatParallele, len(requetes))
	depart := make(chan struct{})
	var prets, finis sync.WaitGroup
	for i, requete := range requetes {
		prets.Add(1)
		finis.Add(1)
		go func() {
			defer finis.Done()
			prets.Done()
			<-depart
			time.Sleep(decalages[i])
			resultats[i] = executerSansT(client, requete)
		}()
	}
	prets.Wait()
	close(depart)
	finis.Wait()
	reponses := make([]reponse, len(requetes))
	for i, r := range resultats {
		if r.err != nil {
			t.Fatalf("%s %s : %v", requetes[i].Method, requetes[i].URL.Path, r.err)
		}
		requetes[i].Body = io.NopCloser(bytes.NewReader(corps[i]))
		b.archiver(echange{requete: requetes[i], reponse: r.reponse})
		reponses[i] = r.reponse
	}
	return reponses
}

func (b *bancReservation) archiver(ech echange) {
	adresse := ech.requete.URL.String()
	if strings.HasPrefix(adresse, b.public) {
		b.environnement.mu.Lock()
		b.echanges = append(b.echanges, ech)
		b.environnement.mu.Unlock()
		return
	}
	b.consigner(strings.HasPrefix(adresse, b.interne), ech)
}

func lireProbleme(t *testing.T, r reponse) problemeRecu {
	t.Helper()
	if !strings.HasPrefix(r.entetes.Get("Content-Type"), typeProbleme) {
		t.Fatalf("réponse %d sans Problem Details (%q) : %s", r.statut, r.entetes.Get("Content-Type"), r.corps)
	}
	var p problemeRecu
	r.lire(t, &p)
	return p
}

func exigerProbleme(t *testing.T, r reponse, statut int, typeAttendu string) problemeRecu {
	t.Helper()
	if r.statut != statut {
		t.Fatalf("statut %d attendu (%s), %d reçu : %s", statut, typeAttendu, r.statut, r.corps)
	}
	p := lireProbleme(t, r)
	if p.Type != typeAttendu || p.Status != statut {
		t.Fatalf("problème %+v, %s attendu", p, typeAttendu)
	}
	return p
}

func gagnantUnique(t *testing.T, reponses []reponse, statutDeSucces int, typeDesPerdants string) int {
	t.Helper()
	gagnant := -1
	perdants := 0
	for i, r := range reponses {
		if r.statut == statutDeSucces {
			if gagnant >= 0 {
				t.Fatalf("deux succès : demandes %d et %d", gagnant, i)
			}
			gagnant = i
			continue
		}
		exigerProbleme(t, r, http.StatusConflict, typeDesPerdants)
		perdants++
	}
	if gagnant < 0 {
		t.Fatalf("aucune demande acceptée sur %d", len(reponses))
	}
	if perdants != len(reponses)-1 {
		t.Fatalf("%d refus, %d attendus", perdants, len(reponses)-1)
	}
	return gagnant
}

func (b *bancReservation) compterReservationsActives(t *testing.T, filtre string, argument any) int {
	t.Helper()
	var n int
	if err := b.marche.maitre.QueryRow(t.Context(), "SELECT count(*) FROM reservation WHERE echeance > now() AND "+filtre, argument).Scan(&n); err != nil {
		t.Fatalf("comptage des réservations actives : %v", err)
	}
	return n
}

func (b *bancReservation) exigerUneSeuleReservationPour(t *testing.T, articleID, acheteur string) {
	t.Helper()
	if n := b.compterReservations(t, articleID); n != 1 {
		t.Errorf("%d lignes de réservation pour l'article, 1 attendue", n)
	}
	if n := b.compterReservationsActives(t, "article_id = $1", articleID); n != 1 {
		t.Errorf("%d réservations actives pour l'article, 1 attendue", n)
	}
	if n := b.compterCommandes(t, "WHERE article_id = $1", articleID); n != 1 {
		t.Errorf("%d commandes pour l'article, 1 attendue", n)
	}
	if acheteur == "" {
		return
	}
	if n := b.compterCommandes(t, "WHERE article_id = $1 AND acheteur_id = $2", articleID, acheteur); n != 1 {
		t.Errorf("commande de l'article absente pour l'acheteur retenu %s", acheteur)
	}
	var reservePour string
	if err := b.marche.maitre.QueryRow(t.Context(), `SELECT r.acheteur_id::text FROM article a JOIN reservation r ON r.id = a.reservation_id WHERE a.id = $1`, articleID).Scan(&reservePour); err != nil {
		t.Fatalf("réservation courante de l'article : %v", err)
	}
	if reservePour != acheteur {
		t.Errorf("article réservé pour %s, attendu pour l'acheteur accepté %s", reservePour, acheteur)
	}
}

func (b *bancReservation) ac1ConcurrenceParTransactions(t *testing.T) {
	articles := b.publierPlusieurs(t, b.kc.creerVendeur(t), len(tailleDesToursDeConcurrence))
	reserve := b.connecterAcheteurs(t, 20)
	for tour, taille := range tailleDesToursDeConcurrence {
		article := articles[tour]
		acheteurs := reserve[:taille]
		requetes := make([]*http.Request, taille)
		for i, a := range acheteurs {
			requetes[i] = b.requeteDeCommande(t, a.jeton, article.ID)
		}
		debut := time.Now()
		reponses := b.enParallele(t, requetes)
		ecoule := time.Since(debut)
		gagnant := gagnantUnique(t, reponses, http.StatusCreated, typeDejaReserve)
		var commande commandeRecue
		reponses[gagnant].lire(t, &commande)
		if commande.Statut != statutEnAttente || commande.ArticleID != article.ID {
			t.Fatalf("tour %d : commande gagnante %+v", tour+1, commande)
		}
		active := exigerAvantLEcheance(t, commande.Echeance)
		b.exigerUneSeuleReservationPour(t, article.ID, acheteurs[gagnant].compte.sujet)
		if !b.fiche(t, article.ID).estReserve() {
			t.Errorf("tour %d : article non marqué réservé", tour+1)
		}
		active(fmt.Sprintf("tour %d : relectures en base et fiche", tour+1))
		t.Logf("tour %d : %d acheteurs simultanés en %s, 1 × 201 (acheteur %d), %d × 409 article-deja-reserve", tour+1, taille, ecoule.Round(time.Millisecond), gagnant, taille-1)
		reserve[gagnant] = b.connecterAcheteur(t)
	}
}

func (e entreeReservable) estReserve() bool {
	return e.Reserve != nil && *e.Reserve
}

func (b *bancReservation) ac1ConcurrenceSurReserver(t *testing.T) {
	articles := b.publierPlusieurs(t, b.kc.creerVendeur(t), 3)
	acheteurs := b.connecterAcheteurs(t, 16)
	for tour, article := range articles {
		jetonService := b.kc.jetonDeTransactions(t)
		requetes := make([]*http.Request, len(acheteurs))
		for i, a := range acheteurs {
			requetes[i] = b.requeteDeReservationDirecte(t, jetonService, a.jeton, article.ID)
		}
		reponses := b.enParallele(t, requetes)
		gagnant := gagnantUnique(t, reponses, http.StatusCreated, typeDejaReserve)
		var reservation reservationRecue
		reponses[gagnant].lire(t, &reservation)
		etat := b.reservationEnBase(t, article.ID)
		if etat.reservationID == nil || *etat.reservationID != reservation.ReservationID {
			t.Errorf("tour %d : article réservé par %v, réservation acceptée %s", tour+1, etat.reservationID, reservation.ReservationID)
		}
		if n := b.compterReservations(t, article.ID); n != 1 {
			t.Errorf("tour %d : %d lignes de réservation, 1 attendue", tour+1, n)
		}
		if n := b.compterReservationsActives(t, "article_id = $1", article.ID); n != 1 {
			t.Errorf("tour %d : %d réservations actives, 1 attendue", tour+1, n)
		}
		exigerAvantLEcheance(t, reservation.Echeance)(fmt.Sprintf("tour %d direct : relectures en base", tour+1))
		t.Logf("tour %d direct : 16 acheteurs, 1 × 201, 15 × 409 article-deja-reserve", tour+1)
		acheteurs[gagnant] = b.connecterAcheteur(t)
	}
}

func (b *bancReservation) ac2RafaleDuMemeAcheteur(t *testing.T) {
	articles := b.publierPlusieurs(t, b.kc.creerVendeur(t), toursDeRafale)
	for tour, article := range articles {
		t.Run(fmt.Sprintf("tour %d", tour+1), func(t *testing.T) { b.rafaleSurUnArticle(t, article) })
	}
}

func (b *bancReservation) rafaleSurUnArticle(t *testing.T, article articleRecu) {
	acheteur := b.connecterAcheteur(t)
	requetes := make([]*http.Request, rafaleDuMemeAcheteur)
	for i := range requetes {
		requetes[i] = b.requeteDeCommande(t, acheteur.jeton, article.ID)
	}
	reference, crees := exigerMemeCommande(t, b.enParallele(t, requetes))
	active := exigerAvantLEcheance(t, reference.Echeance)
	b.exigerUneSeuleReservationPour(t, article.ID, acheteur.compte.sujet)
	active("rafale parallèle et relecture en base")
	time.Sleep(1500 * time.Millisecond)
	relance := b.commander(t, acheteur.jeton, corpsDeCommande(article.ID))
	active("relance après 1,5 s")
	var relue commandeRecue
	relance.exiger(t, http.StatusOK).lire(t, &relue)
	exigerMemeIdEtEcheance(t, "relance après 1,5 s", relue, reference)
	b.exigerRafaleDirecteMemeEcheance(t, acheteur, article.ID, reference.Echeance, active)
	b.exigerUneSeuleReservationPour(t, article.ID, acheteur.compte.sujet)
	ligne := b.commandeEnBase(t, reference.ID)
	etat := b.reservationEnBase(t, article.ID)
	if etat.reservationID == nil || ligne["reservation_id"] != *etat.reservationID {
		t.Errorf("commande sur la réservation %v, article réservé par %v", ligne["reservation_id"], etat.reservationID)
	}
	active("relectures finales en base")
	t.Logf("%d demandes parallèles du même acheteur → %d × 201, %d × 200, une commande %s, échéance %s ; relance et %d appels directs : même échéance", rafaleDuMemeAcheteur, crees, rafaleDuMemeAcheteur-crees, reference.ID, reference.Echeance, rafaleDuMemeAcheteur)
}

func (b *bancReservation) exigerRafaleDirecteMemeEcheance(t *testing.T, acheteur acheteurConnecte, articleID, echeance string, active garde) {
	t.Helper()
	jetonService := b.kc.jetonDeTransactions(t)
	directes := make([]*http.Request, rafaleDuMemeAcheteur)
	for i := range directes {
		directes[i] = b.requeteDeReservationDirecte(t, jetonService, acheteur.jeton, articleID)
	}
	reponses := b.enParallele(t, directes)
	active("rafale directe sur réserver")
	for i, r := range reponses {
		if r.statut != http.StatusOK && r.statut != http.StatusCreated {
			t.Fatalf("rafale directe %d : statut %d : %s", i, r.statut, r.corps)
		}
		var reservation reservationRecue
		r.lire(t, &reservation)
		if !instantRFC3339(t, reservation.Echeance).Equal(instantRFC3339(t, echeance)) {
			t.Errorf("rafale directe %d : échéance %s, %s attendue", i, reservation.Echeance, echeance)
		}
	}
}

func exigerMemeCommande(t *testing.T, reponses []reponse) (commandeRecue, int) {
	t.Helper()
	var reference commandeRecue
	crees := 0
	for i, r := range reponses {
		if r.statut != http.StatusOK && r.statut != http.StatusCreated {
			t.Fatalf("demande %d du même acheteur : statut %d : %s", i, r.statut, r.corps)
		}
		if r.statut == http.StatusCreated {
			crees++
		}
		var commande commandeRecue
		r.lire(t, &commande)
		if i == 0 {
			reference = commande
			continue
		}
		exigerMemeIdEtEcheance(t, fmt.Sprintf("demande %d", i), commande, reference)
	}
	if crees > 1 {
		t.Logf("%d réponses 201 pour la même commande", crees)
	}
	return reference, crees
}

func exigerMemeIdEtEcheance(t *testing.T, quoi string, commande, reference commandeRecue) {
	t.Helper()
	if commande.ID != reference.ID || !instantRFC3339(t, commande.Echeance).Equal(instantRFC3339(t, reference.Echeance)) {
		t.Errorf("%s : commande %s échéance %s, attendu %s échéance %s", quoi, commande.ID, commande.Echeance, reference.ID, reference.Echeance)
	}
}

func (b *bancReservation) exigerArticleLibre(t *testing.T, articleID string) {
	t.Helper()
	if b.fiche(t, articleID).estReserve() || b.entreeDuCatalogue(t, articleID).estReserve() {
		t.Errorf("article %s marqué réservé", articleID)
	}
}

func (b *bancReservation) exigerArticleReserve(t *testing.T, articleID string) {
	t.Helper()
	if !b.fiche(t, articleID).estReserve() || !b.entreeDuCatalogue(t, articleID).estReserve() {
		t.Errorf("article %s non marqué réservé", articleID)
	}
}

func (b *bancReservation) ac3SecondArticleEnSequence(t *testing.T) {
	articles := b.publierPlusieurs(t, b.kc.creerVendeur(t), 2)
	a, autre := articles[0], articles[1]
	acheteur := b.connecterAcheteur(t)
	premiere := b.commanderAvecSucces(t, acheteur.jeton, a.ID, http.StatusCreated)
	active := exigerAvantLEcheance(t, premiere.Echeance)
	parTransactions := b.commander(t, acheteur.jeton, corpsDeCommande(autre.ID))
	enDirect := b.reserverEnDirect(t, autre.ID, entetesDeService(b.kc.jetonDeTransactions(t), acheteur.jeton))
	active("second article par Transactions et en direct")
	exigerProbleme(t, parTransactions, http.StatusConflict, typeReservationActive)
	exigerProbleme(t, enDirect, http.StatusConflict, typeReservationActive)
	b.exigerArticleLibre(t, autre.ID)
	if etat := b.reservationEnBase(t, autre.ID); etat.reservationID != nil {
		t.Errorf("second article réservé en base : %v", *etat.reservationID)
	}
	if n := b.compterCommandes(t, "WHERE acheteur_id = $1", acheteur.compte.sujet); n != 1 {
		t.Errorf("%d commandes pour l'acheteur, 1 attendue", n)
	}
	if n := b.compterCommandes(t, "WHERE article_id = $1", autre.ID); n != 0 {
		t.Errorf("%d commandes sur le second article, 0 attendue", n)
	}
	var relue commandeRecue
	b.lireCommande(t, acheteur.jeton, premiere.ID).exiger(t, http.StatusOK).lire(t, &relue)
	exigerMemeIdEtEcheance(t, "première commande après le refus", relue, premiere)
	active("relectures après le refus")
	tiers := b.connecterAcheteur(t)
	b.commanderAvecSucces(t, tiers.jeton, autre.ID, http.StatusCreated)
}

func (b *bancReservation) ac3DeuxArticlesEnParallele(t *testing.T) {
	articles := b.publierPlusieurs(t, b.kc.creerVendeur(t), 2*toursDeDeuxArticles)
	for tour := range toursDeDeuxArticles {
		t.Run(fmt.Sprintf("tour %d", tour+1), func(t *testing.T) {
			b.deuxArticlesEnParallele(t, articles[2*tour].ID, articles[2*tour+1].ID)
		})
	}
}

func repartirParArticle(t *testing.T, reponses []reponse, ids [2]string) (map[string]int, int) {
	t.Helper()
	retenus := map[string]int{}
	refus := 0
	for i, r := range reponses {
		if r.statut == http.StatusOK || r.statut == http.StatusCreated {
			retenus[ids[i%2]]++
			continue
		}
		exigerProbleme(t, r, http.StatusConflict, typeReservationActive)
		refus++
	}
	return retenus, refus
}

func echeanceRetenue(t *testing.T, reponses []reponse) string {
	t.Helper()
	for _, r := range reponses {
		if r.statut == http.StatusOK || r.statut == http.StatusCreated {
			var commande commandeRecue
			r.lire(t, &commande)
			return commande.Echeance
		}
	}
	t.Fatalf("aucune demande acceptée sur %d", len(reponses))
	return ""
}

func (b *bancReservation) deuxArticlesEnParallele(t *testing.T, a, autre string) {
	acheteur := b.connecterAcheteur(t)
	requetes := make([]*http.Request, 0, 2*requetesParArticle)
	for range requetesParArticle {
		requetes = append(requetes, b.requeteDeCommande(t, acheteur.jeton, a), b.requeteDeCommande(t, acheteur.jeton, autre))
	}
	reponses := b.enParallele(t, requetes)
	active := exigerAvantLEcheance(t, echeanceRetenue(t, reponses))
	active("demandes parallèles sur deux articles")
	retenus, refus := repartirParArticle(t, reponses, [2]string{a, autre})
	if len(retenus) != 1 {
		t.Fatalf("articles acceptés %v, un seul attendu", retenus)
	}
	if n := b.compterCommandes(t, "WHERE acheteur_id = $1", acheteur.compte.sujet); n != 1 {
		t.Errorf("%d commandes pour l'acheteur, 1 attendue", n)
	}
	if n := b.compterReservationsActives(t, "acheteur_id = $1", acheteur.compte.sujet); n != 1 {
		t.Errorf("%d réservations actives pour l'acheteur, 1 attendue", n)
	}
	retenu, perdant := a, autre
	if retenus[a] == 0 {
		retenu, perdant = autre, a
	}
	b.exigerArticleReserve(t, retenu)
	b.exigerArticleLibre(t, perdant)
	active("relectures en base, fiches et catalogue")
	t.Logf("%d demandes parallèles sur deux articles → un article retenu (%v), %d × 409 reservation-deja-active", len(requetes), retenus, refus)
}

func attendreLEcheance(t *testing.T, echeance string) {
	t.Helper()
	time.Sleep(time.Until(instantRFC3339(t, echeance).Add(margeApresEcheance)))
}

func (b *bancReservation) ac3ApresEcheanceAutreArticleReservable(t *testing.T) {
	articles := b.publierPlusieurs(t, b.kc.creerVendeur(t), 2)
	acheteur := b.connecterAcheteur(t)
	premiere := b.commanderAvecSucces(t, acheteur.jeton, articles[0].ID, http.StatusCreated)
	refus := b.commander(t, acheteur.jeton, corpsDeCommande(articles[1].ID))
	exigerAvantLEcheance(t, premiere.Echeance)("second article avant l'échéance")
	exigerProbleme(t, refus, http.StatusConflict, typeReservationActive)
	attendreLEcheance(t, premiere.Echeance)
	seconde := b.commanderAvecSucces(t, acheteur.jeton, articles[1].ID, http.StatusCreated)
	if seconde.Statut != statutEnAttente {
		t.Errorf("commande du second article %s, %s attendu", seconde.Statut, statutEnAttente)
	}
}

func (b *bancReservation) lireCommandeRecue(t *testing.T, jeton, id string) commandeRecue {
	t.Helper()
	var commande commandeRecue
	b.lireCommande(t, jeton, id).exiger(t, http.StatusOK).lire(t, &commande)
	return commande
}

func exigerCommandeExpiree(t *testing.T, lue, origine commandeRecue) {
	t.Helper()
	if lue.Statut != statutExpiree {
		t.Errorf("commande %s : statut %q après l'échéance, %q attendu", lue.ID, lue.Statut, statutExpiree)
	}
	exigerMemeIdEtEcheance(t, "commande expirée", lue, origine)
	exigerMontant(t, "prix de la commande expirée", lue.Prix, origine.Prix.MontantCentimes)
	exigerMontant(t, "total de la commande expirée", lue.TotalPaye, origine.TotalPaye.MontantCentimes)
}

func (b *bancReservation) ac4EcheancePasseeMemeAcheteur(t *testing.T) {
	article := b.publierPlusieurs(t, b.kc.creerVendeur(t), 1)[0]
	acheteur := b.connecterAcheteur(t)
	premiere := b.commanderAvecSucces(t, acheteur.jeton, article.ID, http.StatusCreated)
	ancienne := b.reservationEnBase(t, article.ID)
	duree := instantRFC3339(t, premiere.Echeance).Sub(instantRFC3339(t, premiere.CreeLe))
	if duree <= 0 || duree > dureeCourte+toleranceEcheance {
		t.Errorf("échéance à %s de la création, %s attendus", duree, dureeCourte)
	}
	if lue := b.lireCommandeRecue(t, acheteur.jeton, premiere.ID); lue.Statut != statutEnAttente {
		t.Errorf("avant l'échéance : statut %q, %q attendu", lue.Statut, statutEnAttente)
	}
	b.exigerArticleReserve(t, article.ID)
	exigerAvantLEcheance(t, premiere.Echeance)("lectures avant l'échéance")
	attendreLEcheance(t, premiere.Echeance)
	b.exigerArticleLibre(t, article.ID)
	exigerCommandeExpiree(t, b.lireCommandeRecue(t, acheteur.jeton, premiere.ID), premiere)
	nouvelle := b.commanderAvecSucces(t, acheteur.jeton, article.ID, http.StatusCreated)
	if nouvelle.ID == premiere.ID || nouvelle.Statut != statutEnAttente {
		t.Errorf("nouvel essai : commande %s (%s), une nouvelle commande en attente de paiement attendue", nouvelle.ID, nouvelle.Statut)
	}
	if !instantRFC3339(t, nouvelle.Echeance).After(instantRFC3339(t, premiere.Echeance)) {
		t.Errorf("nouvelle échéance %s non postérieure à %s", nouvelle.Echeance, premiere.Echeance)
	}
	if ligne := b.commandeEnBase(t, nouvelle.ID); ancienne.reservationID == nil || ligne["reservation_id"] == *ancienne.reservationID {
		t.Errorf("nouvelle commande sur l'ancienne réservation %v", ligne["reservation_id"])
	}
	b.exigerArticleReserve(t, article.ID)
	exigerCommandeExpiree(t, b.lireCommandeRecue(t, acheteur.jeton, premiere.ID), premiere)
	if n := b.compterCommandes(t, "WHERE article_id = $1", article.ID); n != 2 {
		t.Errorf("%d commandes pour l'article, 2 attendues", n)
	}
	if n := b.compterReservationsActives(t, "article_id = $1", article.ID); n != 1 {
		t.Errorf("%d réservations actives, 1 attendue", n)
	}
	b.lireCommande(t, b.connecterAcheteur(t).jeton, premiere.ID).exiger(t, http.StatusNotFound)
}

func (b *bancReservation) ac4EcheancePasseeAutreAcheteur(t *testing.T) {
	article := b.publierPlusieurs(t, b.kc.creerVendeur(t), 1)[0]
	premier, second := b.connecterAcheteur(t), b.connecterAcheteur(t)
	commande := b.commanderAvecSucces(t, premier.jeton, article.ID, http.StatusCreated)
	avantEcheance := b.commander(t, second.jeton, corpsDeCommande(article.ID))
	exigerAvantLEcheance(t, commande.Echeance)("second acheteur avant l'échéance")
	exigerProbleme(t, avantEcheance, http.StatusConflict, typeDejaReserve)
	attendreLEcheance(t, commande.Echeance)
	b.exigerArticleLibre(t, article.ID)
	reprise := b.commanderAvecSucces(t, second.jeton, article.ID, http.StatusCreated)
	if reprise.Statut != statutEnAttente {
		t.Errorf("commande du second acheteur %s", reprise.Statut)
	}
	exigerCommandeExpiree(t, b.lireCommandeRecue(t, premier.jeton, commande.ID), commande)
	exigerProbleme(t, b.commander(t, premier.jeton, corpsDeCommande(article.ID)), http.StatusConflict, typeDejaReserve)
	b.exigerUneSeuleReservationActivePour(t, article.ID, second.compte.sujet)
}

func (b *bancReservation) exigerUneSeuleReservationActivePour(t *testing.T, articleID, acheteur string) {
	t.Helper()
	if n := b.compterReservationsActives(t, "article_id = $1", articleID); n != 1 {
		t.Errorf("%d réservations actives, 1 attendue", n)
	}
	var reservePour string
	if err := b.marche.maitre.QueryRow(t.Context(), `SELECT r.acheteur_id::text FROM article a JOIN reservation r ON r.id = a.reservation_id WHERE a.id = $1`, articleID).Scan(&reservePour); err != nil {
		t.Fatalf("réservation courante : %v", err)
	}
	if reservePour != acheteur {
		t.Errorf("article réservé pour %s, attendu %s", reservePour, acheteur)
	}
}

type articleFige struct {
	Titre       string `json:"titre"`
	Description string `json:"description"`
	Categorie   string `json:"categorie_id"`
	Prix        int64  `json:"prix_centimes"`
	Frais       int64  `json:"frais_port_centimes"`
	Revision    int    `json:"revision"`
	Etat        string `json:"etat"`
	ModifieLe   string `json:"modifie_le"`
}

func (b *bancReservation) articleEnBase(t *testing.T, id string) (articleFige, string) {
	t.Helper()
	var brut []byte
	if err := b.marche.maitre.QueryRow(t.Context(), `SELECT to_jsonb(a) FROM article a WHERE id = $1`, id).Scan(&brut); err != nil {
		t.Fatalf("article %s en base : %v", id, err)
	}
	var fige articleFige
	reponse{corps: brut}.lire(t, &fige)
	return fige, string(brut)
}

func (b *bancReservation) exigerArticleInchange(t *testing.T, id string, avant articleFige, ligneAvant string) {
	t.Helper()
	apres, ligneApres := b.articleEnBase(t, id)
	if apres != avant {
		t.Errorf("article modifié malgré le refus :\navant %+v\naprès %+v", avant, apres)
	}
	if ligneApres != ligneAvant {
		t.Errorf("ligne de l'article réécrite malgré le refus :\navant %s\naprès %s", ligneAvant, ligneApres)
	}
}

func (b *bancReservation) refusArticleReserve(t *testing.T, r reponse, secrets ...string) {
	t.Helper()
	exigerProbleme(t, r, http.StatusConflict, typeArticleReserve)
	for _, secret := range secrets {
		if secret != "" && strings.Contains(string(r.corps), secret) {
			t.Errorf("refus qui révèle %q : %s", secret, r.corps)
		}
	}
}

type garde func(etape string)

func sansGarde(string) {}

func exigerAvantLEcheance(t *testing.T, echeance string) garde {
	t.Helper()
	fin := instantRFC3339(t, echeance)
	return func(etape string) {
		t.Helper()
		if !time.Now().Before(fin) {
			t.Fatalf("échéance %s dépassée avant la fin de « %s » (réservation de %s sur ce banc) : la réservation n'est plus active, le scénario n'est pas probant ; relancer sur une machine moins chargée ou allonger dureeCourte", echeance, etape, dureeCourte)
		}
	}
}

func (b *bancReservation) tentativesDuVendeur(t *testing.T, jetonVendeur string, article articleRecu, verifier garde, secrets ...string) {
	t.Helper()
	revision := article.Revision
	modifications := map[string]string{
		"modification complète":    fmt.Sprintf("{\"revision\":%d,"+corpsDeModificationVol+"}", revision, categorieDeLaReservaton),
		"prix seul":                fmt.Sprintf(`{"revision":%d,"prix":{"montantCentimes":1,"devise":"EUR"}}`, revision),
		"frais seuls":              fmt.Sprintf(`{"revision":%d,"fraisDePort":{"montantCentimes":0,"devise":"EUR"}}`, revision),
		"retrait par modification": fmt.Sprintf(`{"revision":%d,"titre":"Retirée de la vente"}`, revision),
		"révision périmée":         fmt.Sprintf(`{"revision":%d,"titre":"Révision périmée"}`, revision-1),
	}
	for etape, corps := range modifications {
		r := b.modifier(t, jetonVendeur, article.ID, corps)
		verifier(etape)
		b.refusArticleReserve(t, r, secrets...)
	}
	r := b.soumettre(t, jetonVendeur, article.ID, revision)
	verifier("soumission")
	b.refusArticleReserve(t, r, secrets...)
}

func (b *bancReservation) ac5ArticleReserveFige(t *testing.T) {
	vendeur := b.kc.creerVendeur(t)
	article := b.publierPlusieurs(t, vendeur, 1)[0]
	jetonVendeur := bearer(b.kc.seConnecter(t, vendeur))
	acheteur := b.connecterAcheteur(t)
	commande := b.commanderAvecSucces(t, acheteur.jeton, article.ID, http.StatusCreated)
	active := exigerAvantLEcheance(t, commande.Echeance)
	avant, ligneAvant := b.articleEnBase(t, article.ID)
	b.tentativesDuVendeur(t, jetonVendeur, article, active, acheteur.compte.sujet, commande.Echeance, commande.ID)
	b.exigerArticleInchange(t, article.ID, avant, ligneAvant)
	relu := b.relire(t, jetonVendeur, article.ID)
	if relu.Revision != article.Revision || relu.Etat != etatPublie {
		t.Errorf("article relu par le vendeur : révision %d, état %s", relu.Revision, relu.Etat)
	}
	exigerMontant(t, "prix relu par le vendeur", relu.Prix, prixDeReference)
	exigerMontant(t, "frais relus par le vendeur", relu.FraisDePort, fraisDeReference)
	b.exigerArticleReserve(t, article.ID)
	exigerMontant(t, "prix de la commande", b.lireCommandeRecue(t, acheteur.jeton, commande.ID).Prix, prixDeReference)
	active("relectures du vendeur, de la fiche et de la commande")
	if avant.Prix != prixDeReference || avant.Frais != fraisDeReference {
		t.Errorf("prix en base %d, frais %d", avant.Prix, avant.Frais)
	}
	t.Logf("article réservé : 5 modifications et 1 soumission refusées en 409 article-reserve, ligne de l'article identique")
	attendreLEcheance(t, commande.Echeance)
	var modifie articleRecu
	b.modifier(t, jetonVendeur, article.ID, fmt.Sprintf(`{"revision":%d,"prix":{"montantCentimes":9000,"devise":"EUR"}}`, article.Revision)).exiger(t, http.StatusOK).lire(t, &modifie)
	if modifie.Revision != article.Revision+1 || modifie.Etat != etatEnControle || modifie.Prix == nil || modifie.Prix.MontantCentimes != 9000 {
		t.Errorf("modification après l'échéance : %+v", modifie)
	}
}

func (b *bancReservation) ac5ArticleVenduFige(t *testing.T) {
	vendeur := b.kc.creerVendeur(t)
	article := b.publierPlusieurs(t, vendeur, 1)[0]
	jetonVendeur := bearer(b.kc.seConnecter(t, vendeur))
	if _, err := b.marche.maitre.Exec(t.Context(), `UPDATE article SET vendu = true WHERE id = $1`, article.ID); err != nil {
		t.Fatalf("article vendu : %v", err)
	}
	avant, ligneAvant := b.articleEnBase(t, article.ID)
	b.tentativesDuVendeur(t, jetonVendeur, article, sansGarde)
	b.exigerArticleInchange(t, article.ID, avant, ligneAvant)
}

func (b *bancReservation) ac5CourseVendeurContreAcheteurs(t *testing.T) {
	b.courirContreLeVendeur(t, toursDeCourseVendeur, pasDeDecalageVendeur, func(t *testing.T, jeton, articleID string) *http.Request {
		return b.requeteDeCommande(t, jeton, articleID)
	})
}

func (b *bancReservation) ac5CourseVendeurContreReserverDirect(t *testing.T) {
	b.courirContreLeVendeur(t, toursDeCourseDirecte, pasDeDecalageDirect, func(t *testing.T, jeton, articleID string) *http.Request {
		return b.requeteDeReservationDirecte(t, b.kc.jetonDeTransactions(t), jeton, articleID)
	})
}

func (b *bancReservation) courirContreLeVendeur(t *testing.T, tours int, pas time.Duration, reserver func(*testing.T, string, string) *http.Request) {
	t.Helper()
	vendeur := b.kc.creerVendeur(t)
	articles := b.publierPlusieurs(t, vendeur, tours)
	jetonVendeur := bearer(b.kc.seConnecter(t, vendeur))
	bilan := map[string]int{}
	for tour, article := range articles {
		acheteurs := b.connecterAcheteurs(t, acheteursParCourse)
		requetes := []*http.Request{b.requeteDeModification(t, jetonVendeur, article.ID, article.Revision)}
		decalages := []time.Duration{time.Duration(tour) * pas}
		for _, a := range acheteurs {
			requetes = append(requetes, reserver(t, a.jeton, article.ID))
			decalages = append(decalages, 0)
		}
		reponses := b.enParalleleDecale(t, requetes, decalages)
		bilan[b.issueDeLaCourse(t, tour+1, article, reponses[0], reponses[1:])]++
	}
	t.Logf("%d courses (décalage du vendeur de 0 à %s) : %v", tours, time.Duration(tours-1)*pas, bilan)
}

func reservationsAcceptees(reservations []reponse) int {
	acceptees := 0
	for _, r := range reservations {
		if r.statut == http.StatusCreated {
			acceptees++
		}
	}
	return acceptees
}

func (b *bancReservation) issueDeLaCourse(t *testing.T, tour int, article articleRecu, modification reponse, reservations []reponse) string {
	t.Helper()
	acceptees := reservationsAcceptees(reservations)
	if acceptees > 1 {
		t.Fatalf("tour %d : %d réservations acceptées", tour, acceptees)
	}
	if modification.statut == http.StatusOK && acceptees == 1 {
		t.Fatalf("tour %d : modification du vendeur (200) ET réservation (201) acceptées toutes deux", tour)
	}
	switch {
	case acceptees == 1:
		b.refusArticleReserve(t, modification)
		if apres, _ := b.articleEnBase(t, article.ID); apres.Revision != article.Revision || apres.Prix != prixDeReference || apres.Frais != fraisDeReference || apres.Etat != etatPublie {
			t.Errorf("tour %d : article réservé mais modifié : %+v", tour, apres)
		}
		return "réservation d'abord, modification 409 article-reserve, article intact"
	case modification.statut == http.StatusOK:
		for _, r := range reservations {
			exigerProbleme(t, r, http.StatusUnprocessableEntity, typeNonReservable)
		}
		if n := b.compterCommandes(t, "WHERE article_id = $1", article.ID); n != 0 {
			t.Errorf("tour %d : %d commandes sur l'article modifié", tour, n)
		}
		if b.reservationEnBase(t, article.ID).reservationID != nil {
			t.Errorf("tour %d : article modifié mais réservé en base", tour)
		}
		return "modification d'abord, réservations 422 article-non-reservable, aucune commande"
	default:
		t.Fatalf("tour %d : aucune issue : modification %d %s, réservations %d %d", tour, modification.statut, modification.corps, reservations[0].statut, reservations[1].statut)
		return ""
	}
}

func (b *bancReservation) contratsUS031(t *testing.T) {
	b.reponsesConformesAuxContrats(t)
	routeur := chargerContrat(t, b.public)
	vendeur := 0
	for _, e := range b.historique() {
		if strings.HasPrefix(e.requete.URL.Path, cheminArticles) {
			vendeur++
			verifierEchange(t, routeur, e)
		}
	}
	b.mu.Lock()
	expirees := 0
	for _, e := range b.echangesTx {
		if strings.Contains(string(e.reponse.corps), `"statut":"`+statutExpiree+`"`) {
			expirees++
		}
	}
	b.mu.Unlock()
	if expirees == 0 {
		t.Errorf("aucune commande expiree vérifiée au contrat")
	}
	t.Logf("routes du vendeur : %d échanges vérifiés ; réponses de Transactions au statut expiree : %d", vendeur, expirees)
}
