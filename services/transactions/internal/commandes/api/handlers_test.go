package api_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/libs/platform/token"
	"collector/services/transactions/internal/commandes"
	"collector/services/transactions/internal/commandes/api"
	"collector/services/transactions/internal/montants"
	"collector/services/transactions/internal/problemes"
)

const (
	emetteur      = "https://auth.collector.test/realms/collector"
	audience      = "transactions"
	clientDuSite  = "collector-web"
	cheminCommand = "/transactions/v1/commandes"
	typeProbleme  = "application/problem+json"
)

type emetteurDeTest struct {
	t   *testing.T
	cle *rsa.PrivateKey
	jwk *httptest.Server
}

func b64(octets []byte) string {
	return base64.RawURLEncoding.EncodeToString(octets)
}

func nouvelEmetteur(t *testing.T) *emetteurDeTest {
	t.Helper()
	cle, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("clé : %v", err)
	}
	jwks, err := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "k1",
		"n": b64(cle.N.Bytes()), "e": b64(big.NewInt(int64(cle.E)).Bytes()),
	}}})
	if err != nil {
		t.Fatalf("JWKS : %v", err)
	}
	serveur := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwks)
	}))
	t.Cleanup(serveur.Close)
	return &emetteurDeTest{t: t, cle: cle, jwk: serveur}
}

func (e *emetteurDeTest) jeton(sujet string, modifier func(map[string]any), roles ...string) string {
	e.t.Helper()
	revendications := map[string]any{
		"iss": emetteur, "aud": []string{"place-de-marche", audience}, "azp": clientDuSite, "typ": "Bearer", "sub": sujet,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(),
		"realm_access": map[string]any{"roles": roles},
	}
	modifier(revendications)
	entete, _ := json.Marshal(map[string]any{"alg": "RS256", "typ": "JWT", "kid": "k1"})
	charge, _ := json.Marshal(revendications)
	signe := b64(entete) + "." + b64(charge)
	empreinte := sha256.Sum256([]byte(signe))
	signature, err := rsa.SignPKCS1v15(rand.Reader, e.cle, crypto.SHA256, empreinte[:])
	if err != nil {
		e.t.Fatalf("signature : %v", err)
	}
	return signe + "." + b64(signature)
}

func (e *emetteurDeTest) acheteur(sujet string) string {
	return e.jeton(sujet, func(map[string]any) {}, "acheteur")
}

type appelPlace struct {
	acheteur   commandes.BuyerID
	credential token.Credential
	article    commandes.ArticleID
}

type appelListe struct {
	acheteur commandes.BuyerID
	page     commandes.PageRequest
}

type commandesSimulees struct {
	mu          sync.Mutex
	appels      []appelPlace
	listes      []appelListe
	resultat    commandes.Recorded
	erreur      error
	commandes   map[commandes.ID]commandes.Order
	page        commandes.OrderPage
	erreurListe error
}

func (c *commandesSimulees) Place(_ context.Context, acheteur commandes.BuyerID, credential token.Credential, article commandes.ArticleID) (commandes.Recorded, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.appels = append(c.appels, appelPlace{acheteur: acheteur, credential: credential, article: article})
	return c.resultat, c.erreur
}

func (c *commandesSimulees) OwnOrder(_ context.Context, acheteur commandes.BuyerID, id commandes.ID) (commandes.Order, error) {
	commande, trouvee := c.commandes[id]
	if !trouvee || commande.Buyer != acheteur {
		return commandes.Order{}, commandes.ErrOrderNotFound
	}
	return commande, nil
}

func (c *commandesSimulees) OwnOrders(_ context.Context, acheteur commandes.BuyerID, page commandes.PageRequest) (commandes.OrderPage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listes = append(c.listes, appelListe{acheteur: acheteur, page: page})
	return c.page, c.erreurListe
}

type limiteur struct {
	refuser bool
}

func (l *limiteur) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.refuser {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type banc struct {
	emetteur  *emetteurDeTest
	commandes *commandesSimulees
	limiteur  *limiteur
	mux       *http.ServeMux
}

func nouveauBanc(t *testing.T) *banc {
	t.Helper()
	e := nouvelEmetteur(t)
	verificateur, err := token.NewVerifier(t.Context(), token.Config{Issuer: emetteur, Audience: audience, AuthorizedParty: clientDuSite, KeySetURL: e.jwk.URL})
	if err != nil {
		t.Fatalf("vérificateur : %v", err)
	}
	b := &banc{emetteur: e, commandes: &commandesSimulees{commandes: map[commandes.ID]commandes.Order{}}, limiteur: &limiteur{}, mux: http.NewServeMux()}
	api.Register(b.mux, b.commandes, verificateur, b.limiteur, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return b
}

type reponse struct {
	statut  int
	entetes http.Header
	corps   []byte
}

func (b *banc) appeler(t *testing.T, methode, chemin, jeton, corps string) reponse {
	t.Helper()
	requete := httptest.NewRequestWithContext(t.Context(), methode, chemin, strings.NewReader(corps))
	if corps != "" {
		requete.Header.Set("Content-Type", "application/json")
	}
	if jeton != "" {
		requete.Header.Set("Authorization", "Bearer "+jeton)
	}
	enregistreur := httptest.NewRecorder()
	b.mux.ServeHTTP(enregistreur, requete)
	return reponse{statut: enregistreur.Code, entetes: enregistreur.Header(), corps: enregistreur.Body.Bytes()}
}

func commandeDe(t *testing.T, acheteur commandes.BuyerID, article commandes.ArticleID) commandes.Order {
	t.Helper()
	detail, err := montants.Compute(montants.Money{Cents: 12000, Currency: montants.Euro}, montants.Money{Cents: 590, Currency: montants.Euro})
	if err != nil {
		t.Fatalf("montants : %v", err)
	}
	id, _ := commandes.ParseID(uuid.NewString())
	reservation, _ := commandes.ParseReservationID(uuid.NewString())
	return commandes.Order{
		ID: id, Article: article, Title: "Figurine de collection", Buyer: acheteur, Reservation: reservation, Status: commandes.StatusAwaitingPayment, Amounts: detail,
		ExpiresAt: time.Date(2026, 11, 16, 10, 25, 0, 123456000, time.UTC), CreatedAt: time.Date(2026, 11, 16, 10, 15, 0, 0, time.UTC),
	}
}

type montantRecu struct {
	MontantCentimes int64  `json:"montantCentimes"`
	Devise          string `json:"devise"`
}

type commandeRecue struct {
	ID             string      `json:"id"`
	ArticleID      string      `json:"articleId"`
	Titre          string      `json:"titre"`
	Statut         string      `json:"statut"`
	Prix           montantRecu `json:"prix"`
	FraisDePort    montantRecu `json:"fraisDePort"`
	TotalPaye      montantRecu `json:"totalPaye"`
	Commission     montantRecu `json:"commission"`
	MontantVendeur montantRecu `json:"montantVendeur"`
	Echeance       string      `json:"echeance"`
	CreeLe         string      `json:"creeLe"`
}

type problemeRecu struct {
	Type   string `json:"type"`
	Status int    `json:"status"`
}

func lire(t *testing.T, r reponse, cible any) {
	t.Helper()
	if err := json.Unmarshal(r.corps, cible); err != nil {
		t.Fatalf("réponse illisible (%d) : %s", r.statut, r.corps)
	}
}

func identifiants(t *testing.T) (string, commandes.BuyerID, commandes.ArticleID) {
	t.Helper()
	sujet := uuid.NewString()
	acheteur, err := commandes.ParseBuyerID(sujet)
	if err != nil {
		t.Fatalf("acheteur : %v", err)
	}
	article, err := commandes.ParseArticleID(uuid.NewString())
	if err != nil {
		t.Fatalf("article : %v", err)
	}
	return sujet, acheteur, article
}

func TestCommandeCreeeAvecLocationEtMontants(t *testing.T) {
	b := nouveauBanc(t)
	sujet, acheteur, article := identifiants(t)
	jeton := b.emetteur.acheteur(sujet)
	commande := commandeDe(t, acheteur, article)
	b.commandes.resultat = commandes.Recorded{Order: commande, Created: true}
	r := b.appeler(t, http.MethodPost, cheminCommand, jeton, `{"articleId":"`+article.String()+`"}`)
	if r.statut != http.StatusCreated || r.entetes.Get("Location") != cheminCommand+"/"+commande.ID.String() {
		t.Fatalf("statut %d, Location %q : %s", r.statut, r.entetes.Get("Location"), r.corps)
	}
	var recue commandeRecue
	lire(t, r, &recue)
	attendus := map[string]montantRecu{"prix": {12000, "EUR"}, "frais": {590, "EUR"}, "total": {12590, "EUR"}, "commission": {630, "EUR"}, "vendeur": {11960, "EUR"}}
	obtenus := map[string]montantRecu{"prix": recue.Prix, "frais": recue.FraisDePort, "total": recue.TotalPaye, "commission": recue.Commission, "vendeur": recue.MontantVendeur}
	for nom, attendu := range attendus {
		if obtenus[nom] != attendu {
			t.Errorf("%s %+v, %+v attendu", nom, obtenus[nom], attendu)
		}
	}
	if recue.ID != commande.ID.String() || recue.ArticleID != article.String() || recue.Statut != "en_attente_paiement" || recue.Echeance != "2026-11-16T10:25:00.123456Z" {
		t.Errorf("commande inattendue : %+v", recue)
	}
	if len(b.commandes.appels) != 1 {
		t.Fatalf("%d appels du service, 1 attendu", len(b.commandes.appels))
	}
	appel := b.commandes.appels[0]
	if appel.acheteur != acheteur || appel.article != article || appel.credential.Reveal() != jeton {
		t.Errorf("service appelé pour %v, article %v, jeton identique %t", appel.acheteur, appel.article, appel.credential.Reveal() == jeton)
	}
}

func TestNouvelEssaiRenvoieLaCommandeExistanteEn200(t *testing.T) {
	b := nouveauBanc(t)
	sujet, acheteur, article := identifiants(t)
	commande := commandeDe(t, acheteur, article)
	b.commandes.resultat = commandes.Recorded{Order: commande, Created: false}
	r := b.appeler(t, http.MethodPost, cheminCommand, b.emetteur.acheteur(sujet), `{"articleId":"`+article.String()+`"}`)
	var recue commandeRecue
	lire(t, r, &recue)
	if r.statut != http.StatusOK || recue.ID != commande.ID.String() {
		t.Fatalf("statut %d, commande %s ; 200 et %s attendus", r.statut, recue.ID, commande.ID)
	}
}

func TestChampsInconnusIgnoresAcheteurDuJeton(t *testing.T) {
	b := nouveauBanc(t)
	sujet, acheteur, article := identifiants(t)
	b.commandes.resultat = commandes.Recorded{Order: commandeDe(t, acheteur, article), Created: true}
	corps := `{"articleId":"` + article.String() + `","prix":{"montantCentimes":1,"devise":"EUR"},"commission":0,"acheteurId":"` + uuid.NewString() + `","statut":"payee"}`
	r := b.appeler(t, http.MethodPost, cheminCommand, b.emetteur.acheteur(sujet), corps)
	if r.statut != http.StatusCreated {
		t.Fatalf("statut %d : %s", r.statut, r.corps)
	}
	if appel := b.commandes.appels[0]; appel.acheteur != acheteur || appel.article != article {
		t.Fatalf("service appelé pour %v et %v, acheteur du jeton et article demandé attendus", appel.acheteur, appel.article)
	}
}

func TestSaisieInvalide(t *testing.T) {
	b := nouveauBanc(t)
	sujet, _, _ := identifiants(t)
	jeton := b.emetteur.acheteur(sujet)
	cas := map[string]struct {
		corps  string
		statut int
		urn    string
	}{
		"articleId absent":       {`{}`, http.StatusBadRequest, "urn:collector:probleme:saisie-invalide"},
		"articleId vide":         {`{"articleId":""}`, http.StatusBadRequest, "urn:collector:probleme:saisie-invalide"},
		"articleId non UUID":     {`{"articleId":"pas-un-uuid"}`, http.StatusBadRequest, "urn:collector:probleme:saisie-invalide"},
		"articleId d'autre type": {`{"articleId":3}`, http.StatusBadRequest, "urn:collector:probleme:corps-invalide"},
		"clé dupliquée":          {`{"articleId":"a","articleId":"b"}`, http.StatusBadRequest, "urn:collector:probleme:corps-invalide"},
		"JSON mal formé":         {`{"articleId":`, http.StatusBadRequest, "urn:collector:probleme:corps-invalide"},
		"tableau":                {`[]`, http.StatusBadRequest, "urn:collector:probleme:corps-invalide"},
		"corps au-delà de 4 Kio": {`{"articleId":"` + strings.Repeat("a", 4100) + `"}`, http.StatusRequestEntityTooLarge, ""},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			r := b.appeler(t, http.MethodPost, cheminCommand, jeton, c.corps)
			var p problemeRecu
			lire(t, r, &p)
			if r.statut != c.statut || (c.urn != "" && p.Type != c.urn) || r.entetes.Get("Content-Type") != typeProbleme {
				t.Fatalf("statut %d type %q, attendus %d %q", r.statut, p.Type, c.statut, c.urn)
			}
		})
	}
	requete := httptest.NewRequestWithContext(t.Context(), http.MethodPost, cheminCommand, strings.NewReader(`{}`))
	requete.Header.Set("Authorization", "Bearer "+jeton)
	requete.Header.Set("Content-Type", "text/plain")
	enregistreur := httptest.NewRecorder()
	b.mux.ServeHTTP(enregistreur, requete)
	if enregistreur.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("text/plain : %d, 415 attendu", enregistreur.Code)
	}
	if len(b.commandes.appels) != 0 {
		t.Fatalf("service appelé malgré une saisie invalide")
	}
}

func TestErreursMetierTraduites(t *testing.T) {
	cas := map[string]struct {
		erreur error
		statut int
		urn    string
	}{
		"non réservable":          {commandes.ErrArticleNotReservable, http.StatusUnprocessableEntity, problemes.ArticleNotReservable},
		"déjà réservé":            {commandes.ErrArticleAlreadyReserved, http.StatusConflict, problemes.ArticleAlreadyReserved},
		"réservation active":      {commandes.ErrBuyerAlreadyHoldsAnother, http.StatusConflict, problemes.ReservationActive},
		"place de marché absente": {commandes.ErrMarketplaceUnavailable, http.StatusServiceUnavailable, ""},
		"erreur enveloppée":       {errors.Join(errors.New("contexte"), commandes.ErrArticleAlreadyReserved), http.StatusConflict, problemes.ArticleAlreadyReserved},
		"erreur inattendue":       {errors.New("base perdue"), http.StatusInternalServerError, ""},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			b := nouveauBanc(t)
			sujet, _, article := identifiants(t)
			b.commandes.erreur = c.erreur
			r := b.appeler(t, http.MethodPost, cheminCommand, b.emetteur.acheteur(sujet), `{"articleId":"`+article.String()+`"}`)
			var p problemeRecu
			lire(t, r, &p)
			if r.statut != c.statut || p.Status != c.statut || (c.urn != "" && p.Type != c.urn) {
				t.Fatalf("statut %d type %q, attendus %d %q", r.statut, p.Type, c.statut, c.urn)
			}
			if strings.Contains(string(r.corps), "base perdue") {
				t.Fatalf("détail technique divulgué : %s", r.corps)
			}
		})
	}
}

func TestLimiteDeDebitAppliqueeAuxCommandes(t *testing.T) {
	b := nouveauBanc(t)
	sujet, _, article := identifiants(t)
	b.limiteur.refuser = true
	r := b.appeler(t, http.MethodPost, cheminCommand, b.emetteur.acheteur(sujet), `{"articleId":"`+article.String()+`"}`)
	if r.statut != http.StatusTooManyRequests || len(b.commandes.appels) != 0 {
		t.Fatalf("statut %d, %d appels du service ; 429 sans appel attendu", r.statut, len(b.commandes.appels))
	}
	if anonyme := b.appeler(t, http.MethodPost, cheminCommand, "", `{"articleId":"`+article.String()+`"}`); anonyme.statut != http.StatusUnauthorized {
		t.Fatalf("sans jeton : %d, 401 attendu avant la limite", anonyme.statut)
	}
}

func TestLectureDeSaCommandeSeulement(t *testing.T) {
	b := nouveauBanc(t)
	sujet, acheteur, article := identifiants(t)
	commande := commandeDe(t, acheteur, article)
	b.commandes.commandes[commande.ID] = commande
	r := b.appeler(t, http.MethodGet, cheminCommand+"/"+commande.ID.String(), b.emetteur.acheteur(sujet), "")
	var recue commandeRecue
	lire(t, r, &recue)
	if r.statut != http.StatusOK || recue.ID != commande.ID.String() {
		t.Fatalf("propriétaire : %d %+v", r.statut, recue)
	}
	autre := b.appeler(t, http.MethodGet, cheminCommand+"/"+commande.ID.String(), b.emetteur.acheteur(uuid.NewString()), "")
	inexistante := b.appeler(t, http.MethodGet, cheminCommand+"/"+uuid.NewString(), b.emetteur.acheteur(sujet), "")
	malFormee := b.appeler(t, http.MethodGet, cheminCommand+"/pas-un-uuid", b.emetteur.acheteur(sujet), "")
	for nom, rep := range map[string]reponse{"autre acheteur": autre, "inexistante": inexistante, "mal formée": malFormee} {
		if rep.statut != http.StatusNotFound {
			t.Errorf("%s : %d, 404 attendu", nom, rep.statut)
		}
	}
	if string(autre.corps) != string(inexistante.corps) {
		t.Errorf("réponses différentes pour une commande d'autrui et une inexistante : %s / %s", autre.corps, inexistante.corps)
	}
}

func TestAuthentificationEtRoleAcheteur(t *testing.T) {
	b := nouveauBanc(t)
	sujet, _, article := identifiants(t)
	corps := `{"articleId":"` + article.String() + `"}`
	cas := map[string]struct {
		jeton  string
		statut int
	}{
		"sans jeton":      {"", http.StatusUnauthorized},
		"jeton illisible": {"pas-un-jeton", http.StatusUnauthorized},
		"autre audience":  {b.emetteur.jeton(sujet, func(r map[string]any) { r["aud"] = "place-de-marche" }, "acheteur"), http.StatusUnauthorized},
		"jeton de service": {b.emetteur.jeton(sujet, func(r map[string]any) {
			r["azp"] = "transactions"
			r["aud"] = "place-de-marche"
			r["scope"] = "reserver"
			delete(r, "realm_access")
		}), http.StatusUnauthorized},
		"expiré":                {b.emetteur.jeton(sujet, func(r map[string]any) { r["exp"] = int64(1) }, "acheteur"), http.StatusUnauthorized},
		"sans le rôle acheteur": {b.emetteur.jeton(sujet, func(map[string]any) {}, "vendeur"), http.StatusForbidden},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			for _, r := range []reponse{
				b.appeler(t, http.MethodPost, cheminCommand, c.jeton, corps),
				b.appeler(t, http.MethodGet, cheminCommand+"/"+uuid.NewString(), c.jeton, ""),
			} {
				if r.statut != c.statut {
					t.Fatalf("statut %d, %d attendu", r.statut, c.statut)
				}
			}
		})
	}
	if len(b.commandes.appels) != 0 {
		t.Fatalf("service appelé sans acheteur authentifié")
	}
}

func TestSujetQuiNEstPasUnUUIDRefuse(t *testing.T) {
	b := nouveauBanc(t)
	_, _, article := identifiants(t)
	r := b.appeler(t, http.MethodPost, cheminCommand, b.emetteur.acheteur("compte-sans-uuid"), `{"articleId":"`+article.String()+`"}`)
	if r.statut < 400 || r.statut >= 500 || len(b.commandes.appels) != 0 {
		t.Fatalf("sujet non UUID : %d, %d appels ; refus sans appel attendu", r.statut, len(b.commandes.appels))
	}
	t.Logf("sujet non UUID : %d", r.statut)
}
