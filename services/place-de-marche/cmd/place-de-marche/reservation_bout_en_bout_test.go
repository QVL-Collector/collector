package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestUS030BoutEnBoutTransactionsPlaceDeMarche(t *testing.T) {
	b := demarrerBancReservation(t)
	t.Run("AC1 réserver : jeton de service propagé avec le jeton de l'acheteur, commande en attente de paiement, réservation en base", b.ac1ReserverEtCommander)
	t.Run("AC2 montants lus par la Place de marché, commission de 5 % au centime, échéance plafonnée", b.ac2MontantsEtEcheance)
	t.Run("AC3 article réservé : mention réservé au catalogue et sur la fiche, plus réservable", b.ac3ArticleReserve)
	t.Run("AC3 nouvel essai du même acheteur : même commande, même échéance", b.ac3NouvelEssaiMemeCommande)
	t.Run("AC4 route réserver : jetons de service absents ou invalides refusés en 401 ou 403", b.ac4JetonsDeServiceRefuses)
	t.Run("AC5 identité de l'acheteur absente, forgée ou d'un autre : réservation refusée", b.ac5IdentiteDeLAcheteur)
	t.Run("AC5 article non publié ou inexistant, vendeur sur son propre article : 422 identique", b.ac5ArticleNonReservable)
	t.Run("AC6 prix, montants et acheteur envoyés par l'acheteur ignorés", b.ac6MontantsIgnores)
	t.Run("AC8 bases transactions et marche cloisonnées", b.ac8BasesCloisonnees)
	t.Run("AC9 jeton de service de Transactions refusé sur les routes d'acheteur", b.ac9JetonDeServiceSurRoutesAcheteur)
	t.Run("Exposition : réserver absente du port public, routes publiques absentes du port interne", b.routeInterneAbsenteDuPortPublic)
	t.Run("Hors AC : jeton de l'acheteur jamais journalisé", b.jetonAcheteurNonJournalise)
	t.Run("Contrats OpenAPI : Transactions, Place de marché interne et catalogue public", b.reponsesConformesAuxContrats)
}

func (b *bancReservation) ac1ReserverEtCommander(t *testing.T) {
	article := b.articlePublie(t, 12000, 590)
	acheteur := b.kc.creerAcheteur(t)
	jeton := b.kc.seConnecter(t, acheteur)
	avant := time.Now()
	r := b.commander(t, jeton, corpsDeCommande(article.ID)).exiger(t, http.StatusCreated)
	var commande commandeRecue
	r.lire(t, &commande)
	if commande.Statut != statutEnAttente || commande.ArticleID != article.ID {
		t.Fatalf("commande %+v : statut %s et article %s attendus", commande, statutEnAttente, article.ID)
	}
	if emplacement := r.entetes.Get("Location"); !strings.HasSuffix(emplacement, cheminCommandes+"/"+commande.ID) {
		t.Fatalf("Location %q, attendu …%s/%s", emplacement, cheminCommandes, commande.ID)
	}
	b.exigerAppelDeReserverDelegue(t, article.ID, jeton)
	b.exigerCommandeEtReservationEnBase(t, commande.ID, article.ID, acheteur.sujet, avant)
	var relue commandeRecue
	b.lireCommande(t, jeton, commande.ID).exiger(t, http.StatusOK).lire(t, &relue)
	if !reflect.DeepEqual(relue, commande) {
		t.Errorf("commande relue %+v différente de la commande créée %+v", relue, commande)
	}
	b.lireCommande(t, b.kc.seConnecter(t, b.kc.creerAcheteur(t)), commande.ID).exiger(t, http.StatusNotFound)
}

func (b *bancReservation) exigerAppelDeReserverDelegue(t *testing.T, articleID, jetonAcheteur string) {
	t.Helper()
	appels := b.relais.appelsVers(articleID)
	if len(appels) != 1 || appels[0].methode != http.MethodPost || appels[0].chemin != cheminDeReservation(articleID) {
		t.Fatalf("appels de réserver relayés : %+v, un POST %s attendu", appels, cheminDeReservation(articleID))
	}
	appel := appels[0]
	if valeurs := appel.entetes.Values(enteteJetonAcheteur); len(valeurs) != 1 || valeurs[0] != jetonAcheteur {
		t.Fatalf("jeton de l'acheteur non propagé tel quel dans %s", enteteJetonAcheteur)
	}
	jetonService, bearerPresent := strings.CutPrefix(appel.entetes.Get("Authorization"), "Bearer ")
	if !bearerPresent || jetonService == jetonAcheteur {
		t.Fatalf("Authorization sans jeton de service Bearer distinct du jeton de l'acheteur")
	}
	b.exigerJetonDeServiceRestreint(t, lireLaCharge(t, jetonService))
	if appel.statut != http.StatusCreated {
		t.Fatalf("réserver a répondu %d, 201 attendu : %s", appel.statut, appel.reponse)
	}
}

func (b *bancReservation) exigerJetonDeServiceRestreint(t *testing.T, charge chargeDeJeton) {
	t.Helper()
	if charge["iss"] != b.kc.emetteur() || charge["azp"] != clientTransactions {
		t.Errorf("jeton de service émis par %v pour %v, attendu %s et %s", charge["iss"], charge["azp"], b.kc.emetteur(), clientTransactions)
	}
	if !slices.Contains(charge.audiences(), audienceAttendue) {
		t.Errorf("audience %v sans %s", charge.audiences(), audienceAttendue)
	}
	if !slices.Contains(charge.portees(), porteeReserver) {
		t.Errorf("scope %v sans %s", charge.portees(), porteeReserver)
	}
	if vie := charge.dureeDeVie(); vie <= 0 || vie > 5*time.Minute {
		t.Errorf("durée de vie du jeton de service %s, 5 min au plus attendu", vie)
	}
	if _, present := charge["realm_access"]; present {
		t.Errorf("jeton de service avec realm_access %v", charge["realm_access"])
	}
}

func (b *bancReservation) exigerCommandeEtReservationEnBase(t *testing.T, commandeID, articleID, acheteur string, avant time.Time) {
	t.Helper()
	ligne := b.commandeEnBase(t, commandeID)
	reservation := b.reservationEnBase(t, articleID)
	if ligne["acheteur_id"] != acheteur || ligne["statut"] != statutEnAttente {
		t.Errorf("commande en base pour %v (%v), attendue pour %s en %s", ligne["acheteur_id"], ligne["statut"], acheteur, statutEnAttente)
	}
	if reservation.reservationID == nil || ligne["reservation_id"] != *reservation.reservationID {
		t.Errorf("commande sur la réservation %v, article réservé par %v", ligne["reservation_id"], reservation.reservationID)
	}
	if reservation.jusquA == nil || !reservation.jusquA.After(avant) {
		t.Errorf("article non réservé en base : échéance %v", reservation.jusquA)
	}
	if n := b.compterReservations(t, articleID); n != 1 {
		t.Errorf("%d réservation(s) journalisée(s) pour l'article, 1 attendue", n)
	}
}

func (b *bancReservation) publierAuPrix(t *testing.T, prix int64, fraisDePort *int64, enBase bool) articleRecu {
	t.Helper()
	if !enBase {
		return b.publier(t, b.kc.creerVendeur(t), "Pièce "+texteAleatoire()[:6], prix, fraisDePort)
	}
	article := b.articlePublie(t, 3000, 400)
	if _, err := b.marche.maitre.Exec(t.Context(), `UPDATE article SET prix_centimes = $2, frais_port_centimes = $3 WHERE id = $1`, article.ID, prix, fraisDePort); err != nil {
		t.Fatalf("prix de l'article en base : %v", err)
	}
	return article
}

func exigerMontant(t *testing.T, nom string, m *montant, centimes int64) {
	t.Helper()
	if m == nil || m.MontantCentimes != centimes || m.Devise != "EUR" {
		t.Errorf("%s %+v, attendu %d centimes EUR", nom, m, centimes)
	}
}

type casDeMontants struct {
	nom        string
	prix       int64
	frais      *int64
	commission int64
	enBase     bool
}

func (c casDeMontants) fraisAttendus() int64 {
	if c.frais == nil {
		return 0
	}
	return *c.frais
}

func (c casDeMontants) total() int64 {
	return c.prix + c.fraisAttendus()
}

func (b *bancReservation) ac2MontantsEtEcheance(t *testing.T) {
	cas := []casDeMontants{
		{"120 € + 5,90 € : 6,295 € arrondi à 6,30 €", 12000, frais(590), 630, false},
		{"45 € + 6 € : 2,55 €", 4500, frais(600), 255, false},
		{"19,90 € sans frais renseignés : frais 0, 99,5 c arrondi à 1 €", 1990, nil, 100, false},
		{"prix changé dans la base de la Place de marché après publication : 0,10 € et frais nuls, demi-centime arrondi vers le haut", 10, frais(0), 1, true},
		{"prix changé dans la base de la Place de marché après publication : 1 000 000 € + 1 000 €, bornes de l'article", 100_000_000, frais(100_000), 5_005_000, true},
		{"prix changé dans la base de la Place de marché après publication : frais effacés", 3010, nil, 151, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			article := b.publierAuPrix(t, c.prix, c.frais, c.enBase)
			jeton := b.kc.seConnecter(t, b.kc.creerAcheteur(t))
			avant := time.Now()
			creee := b.commanderAvecSucces(t, jeton, article.ID, http.StatusCreated)
			apres := time.Now()
			var commande commandeRecue
			b.lireCommande(t, jeton, creee.ID).exiger(t, http.StatusOK).lire(t, &commande)
			exigerMontantsDeCommande(t, commande, c)
			b.exigerMontantsEnBase(t, creee.ID, c)
			echeance := exigerEcheancePlafonnee(t, commande.Echeance, avant, apres)
			b.exigerEcheanceEtPrixDeLaPlaceDeMarche(t, article.ID, echeance, c)
		})
	}
}

func exigerMontantsDeCommande(t *testing.T, commande commandeRecue, c casDeMontants) {
	t.Helper()
	exigerMontant(t, "prix", commande.Prix, c.prix)
	exigerMontant(t, "frais de port", commande.FraisDePort, c.fraisAttendus())
	exigerMontant(t, "total payé", commande.TotalPaye, c.total())
	exigerMontant(t, "commission", commande.Commission, c.commission)
	exigerMontant(t, "montant vendeur", commande.MontantVendeur, c.total()-c.commission)
	if commande.Commission != nil && commande.MontantVendeur != nil && commande.Commission.MontantCentimes+commande.MontantVendeur.MontantCentimes != c.total() {
		t.Errorf("ENF-01 : commission + montant vendeur != total payé")
	}
}

func (b *bancReservation) exigerMontantsEnBase(t *testing.T, commandeID string, c casDeMontants) {
	t.Helper()
	ligne := b.commandeEnBase(t, commandeID)
	attendus := map[string]int64{
		"prix_centimes":            c.prix,
		"frais_port_centimes":      c.fraisAttendus(),
		"total_paye_centimes":      c.total(),
		"commission_centimes":      c.commission,
		"montant_vendeur_centimes": c.total() - c.commission,
	}
	for colonne, attendu := range attendus {
		if valeur, ok := ligne[colonne].(float64); !ok || int64(valeur) != attendu {
			t.Errorf("colonne %s en base : %v, attendu %d", colonne, ligne[colonne], attendu)
		}
	}
}

func exigerEcheancePlafonnee(t *testing.T, valeur string, avant, apres time.Time) time.Time {
	t.Helper()
	echeance := instantRFC3339(t, valeur)
	if !strings.HasSuffix(valeur, "Z") {
		t.Errorf("échéance %q hors UTC", valeur)
	}
	if echeance.Before(avant.Add(plafondDeReservation-toleranceEcheance)) || echeance.After(apres.Add(plafondDeReservation+toleranceEcheance)) {
		t.Errorf("échéance %s, attendue au plafond de %s (entre %s et %s), durée configurée %s", echeance, plafondDeReservation, avant.Add(plafondDeReservation), apres.Add(plafondDeReservation), dureeDeReservation)
	}
	return echeance
}

func (b *bancReservation) exigerEcheanceEtPrixDeLaPlaceDeMarche(t *testing.T, articleID string, echeance time.Time, c casDeMontants) {
	t.Helper()
	enBase := b.reservationEnBase(t, articleID)
	if enBase.jusquA == nil || !enBase.jusquA.Equal(echeance) {
		t.Errorf("échéance de la commande %s, échéance de la Place de marché %v", echeance, enBase.jusquA)
	}
	appels := b.relais.appelsVers(articleID)
	if len(appels) != 1 {
		t.Fatalf("%d appel(s) de réserver, 1 attendu", len(appels))
	}
	var reservation reservationRecue
	reponse{statut: appels[0].statut, corps: appels[0].reponse}.lire(t, &reservation)
	if !instantRFC3339(t, reservation.Echeance).Equal(echeance) {
		t.Errorf("échéance renvoyée par la Place de marché %s, reprise %s", reservation.Echeance, echeance)
	}
	exigerMontant(t, "prix lu par la Place de marché", reservation.Prix, c.prix)
	exigerMontant(t, "frais lus par la Place de marché", reservation.FraisDePort, c.fraisAttendus())
}

func (b *bancReservation) ac3ArticleReserve(t *testing.T) {
	article := b.articlePublie(t, 3000, 450)
	voisin := b.articlePublie(t, 3100, 450)
	b.exigerMention(t, article.ID, false)
	b.exigerMention(t, voisin.ID, false)
	premier := b.kc.seConnecter(t, b.kc.creerAcheteur(t))
	commande := b.commanderAvecSucces(t, premier, article.ID, http.StatusCreated)
	b.exigerMention(t, article.ID, true)
	b.exigerMention(t, voisin.ID, false)
	visiteur := b.envoyer(t, http.MethodGet, cheminCatalogue+"/"+article.ID, "").exiger(t, http.StatusOK)
	if bytes.Contains(visiteur.corps, []byte(commande.ID)) {
		t.Errorf("la fiche publique révèle la commande")
	}
	avant := b.reservationEnBase(t, article.ID)
	second := b.kc.creerAcheteur(t)
	jetonSecond := b.kc.seConnecter(t, second)
	refus := b.commander(t, jetonSecond, corpsDeCommande(article.ID)).probleme(t, http.StatusConflict)
	if refus.Type != typeDejaReserve {
		t.Errorf("refus %+v, type %s attendu", refus, typeDejaReserve)
	}
	direct := b.reserverEnDirect(t, article.ID, entetesDeService(b.kc.jetonDeTransactions(t), jetonSecond)).probleme(t, http.StatusConflict)
	if direct.Type != typeDejaReserve {
		t.Errorf("refus direct %+v, type %s attendu", direct, typeDejaReserve)
	}
	if n := b.compterCommandes(t, "WHERE acheteur_id::text = $1", second.sujet); n != 0 {
		t.Errorf("%d commande(s) pour le second acheteur, 0 attendue", n)
	}
	apres := b.reservationEnBase(t, article.ID)
	if *apres.reservationID != *avant.reservationID || !apres.jusquA.Equal(*avant.jusquA) {
		t.Errorf("réservation modifiée par le refus : %v %v → %v %v", *avant.reservationID, avant.jusquA, *apres.reservationID, apres.jusquA)
	}
	b.exigerMention(t, article.ID, true)
	autre := b.articlePublie(t, 2000, 300)
	active := b.commander(t, premier, corpsDeCommande(autre.ID)).probleme(t, http.StatusConflict)
	if active.Type != typeReservationActive {
		t.Errorf("seconde réservation du même acheteur : %+v, type %s attendu", active, typeReservationActive)
	}
	b.exigerMention(t, autre.ID, false)
}

func (b *bancReservation) ac3NouvelEssaiMemeCommande(t *testing.T) {
	article := b.articlePublie(t, 5000, 500)
	acheteur := b.kc.creerAcheteur(t)
	jeton := b.kc.seConnecter(t, acheteur)
	premiere := b.commanderAvecSucces(t, jeton, article.ID, http.StatusCreated)
	time.Sleep(1500 * time.Millisecond)
	for essai := 2; essai <= 3; essai++ {
		nouvelle := b.commanderAvecSucces(t, jeton, article.ID, http.StatusOK)
		if !reflect.DeepEqual(nouvelle, premiere) {
			t.Fatalf("essai %d : commande %+v, attendue identique à %+v", essai, nouvelle, premiere)
		}
	}
	direct := b.reserverEnDirect(t, article.ID, entetesDeService(b.kc.jetonDeTransactions(t), b.kc.seConnecter(t, acheteur))).exiger(t, http.StatusOK)
	var reservation reservationRecue
	direct.lire(t, &reservation)
	if !instantRFC3339(t, reservation.Echeance).Equal(instantRFC3339(t, premiere.Echeance)) {
		t.Errorf("nouvel essai direct : échéance %s, attendue %s", reservation.Echeance, premiere.Echeance)
	}
	if n := b.compterCommandes(t, "WHERE acheteur_id::text = $1", acheteur.sujet); n != 1 {
		t.Errorf("%d commandes pour l'acheteur, 1 attendue", n)
	}
	if n := b.compterReservations(t, article.ID); n != 1 {
		t.Errorf("%d réservations journalisées, 1 attendue", n)
	}
}

func (b *bancReservation) exigerMention(t *testing.T, id string, reserve bool) {
	t.Helper()
	fiche, entree := b.fiche(t, id), b.entreeDuCatalogue(t, id)
	if fiche.Reserve == nil || entree.Reserve == nil || *fiche.Reserve != reserve || *entree.Reserve != reserve {
		t.Errorf("article %s : reserve fiche %v, catalogue %v ; %t attendu", id, valeurDe(fiche.Reserve), valeurDe(entree.Reserve), reserve)
	}
}

func valeurDe(reserve *bool) string {
	if reserve == nil {
		return "absent"
	}
	return fmt.Sprint(*reserve)
}

func exigerRefusDAutorisation(t *testing.T, cas string, r reponse) {
	t.Helper()
	if r.statut != http.StatusUnauthorized && r.statut != http.StatusForbidden {
		t.Errorf("%s : statut %d, 401 ou 403 attendu : %s", cas, r.statut, r.corps)
		return
	}
	t.Logf("%s : %d", cas, r.statut)
}

func (b *bancReservation) exigerNonReserve(t *testing.T, articleID string) {
	t.Helper()
	b.exigerMention(t, articleID, false)
	if n := b.compterReservations(t, articleID); n != 0 {
		t.Errorf("%d réservation(s) journalisée(s) malgré les refus", n)
	}
}

func (b *bancReservation) ac4JetonsDeServiceRefuses(t *testing.T) {
	article := b.articlePublie(t, 2500, 300)
	acheteur := b.kc.seConnecter(t, b.kc.creerAcheteur(t))
	valide := b.kc.jetonDeTransactions(t)
	intrusReservant := b.kc.creerClientDeService(t, []string{"basic", porteeReserver})
	intrusSimple := b.kc.creerClientDeService(t, []string{"basic"})
	jetonIntrusReservant := b.kc.jetonDeService(t, intrusReservant.identifiant, intrusReservant.secret)
	if c := lireLaCharge(t, jetonIntrusReservant); !slices.Contains(c.portees(), porteeReserver) || !slices.Contains(c.audiences(), audienceAttendue) {
		t.Fatalf("jeton du client intrus sans la portée reserver : scope %v, aud %v", c.portees(), c.audiences())
	}
	autreAudience := b.kc.jetonDeTransactionsPourUneAutreAudience(t, identiteInteractions)
	if c := lireLaCharge(t, autreAudience); slices.Contains(c.audiences(), audienceAttendue) || !slices.Contains(c.portees(), porteeReserver) {
		t.Fatalf("jeton d'une autre audience mal construit : aud %v, scope %v", c.audiences(), c.portees())
	}
	expire := b.kc.jetonDeTransactionsExpire(t)
	cas := []struct {
		nom     string
		entetes http.Header
	}{
		{"sans jeton de service", entetesDeService("", acheteur)},
		{"jeton de l'acheteur à la place du jeton de service", entetesDeService(acheteur, acheteur)},
		{"jeton d'un utilisateur vendeur", entetesDeService(b.kc.seConnecter(t, b.kc.creerVendeur(t)), acheteur)},
		{"jeton d'un autre client de service portant reserver", entetesDeService(jetonIntrusReservant, acheteur)},
		{"jeton d'un autre client de service sans reserver", entetesDeService(b.kc.jetonDeService(t, intrusSimple.identifiant, intrusSimple.secret), acheteur)},
		{"jeton de service expiré", entetesDeService(expire, acheteur)},
		{"jeton de service destiné à une autre audience", entetesDeService(autreAudience, acheteur)},
		{"jeton de service à la charge altérée", entetesDeService(jetonAvecChargeAlteree(t, valide, func(c chargeDeJeton) { c["azp"] = clientTransactions + "x" }), acheteur)},
		{"jeton de service non signé (alg none)", entetesDeService(jetonNonSigne(t, valide), acheteur)},
		{"jeton de service en Basic", http.Header{"Authorization": {"Basic " + valide}, enteteJetonAcheteur: {acheteur}}},
		{"jeton de service sans le préfixe Bearer", http.Header{"Authorization": {valide}, enteteJetonAcheteur: {acheteur}}},
		{"jeton de service illisible", entetesDeService("pas-un-jeton", acheteur)},
	}
	for _, c := range cas {
		exigerRefusDAutorisation(t, c.nom, b.reserverEnDirect(t, article.ID, c.entetes))
	}
	b.exigerNonReserve(t, article.ID)
	b.reserverEnDirect(t, article.ID, entetesDeService(b.kc.jetonDeTransactions(t), acheteur)).exiger(t, http.StatusCreated)
}

func (b *bancReservation) ac5IdentiteDeLAcheteur(t *testing.T) {
	article := b.articlePublie(t, 2600, 300)
	service := b.kc.jetonDeTransactions(t)
	acheteurA := b.kc.creerAcheteur(t)
	acheteurB := b.kc.creerAcheteur(t)
	jetonA, jetonB := b.kc.seConnecter(t, acheteurA), b.kc.seConnecter(t, acheteurB)
	cas := []struct {
		nom     string
		entetes http.Header
	}{
		{"identité de l'acheteur absente", entetesDeService(service, "")},
		{"identité vide", http.Header{"Authorization": {bearer(service)}, enteteJetonAcheteur: {""}}},
		{"identité forgée : sujet remplacé, signature d'origine", entetesDeService(service, jetonAvecChargeAlteree(t, jetonA, func(c chargeDeJeton) { c["sub"] = acheteurB.sujet }))},
		{"identité forgée : jeton non signé", entetesDeService(service, jetonNonSigne(t, jetonA))},
		{"identité forgée : texte arbitraire", entetesDeService(service, "acheteur:"+acheteurA.sujet)},
		{"jeton de service à la place du jeton de l'acheteur", entetesDeService(service, service)},
		{"deux identités d'acheteurs différents", http.Header{"Authorization": {bearer(service)}, enteteJetonAcheteur: {jetonA, jetonB}}},
	}
	for _, c := range cas {
		r := b.reserverEnDirect(t, article.ID, c.entetes)
		if r.statut < 400 || r.statut >= 500 {
			t.Errorf("%s : statut %d, refus attendu : %s", c.nom, r.statut, r.corps)
			continue
		}
		t.Logf("%s : %d", c.nom, r.statut)
	}
	b.exigerNonReserve(t, article.ID)

	if ligne := b.commandeEnBase(t, b.commanderAvecSucces(t, jetonA, article.ID, http.StatusCreated).ID); ligne["acheteur_id"] != acheteurA.sujet {
		t.Errorf("commande valide attribuée à %v, attendue pour %s", ligne["acheteur_id"], acheteurA.sujet)
	}
	t.Run("identité d'un autre acheteur glissée dans la requête à Transactions : ignorée", func(t *testing.T) {
		article := b.articlePublie(t, 2700, 300)
		acheteurC, acheteurD := b.kc.creerAcheteur(t), b.kc.creerAcheteur(t)
		jetonC, jetonD := b.kc.seConnecter(t, acheteurC), b.kc.seConnecter(t, acheteurD)
		var commande commandeRecue
		b.commander(t, jetonC, fmt.Sprintf(`{"articleId":%q,"acheteurId":%q,"acheteur":%q}`, article.ID, acheteurD.sujet, acheteurD.sujet),
			http.Header{enteteJetonAcheteur: {jetonD}}).exiger(t, http.StatusCreated).lire(t, &commande)
		if ligne := b.commandeEnBase(t, commande.ID); ligne["acheteur_id"] != acheteurC.sujet {
			t.Errorf("commande attribuée à %v, attendue pour le sujet du jeton %s", ligne["acheteur_id"], acheteurC.sujet)
		}
		appels := b.relais.appelsVers(article.ID)
		if len(appels) != 1 || appels[0].entetes.Get(enteteJetonAcheteur) != jetonC || len(appels[0].entetes.Values(enteteJetonAcheteur)) != 1 {
			t.Errorf("identité propagée à la Place de marché différente du jeton de l'appelant")
		}
		b.lireCommande(t, jetonD, commande.ID).exiger(t, http.StatusNotFound)
		b.lireCommande(t, jetonC, commande.ID).exiger(t, http.StatusOK)
	})
}

func (b *bancReservation) ac5ArticleNonReservable(t *testing.T) {
	vendeur := b.kc.creerVendeur(t)
	jetonVendeur := bearer(b.kc.seConnecter(t, vendeur))
	brouillon := b.creer(t, jetonVendeur, corpsComplet("Brouillon", categorieSure, 1500, 400))
	refuse := b.soumettreAvecSucces(t, jetonVendeur, b.creer(t, jetonVendeur, corpsComplet("Figurine, appelez le "+telephoneFictif, categorieSure, 1500, 400)))
	if _, err := b.marche.maitre.Exec(t.Context(), `INSERT INTO categorie (id, libelle) VALUES ($1, 'Timbres') ON CONFLICT DO NOTHING`, categorieNonReconnue); err != nil {
		t.Fatalf("catégorie inconnue du Contrôle : %v", err)
	}
	aVerifier := b.soumettreAvecSucces(t, jetonVendeur, b.creer(t, jetonVendeur, corpsComplet("Timbre rare", categorieNonReconnue, 1500, 400)))
	attendre(t, delaiDePublicationResa, "décisions du Contrôle", func() bool {
		return b.relire(t, jetonVendeur, refuse.ID).Etat == etatRefuse && b.relire(t, jetonVendeur, aVerifier.ID).Etat == etatAVerifier
	})
	propre := b.publier(t, vendeur, "Figurine du vendeur", 4000, frais(500))
	vendu := b.articlePublie(t, 4100, 500)
	if _, err := b.marche.maitre.Exec(t.Context(), `UPDATE article SET vendu = true WHERE id = $1`, vendu.ID); err != nil {
		t.Fatalf("article vendu : %v", err)
	}
	cas := []struct {
		nom     string
		article string
		jeton   func() string
	}{
		{"brouillon", brouillon.ID, func() string { return b.kc.seConnecter(t, b.kc.creerAcheteur(t)) }},
		{"refusé", refuse.ID, func() string { return b.kc.seConnecter(t, b.kc.creerAcheteur(t)) }},
		{"à vérifier", aVerifier.ID, func() string { return b.kc.seConnecter(t, b.kc.creerAcheteur(t)) }},
		{"inexistant", uuid.NewString(), func() string { return b.kc.seConnecter(t, b.kc.creerAcheteur(t)) }},
		{"vendeur sur son propre article publié", propre.ID, func() string { return b.kc.seConnecter(t, vendeur) }},
		{"publié mais déjà vendu", vendu.ID, func() string { return b.kc.seConnecter(t, b.kc.creerAcheteur(t)) }},
	}
	var reference, referenceDirecte *probleme
	for _, c := range cas {
		jeton := c.jeton()
		refus := b.commander(t, jeton, corpsDeCommande(c.article)).probleme(t, http.StatusUnprocessableEntity)
		direct := b.reserverEnDirect(t, c.article, entetesDeService(b.kc.jetonDeTransactions(t), jeton)).probleme(t, http.StatusUnprocessableEntity)
		if refus.Type != typeNonReservable || direct.Type != typeNonReservable {
			t.Errorf("%s : types %s et %s, %s attendu", c.nom, refus.Type, direct.Type, typeNonReservable)
		}
		if reference == nil {
			reference, referenceDirecte = &refus, &direct
		}
		if refus.Title != reference.Title || refus.Detail != reference.Detail || direct.Title != referenceDirecte.Title || direct.Detail != referenceDirecte.Detail {
			t.Errorf("%s : réponse différente des autres cas : %+v / %+v", c.nom, refus, direct)
		}
		if n := b.compterCommandes(t, "WHERE article_id::text = $1", c.article); n != 0 {
			t.Errorf("%s : %d commande(s) créée(s)", c.nom, n)
		}
	}
	b.exigerMention(t, propre.ID, false)
	jeton := b.kc.seConnecter(t, b.kc.creerAcheteur(t))
	b.reserverEnDirect(t, "pas-un-uuid", entetesDeService(b.kc.jetonDeTransactions(t), jeton)).exiger(t, http.StatusNotFound)
	saisie := b.commander(t, jeton, `{"articleId":"pas-un-uuid"}`).probleme(t, http.StatusBadRequest)
	if saisie.Type != typeSaisieInvalideTx {
		t.Errorf("articleId mal formé : %+v, type %s attendu", saisie, typeSaisieInvalideTx)
	}
	b.commander(t, jeton, `{}`).exiger(t, http.StatusBadRequest)
	b.commanderAvecSucces(t, jeton, propre.ID, http.StatusCreated)
}

func (b *bancReservation) ac6MontantsIgnores(t *testing.T) {
	article := b.articlePublie(t, 8000, 700)
	acheteur := b.kc.creerAcheteur(t)
	jeton := b.kc.seConnecter(t, acheteur)
	intrus := uuid.NewString()
	corps := fmt.Sprintf(`{"articleId":%q,"id":%q,"prix":{"montantCentimes":1,"devise":"EUR"},"fraisDePort":{"montantCentimes":0,"devise":"EUR"},"totalPaye":{"montantCentimes":1,"devise":"EUR"},"commission":{"montantCentimes":0,"devise":"EUR"},"montantVendeur":{"montantCentimes":1,"devise":"EUR"},"montant":1,"statut":"payee","echeance":"2099-01-01T00:00:00Z","reservationId":%q,"acheteurId":%q}`,
		article.ID, intrus, intrus, uuid.NewString())
	var commande commandeRecue
	b.commander(t, jeton, corps).exiger(t, http.StatusCreated).lire(t, &commande)
	exigerMontant(t, "prix", commande.Prix, 8000)
	exigerMontant(t, "frais de port", commande.FraisDePort, 700)
	exigerMontant(t, "total payé", commande.TotalPaye, 8700)
	exigerMontant(t, "commission", commande.Commission, 435)
	exigerMontant(t, "montant vendeur", commande.MontantVendeur, 8265)
	if commande.ID == intrus || commande.Statut != statutEnAttente || strings.HasPrefix(commande.Echeance, "2099") {
		t.Errorf("valeurs de la requête reprises : %+v", commande)
	}
	ligne := b.commandeEnBase(t, commande.ID)
	if ligne["reservation_id"] == intrus || ligne["acheteur_id"] != acheteur.sujet {
		t.Errorf("ligne en base reprenant la requête : %v", ligne)
	}
	if v, _ := ligne["total_paye_centimes"].(float64); int64(v) != 8700 {
		t.Errorf("total en base %v, 8700 attendu", ligne["total_paye_centimes"])
	}

	t.Run("corps envoyé à réserver ignoré par la Place de marché", func(t *testing.T) {
		article := b.articlePublie(t, 6000, 400)
		entetes := entetesDeService(b.kc.jetonDeTransactions(t), b.kc.seConnecter(t, b.kc.creerAcheteur(t)))
		entetes.Set("Content-Type", typeJSON)
		r := b.appeler(t, b.interne, http.MethodPost, cheminDeReservation(article.ID), []byte(`{"prix":{"montantCentimes":1,"devise":"EUR"},"fraisDePort":{"montantCentimes":0,"devise":"EUR"},"echeance":"2099-01-01T00:00:00Z"}`), entetes)
		var reservation reservationRecue
		r.exiger(t, http.StatusCreated).lire(t, &reservation)
		exigerMontant(t, "prix", reservation.Prix, 6000)
		exigerMontant(t, "frais de port", reservation.FraisDePort, 400)
		if strings.HasPrefix(reservation.Echeance, "2099") {
			t.Errorf("échéance de la requête reprise : %s", reservation.Echeance)
		}
	})
}

func (b *bancReservation) ac8BasesCloisonnees(t *testing.T) {
	for _, c := range []struct {
		nom    string
		role   string
		base   string
		depuis string
	}{
		{"transactions_execution vers marche", roleTransactionsExec, baseDuService, b.transactions.execution},
		{"transactions_migration vers marche", roleTransactionsMigr, baseDuService, b.transactions.migration},
		{"marche_execution vers transactions", roleExecution, baseTransactions, b.marche.execution},
		{"marche_migration vers transactions", roleMigration, baseTransactions, b.marche.migration},
	} {
		t.Run(c.nom, func(t *testing.T) {
			exigerConnexionRefusee(t, avecLaBase(c.depuis, c.base), c.nom)
		})
	}
	t.Run("transactions_execution ne modifie ni ne supprime une commande", func(t *testing.T) {
		conn := connecter(t.Context(), t, b.transactions.execution)
		var n int
		if err := conn.QueryRow(t.Context(), "SELECT count(*) FROM commande").Scan(&n); err != nil {
			t.Fatalf("lecture des commandes : %v", err)
		}
		for _, instruction := range []string{"UPDATE commande SET statut = statut", "DELETE FROM commande", "TRUNCATE commande", "DROP TABLE commande", "CREATE TABLE intrus (id integer)"} {
			_, err := conn.Exec(t.Context(), instruction)
			var erreur *pgconn.PgError
			if !errors.As(err, &erreur) || erreur.Code != droitInsuffisant {
				t.Errorf("%q : %v, refus 42501 attendu", instruction, err)
			}
		}
	})
}

func (b *bancReservation) ac9JetonDeServiceSurRoutesAcheteur(t *testing.T) {
	article := b.articlePublie(t, 2200, 300)
	service := b.kc.jetonDeTransactions(t)
	exigerRefusDAutorisation(t, "POST commandes", b.commander(t, service, corpsDeCommande(article.ID)))
	exigerRefusDAutorisation(t, "POST commandes avec l'identité d'acheteur du jeton de service", b.commander(t, service, corpsDeCommande(article.ID), http.Header{enteteJetonAcheteur: {service}}))
	acheteur := b.kc.seConnecter(t, b.kc.creerAcheteur(t))
	commande := b.commanderAvecSucces(t, acheteur, article.ID, http.StatusCreated)
	exigerRefusDAutorisation(t, "GET commande", b.lireCommande(t, service, commande.ID))
	exigerRefusDAutorisation(t, "GET commande inexistante", b.lireCommande(t, service, uuid.NewString()))
	exigerRefusDAutorisation(t, "GET mes articles de la Place de marché", b.envoyer(t, http.MethodGet, cheminArticles, bearer(service)))
	if n := b.compterCommandes(t, "WHERE article_id::text = $1", article.ID); n != 1 {
		t.Errorf("%d commandes pour l'article, seule celle de l'acheteur attendue", n)
	}
}

func (b *bancReservation) routeInterneAbsenteDuPortPublic(t *testing.T) {
	article := b.articlePublie(t, 2300, 300)
	jeton := b.kc.seConnecter(t, b.kc.creerAcheteur(t))
	requete, err := http.NewRequestWithContext(t.Context(), http.MethodPost, b.public+cheminDeReservation(article.ID), nil)
	if err != nil {
		t.Fatalf("requête : %v", err)
	}
	requete.Header = entetesDeService(b.kc.jetonDeTransactions(t), jeton)
	if r := b.executer(t, requete); r.statut != http.StatusNotFound && r.statut != http.StatusMethodNotAllowed {
		t.Errorf("réserver sur le port public : %d, route absente attendue", r.statut)
	}
	b.exigerNonReserve(t, article.ID)
	for _, chemin := range []string{cheminCatalogue, cheminCatalogue + "/" + article.ID, cheminCategories} {
		requete, err := http.NewRequestWithContext(t.Context(), http.MethodGet, b.interne+chemin, nil)
		if err != nil {
			t.Fatalf("requête : %v", err)
		}
		if r := b.executer(t, requete); r.statut != http.StatusNotFound && r.statut != http.StatusMethodNotAllowed {
			t.Errorf("GET %s sur le port interne : %d, route absente attendue", chemin, r.statut)
		}
	}
}

func (b *bancReservation) jetonAcheteurNonJournalise(t *testing.T) {
	article := b.articlePublie(t, 2400, 300)
	jeton := b.kc.seConnecter(t, b.kc.creerAcheteur(t))
	b.commanderAvecSucces(t, jeton, article.ID, http.StatusCreated)
	signature := jeton[strings.LastIndex(jeton, ".")+1:]
	if b.tx.journal.contient(signature) {
		t.Errorf("jeton de l'acheteur présent dans le journal de Transactions")
	}
	if b.journal.contient(signature) {
		t.Errorf("jeton de l'acheteur présent dans le journal de la Place de marché")
	}
	if strings.Contains(b.tx.journal.tampon.String(), b.kc.secretTransactions) {
		t.Errorf("secret du client transactions présent dans le journal")
	}
}

func chargerContratDe(t *testing.T, base, fichier, prefixe string) routers.Router {
	t.Helper()
	contrat, err := openapi3.NewLoader().LoadFromFile(filepath.Join("..", "..", "..", "..", "contracts", "api", fichier))
	if err != nil {
		t.Fatalf("chargement de %s : %v", fichier, err)
	}
	if err := contrat.Validate(t.Context()); err != nil {
		t.Fatalf("contrat %s invalide : %v", fichier, err)
	}
	contrat.Servers = openapi3.Servers{{URL: base + prefixe}}
	routeur, err := gorillamux.NewRouter(contrat)
	if err != nil {
		t.Fatalf("routeur de %s : %v", fichier, err)
	}
	return routeur
}

func statutsDe(echanges []echange) []int {
	var statuts []int
	for _, e := range echanges {
		if !slices.Contains(statuts, e.reponse.statut) {
			statuts = append(statuts, e.reponse.statut)
		}
	}
	slices.Sort(statuts)
	return statuts
}

func (b *bancReservation) reponsesConformesAuxContrats(t *testing.T) {
	b.mu.Lock()
	tx, internes := append([]echange(nil), b.echangesTx...), append([]echange(nil), b.echangesInt...)
	b.mu.Unlock()
	for _, a := range b.relais.tous() {
		requete, err := http.NewRequestWithContext(t.Context(), a.methode, b.interne+a.chemin, http.NoBody)
		if err != nil {
			t.Fatalf("requête relayée : %v", err)
		}
		requete.Header = a.entetes
		internes = append(internes, echange{requete: requete, reponse: reponse{statut: a.statut, entetes: a.recu, corps: a.reponse}})
	}
	var internesAuContrat []echange
	for _, e := range internes {
		if e.reponse.statut != http.StatusOK && e.reponse.statut != http.StatusCreated && len(e.reponse.corps) == 0 {
			continue
		}
		internesAuContrat = append(internesAuContrat, e)
	}
	t.Logf("Transactions : %d échanges, statuts %v ; Place de marché interne : %d échanges, statuts %v", len(tx), statutsDe(tx), len(internesAuContrat), statutsDe(internesAuContrat))
	routeurTx := chargerContratDe(t, "http://"+b.tx.api, filepath.Join("transactions", "v1.openapi.yaml"), "/transactions/v1")
	for _, e := range tx {
		verifierEchangeContrat(t, routeurTx, e)
	}
	routeurInterne := chargerContratDe(t, b.interne, filepath.Join("place-de-marche", "interne-v1.openapi.yaml"), "/interne/v1")
	for _, e := range internesAuContrat {
		verifierEchangeContrat(t, routeurInterne, e)
	}
	routeurPublic := chargerContrat(t, b.public)
	catalogue := 0
	for _, e := range b.historique() {
		if strings.HasPrefix(e.requete.URL.Path, cheminCatalogue) {
			catalogue++
			verifierEchange(t, routeurPublic, e)
		}
	}
	t.Logf("catalogue public : %d échanges vérifiés", catalogue)
}

func verifierEchangeContrat(t *testing.T, routeur routers.Router, ech echange) {
	t.Helper()
	route, parametres, err := routeur.FindRoute(ech.requete)
	if err != nil {
		if ech.reponse.statut == http.StatusNotFound {
			return
		}
		t.Errorf("%s %s absent du contrat : %v", ech.requete.Method, ech.requete.URL, err)
		return
	}
	options := &openapi3filter.Options{IncludeResponseStatus: true, AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}
	entree := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: ech.requete, PathParams: parametres, Route: route, Options: options},
		Status:                 ech.reponse.statut,
		Header:                 ech.reponse.entetes,
		Body:                   io.NopCloser(bytes.NewReader(ech.reponse.corps)),
		Options:                options,
	}
	if err := openapi3filter.ValidateResponse(t.Context(), entree); err != nil {
		t.Errorf("%s %s → %d non conforme au contrat : %v\n%s", ech.requete.Method, ech.requete.URL.Path, ech.reponse.statut, err, ech.reponse.corps)
	}
}
