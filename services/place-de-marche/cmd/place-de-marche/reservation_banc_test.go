package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"collector/libs/platform/logging"
	"collector/services/place-de-marche/internal/migrations"
)

const (
	baseTransactions        = "transactions"
	roleTransactionsMigr    = "transactions_migration"
	roleTransactionsExec    = "transactions_execution"
	cheminCommandes         = "/transactions/v1/commandes"
	enteteJetonAcheteur     = "X-Collector-User-Token"
	dureeDeReservation      = 45 * time.Minute
	plafondDeReservation    = 20 * time.Minute
	statutEnAttente         = "en_attente_paiement"
	typeNonReservable       = "urn:collector:probleme:article-non-reservable"
	typeDejaReserve         = "urn:collector:probleme:article-deja-reserve"
	typeReservationActive   = "urn:collector:probleme:reservation-deja-active"
	typeSaisieInvalideTx    = "urn:collector:probleme:saisie-invalide"
	toleranceEcheance       = 5 * time.Second
	delaiDePublicationResa  = 30 * time.Second
	categorieDeLaReservaton = categorieSure
)

type appelRelaye struct {
	chemin  string
	methode string
	entetes http.Header
	statut  int
	reponse []byte
	recu    http.Header
}

type relaisEspion struct {
	mu     sync.Mutex
	appels []appelRelaye
	url    string
}

type processusTransactions struct {
	api       string
	gestion   string
	journal   *journalPartage
	processus *exec.Cmd
	termine   chan error
	une       sync.Once
}

type commandeRecue struct {
	ID             string   `json:"id"`
	ArticleID      string   `json:"articleId"`
	Titre          string   `json:"titre"`
	Statut         string   `json:"statut"`
	Prix           *montant `json:"prix"`
	FraisDePort    *montant `json:"fraisDePort"`
	TotalPaye      *montant `json:"totalPaye"`
	Commission     *montant `json:"commission"`
	MontantVendeur *montant `json:"montantVendeur"`
	Echeance       string   `json:"echeance"`
	CreeLe         string   `json:"creeLe"`
}

type reservationRecue struct {
	ReservationID string   `json:"reservationId"`
	ArticleID     string   `json:"articleId"`
	Titre         string   `json:"titre"`
	Echeance      string   `json:"echeance"`
	Prix          *montant `json:"prix"`
	FraisDePort   *montant `json:"fraisDePort"`
}

type entreeReservable struct {
	ID      string `json:"id"`
	Reserve *bool  `json:"reserve"`
}

type pageReservable struct {
	Elements []entreeReservable `json:"elements"`
	Total    int                `json:"total"`
}

type bancReservation struct {
	*environnement
	kc            *fournisseurKeycloak
	marche        basePostgreSQL
	transactions  basePostgreSQL
	adresseMaitre string
	interne       string
	public        string
	tx            *processusTransactions
	relais        *relaisEspion
	journal       *journalPartage
	mu            sync.Mutex
	echangesTx    []echange
	echangesInt   []echange
}

func amorcerBaseDeService(ctx context.Context, t *testing.T, adresseMaitre, nom, roleMigr, roleExec string) basePostgreSQL {
	t.Helper()
	maitre := connecter(ctx, t, adresseMaitre)
	secretMigration, secretExecution := secretAleatoire(t), secretAleatoire(t)
	executerFormate(ctx, t, maitre, "CREATE ROLE %I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD %L", roleMigr, secretMigration)
	executerFormate(ctx, t, maitre, "CREATE ROLE %I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD %L", roleExec, secretExecution)
	executerFormate(ctx, t, maitre, "CREATE DATABASE %I OWNER %I", nom, roleMigr)
	executerFormate(ctx, t, maitre, "REVOKE CONNECT ON DATABASE %I FROM PUBLIC", nom)
	executerFormate(ctx, t, maitre, "GRANT CONNECT ON DATABASE %I TO %I, %I", nom, roleMigr, roleExec)
	base := basePostgreSQL{
		maitre:    connecter(ctx, t, versUneBase(t, adresseMaitre, nom, enTantQueMaitre)),
		migration: versUneBase(t, adresseMaitre, nom, enTantQue(roleMigr, secretMigration)),
		execution: versUneBase(t, adresseMaitre, nom, enTantQue(roleExec, secretExecution)),
	}
	if _, err := connecter(ctx, t, base.migration).Exec(ctx, "REVOKE CREATE ON SCHEMA public FROM PUBLIC"); err != nil {
		t.Fatalf("révocation de CREATE sur public de %s : %v", nom, err)
	}
	return base
}

func migrerTransactions(t *testing.T, binaire string, base basePostgreSQL) {
	t.Helper()
	for range 2 {
		migration := exec.Command(binaire, "migrate")
		migration.Env = environnementDuProcessus(map[string]string{"BASE_DE_DONNEES_URL": base.migration, "BASE_DE_DONNEES_TLS_EXIGE": "false"})
		if sortie, err := migration.CombinedOutput(); err != nil {
			t.Fatalf("migrations de Transactions : %v\n%s", err, sortie)
		}
	}
}

func demarrerRelaisEspion(t *testing.T, cible string) *relaisEspion {
	t.Helper()
	destination, err := url.Parse(cible)
	if err != nil {
		t.Fatalf("adresse interne : %v", err)
	}
	r := &relaisEspion{}
	mandataire := httputil.NewSingleHostReverseProxy(destination)
	mandataire.ModifyResponse = func(brute *http.Response) error {
		contenu, err := io.ReadAll(brute.Body)
		_ = brute.Body.Close()
		if err != nil {
			return err
		}
		brute.Body = io.NopCloser(bytes.NewReader(contenu))
		requete := brute.Request
		r.mu.Lock()
		r.appels = append(r.appels, appelRelaye{
			chemin:  requete.URL.Path,
			methode: requete.Method,
			entetes: requete.Header.Clone(),
			statut:  brute.StatusCode,
			reponse: contenu,
			recu:    brute.Header.Clone(),
		})
		r.mu.Unlock()
		return nil
	}
	serveur := httptest.NewServer(mandataire)
	t.Cleanup(serveur.Close)
	r.url = serveur.URL
	return r
}

func (r *relaisEspion) appelsVers(articleID string) []appelRelaye {
	r.mu.Lock()
	defer r.mu.Unlock()
	var trouves []appelRelaye
	for _, a := range r.appels {
		if strings.Contains(a.chemin, articleID) {
			trouves = append(trouves, a)
		}
	}
	return trouves
}

func (r *relaisEspion) tous() []appelRelaye {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]appelRelaye(nil), r.appels...)
}

func lancerTransactions(t *testing.T, binaire string, base basePostgreSQL, kc *fournisseurKeycloak, placeDeMarche string) *processusTransactions {
	t.Helper()
	p := &processusTransactions{api: adresseLibre(t), gestion: adresseLibre(t), journal: &journalPartage{}, termine: make(chan error, 1)}
	p.processus = exec.Command(binaire, "serve")
	p.processus.Env = environnementDuProcessus(map[string]string{
		"BASE_DE_DONNEES_URL":         base.execution,
		"BASE_DE_DONNEES_TLS_EXIGE":   "false",
		"OIDC_EMETTEUR":               kc.emetteur(),
		"OIDC_CLIENT_SECRET":          kc.secretTransactions,
		"PLACE_DE_MARCHE_URL_INTERNE": placeDeMarche,
		"ADRESSE_API":                 p.api,
		"ADRESSE_GESTION":             p.gestion,
		"LIMITE_DEBIT_PAR_MINUTE":     "6000000",
		"LIMITE_DEBIT_RAFALE":         "100000",
		"JOURNAL_NIVEAU":              "DEBUG",
	})
	p.processus.Stdout, p.processus.Stderr = p.journal, p.journal
	if err := p.processus.Start(); err != nil {
		t.Fatalf("lancement de Transactions : %v", err)
	}
	go func() { p.termine <- p.processus.Wait() }()
	t.Cleanup(func() {
		p.une.Do(func() {
			_ = p.processus.Process.Kill()
			select {
			case erreur := <-p.termine:
				t.Logf("arrêt de Transactions : %v", erreur)
			case <-time.After(10 * time.Second):
				t.Errorf("Transactions ne s'est pas arrêté en 10 s")
			}
		})
	})
	client := &http.Client{Timeout: 2 * time.Second}
	attendre(t, time.Minute, "démarrage de Transactions", func() bool {
		select {
		case err := <-p.termine:
			t.Fatalf("Transactions s'est arrêté : %v\n%s", err, p.journal.tampon.String())
		default:
		}
		return repond(client, "http://"+p.gestion+"/disponibilite")
	})
	return p
}

func lancerPlaceDeMarcheSurKeycloak(t *testing.T, canal *canalDeTest, adresseBase string, kc *fournisseurKeycloak, duree, plafond time.Duration) (string, string, *journalPartage) {
	t.Helper()
	avecLesVariables(t, nouveauBucket(t).variables())
	api, gestion, interne := adresseLibre(t), adresseLibre(t), adresseLibre(t)
	variables := canal.environnementDeService(identiteMarche)
	variables["BASE_DE_DONNEES_URL"] = adresseBase
	variables["BASE_DE_DONNEES_TLS_EXIGE"] = "false"
	variables["OIDC_EMETTEUR"] = kc.emetteur()
	variables["OIDC_JEU_DE_CLES_URL"] = kc.emetteur() + "/protocol/openid-connect/certs"
	variables["CANAL_EMETTEUR_ATTENDU"] = identiteControle
	variables["ADRESSE_API"] = api
	variables["ADRESSE_GESTION"] = gestion
	variables["ADRESSE_INTERNE"] = interne
	variables["RESERVATION_DUREE"] = duree.String()
	variables["RESERVATION_DUREE_MAX"] = plafond.String()
	variables["LIMITE_DEBIT_PAR_MINUTE"] = "6000000"
	variables["LIMITE_DEBIT_RAFALE"] = "100000"
	for nom, valeur := range variables {
		t.Setenv(nom, valeur)
	}
	journal := &journalPartage{}
	ctx, arreter := context.WithCancel(context.Background())
	termine := make(chan error, 1)
	go func() { termine <- serve(ctx, logging.New(serviceName, journal, slog.LevelInfo)) }()
	t.Cleanup(func() {
		arreter()
		select {
		case erreur := <-termine:
			t.Logf("arrêt de la Place de marché : %v", erreur)
		case <-time.After(15 * time.Second):
			t.Error("la Place de marché ne s'est pas arrêtée")
		}
	})
	client := &http.Client{Timeout: 5 * time.Second}
	attendre(t, 30*time.Second, "démarrage de la Place de marché", func() bool {
		select {
		case err := <-termine:
			t.Fatalf("la Place de marché s'est arrêtée : %v\n%s", err, journal.tampon.String())
		default:
		}
		return repond(client, "http://"+gestion+"/disponibilite")
	})
	return "http://" + api, "http://" + interne, journal
}

func demarrerBancReservation(t *testing.T) *bancReservation {
	t.Helper()
	return demarrerBancReservationAvec(t, dureeDeReservation, plafondDeReservation)
}

func demarrerBancReservationAvec(t *testing.T, duree, plafond time.Duration) *bancReservation {
	t.Helper()
	ctx := contexteBorne(t, 10*time.Minute)
	kc := demarrerKeycloakDuRealm(t)
	adresseMaitre := demarrerPostgreSQL(t)
	marche := amorcer(ctx, t, adresseMaitre)
	if err := migrations.Apply(ctx, journalMuet(), marche.migration); err != nil {
		t.Fatalf("migrations de la Place de marché : %v", err)
	}
	transactions := amorcerBaseDeService(ctx, t, adresseMaitre, baseTransactions, roleTransactionsMigr, roleTransactionsExec)
	binaire := construireBinaire(t, filepath.Join("..", "..", "..", "transactions", "cmd", "transactions"), "transactions")
	migrerTransactions(t, binaire, transactions)
	canal := demarrerRabbitMQ(t)
	public, interne, journal := lancerPlaceDeMarcheSurKeycloak(t, canal, marche.execution, kc, duree, plafond)
	lancerControle(t, construireControle(t), canal)
	relais := demarrerRelaisEspion(t, interne)
	b := &bancReservation{
		environnement: &environnement{adresse: public, client: &http.Client{Timeout: 10 * time.Second}, avecPhotos: true},
		kc:            kc,
		marche:        marche,
		transactions:  transactions,
		adresseMaitre: adresseMaitre,
		interne:       interne,
		public:        public,
		relais:        relais,
		journal:       journal,
	}
	b.tx = lancerTransactions(t, binaire, transactions, kc, relais.url)
	return b
}

func (b *bancReservation) consigner(interne bool, ech echange) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if interne {
		b.echangesInt = append(b.echangesInt, ech)
		return
	}
	b.echangesTx = append(b.echangesTx, ech)
}

func (b *bancReservation) appeler(t *testing.T, base, methode, chemin string, corps []byte, entetes http.Header) reponse {
	t.Helper()
	requete, err := http.NewRequestWithContext(t.Context(), methode, base+chemin, bytes.NewReader(corps))
	if err != nil {
		t.Fatalf("requête %s %s : %v", methode, chemin, err)
	}
	for nom, valeurs := range entetes {
		for _, v := range valeurs {
			requete.Header.Add(nom, v)
		}
	}
	if corps != nil && requete.Header.Get("Content-Type") == "" {
		requete.Header.Set("Content-Type", typeJSON)
	}
	resultat := b.executer(t, requete)
	requete.Body = io.NopCloser(bytes.NewReader(corps))
	b.consigner(base == b.interne, echange{requete: requete, reponse: resultat})
	return resultat
}

func avecJeton(jeton string) http.Header {
	entetes := http.Header{}
	if jeton != "" {
		entetes.Set("Authorization", bearer(jeton))
	}
	return entetes
}

func (b *bancReservation) commander(t *testing.T, jeton, corps string, entetes ...http.Header) reponse {
	t.Helper()
	tous := avecJeton(jeton)
	for _, supplementaires := range entetes {
		for nom, valeurs := range supplementaires {
			tous[nom] = append(tous[nom], valeurs...)
		}
	}
	return b.appeler(t, "http://"+b.tx.api, http.MethodPost, cheminCommandes, []byte(corps), tous)
}

func corpsDeCommande(articleID string) string {
	return fmt.Sprintf(`{"articleId":%q}`, articleID)
}

func (b *bancReservation) commanderAvecSucces(t *testing.T, jeton, articleID string, statut int) commandeRecue {
	t.Helper()
	var commande commandeRecue
	b.commander(t, jeton, corpsDeCommande(articleID)).exiger(t, statut).lire(t, &commande)
	return commande
}

func (b *bancReservation) lireCommande(t *testing.T, jeton, id string) reponse {
	t.Helper()
	return b.appeler(t, "http://"+b.tx.api, http.MethodGet, cheminCommandes+"/"+id, nil, avecJeton(jeton))
}

func cheminDeReservation(articleID string) string {
	return "/interne/v1/articles/" + articleID + "/reservations"
}

func (b *bancReservation) reserverEnDirect(t *testing.T, articleID string, entetes http.Header) reponse {
	t.Helper()
	return b.appeler(t, b.interne, http.MethodPost, cheminDeReservation(articleID), nil, entetes)
}

func entetesDeService(jetonService, jetonAcheteur string) http.Header {
	entetes := avecJeton(jetonService)
	if jetonAcheteur != "" {
		entetes.Set(enteteJetonAcheteur, jetonAcheteur)
	}
	return entetes
}

func (b *bancReservation) publier(t *testing.T, vendeur compteKeycloak, titre string, prix int64, frais *int64) articleRecu {
	t.Helper()
	jeton := bearer(b.kc.seConnecter(t, vendeur))
	corps := fmt.Sprintf(`{"titre":%q,"description":"Belle pièce\nen bon état","categorieId":%q,"prix":{"montantCentimes":%d,"devise":"EUR"}}`, titre, categorieDeLaReservaton, prix)
	if frais != nil {
		corps = corpsComplet(titre, categorieDeLaReservaton, prix, *frais)
	}
	soumis := b.soumettreAvecSucces(t, jeton, b.creer(t, jeton, corps))
	var publie articleRecu
	attendre(t, delaiDePublicationResa, "publication de "+titre, func() bool {
		publie = b.relire(t, jeton, soumis.ID)
		return publie.Etat == etatPublie
	})
	return publie
}

func frais(centimes int64) *int64 {
	return &centimes
}

func (b *bancReservation) articlePublie(t *testing.T, prix, fraisDePort int64) articleRecu {
	t.Helper()
	return b.publier(t, b.kc.creerVendeur(t), "Figurine "+texteAleatoire()[:6], prix, frais(fraisDePort))
}

func (b *bancReservation) fiche(t *testing.T, id string) entreeReservable {
	t.Helper()
	var fiche entreeReservable
	b.envoyer(t, http.MethodGet, cheminCatalogue+"/"+id, "").exiger(t, http.StatusOK).lire(t, &fiche)
	return fiche
}

func (b *bancReservation) entreeDuCatalogue(t *testing.T, id string) entreeReservable {
	t.Helper()
	for page := 1; page <= 20; page++ {
		var contenu pageReservable
		b.envoyer(t, http.MethodGet, fmt.Sprintf("%s?page=%d&taille=50", cheminCatalogue, page), "").exiger(t, http.StatusOK).lire(t, &contenu)
		for _, e := range contenu.Elements {
			if e.ID == id {
				return e
			}
		}
		if len(contenu.Elements) == 0 {
			break
		}
	}
	t.Fatalf("article %s absent du catalogue", id)
	return entreeReservable{}
}

type etatDeReservation struct {
	reservationID *string
	jusquA        *time.Time
}

func (b *bancReservation) reservationEnBase(t *testing.T, articleID string) etatDeReservation {
	t.Helper()
	var etat etatDeReservation
	err := b.marche.maitre.QueryRow(t.Context(), `SELECT reservation_id::text, reserve_jusqu_a FROM article WHERE id = $1`, articleID).Scan(&etat.reservationID, &etat.jusquA)
	if err != nil {
		t.Fatalf("réservation de l'article %s : %v", articleID, err)
	}
	return etat
}

func ligneJSON(t *testing.T, conn *pgx.Conn, requete string, arguments ...any) map[string]any {
	t.Helper()
	var brut []byte
	if err := conn.QueryRow(t.Context(), requete, arguments...).Scan(&brut); err != nil {
		t.Fatalf("lecture de %q : %v", requete, err)
	}
	var ligne map[string]any
	if err := json.Unmarshal(brut, &ligne); err != nil {
		t.Fatalf("ligne illisible : %v", err)
	}
	return ligne
}

func (b *bancReservation) commandeEnBase(t *testing.T, id string) map[string]any {
	t.Helper()
	return ligneJSON(t, b.transactions.maitre, `SELECT to_jsonb(c) FROM commande c WHERE id = $1`, id)
}

func (b *bancReservation) compterCommandes(t *testing.T, filtre string, arguments ...any) int {
	t.Helper()
	var n int
	if err := b.transactions.maitre.QueryRow(t.Context(), "SELECT count(*) FROM commande "+filtre, arguments...).Scan(&n); err != nil {
		t.Fatalf("comptage des commandes : %v", err)
	}
	return n
}

func (b *bancReservation) compterReservations(t *testing.T, articleID string) int {
	t.Helper()
	var n int
	if err := b.marche.maitre.QueryRow(t.Context(), `SELECT count(*) FROM reservation r WHERE to_jsonb(r)::text LIKE '%' || $1 || '%'`, articleID).Scan(&n); err != nil {
		t.Fatalf("comptage des réservations : %v", err)
	}
	return n
}

func instantRFC3339(t *testing.T, valeur string) time.Time {
	t.Helper()
	instant, err := time.Parse(time.RFC3339Nano, valeur)
	if err != nil {
		t.Fatalf("date %q illisible : %v", valeur, err)
	}
	return instant
}
