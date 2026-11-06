package ratelimit_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"collector/libs/platform/ratelimit"
)

const (
	sourceALB      = "10.0.1.20:41000"
	adresseClient  = "203.0.113.7"
	autreClient    = "203.0.113.8"
	adresseDeTir   = "198.51.100.42"
	typeProbleme   = "application/problem+json"
	rafaleNominale = 10
)

type horloge struct {
	mu         sync.Mutex
	maintenant time.Time
}

func (h *horloge) lire() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.maintenant
}

func (h *horloge) avancer(duree time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.maintenant = h.maintenant.Add(duree)
}

type banc struct {
	horloge *horloge
	limite  http.Handler
	servies atomic.Int64
}

type probleme struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
}

func journalMuet() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func nominale() ratelimit.Policy {
	return ratelimit.Policy{Nominal: ratelimit.PerMinute(30, rafaleNominale), LoadTest: ratelimit.PerMinute(6000, 200)}
}

func avecTir(politique ratelimit.Policy, adresse string) ratelimit.Policy {
	politique.LoadTestAddress = netipAddr(adresse)
	return politique
}

func nouveauBanc(politique ratelimit.Policy) *banc {
	b := &banc{horloge: &horloge{maintenant: time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC)}}
	suite := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		b.servies.Add(1)
		w.WriteHeader(http.StatusAccepted)
	})
	b.limite = ratelimit.NewGuard(politique, journalMuet(), b.horloge.lire).Limit(suite)
	return b
}

func (b *banc) envoyer(lignesForwardedFor ...string) *httptest.ResponseRecorder {
	requete := httptest.NewRequest(http.MethodPost, "/marche/v1/vendeur/articles/a/soumission", nil)
	requete.RemoteAddr = sourceALB
	for _, ligne := range lignesForwardedFor {
		requete.Header.Add("X-Forwarded-For", ligne)
	}
	enregistreur := httptest.NewRecorder()
	b.limite.ServeHTTP(enregistreur, requete)
	return enregistreur
}

func (b *banc) depuis(adresse string) *httptest.ResponseRecorder {
	return b.envoyer(adresse)
}

func (b *banc) epuiser(t *testing.T, adresse string, rafale int) {
	t.Helper()
	for i := range rafale {
		if r := b.depuis(adresse); r.Code != http.StatusAccepted {
			t.Fatalf("requête %d de %s dans la rafale : %d reçu", i+1, adresse, r.Code)
		}
	}
}

func exigerRefus(t *testing.T, r *httptest.ResponseRecorder) int {
	t.Helper()
	if r.Code != http.StatusTooManyRequests {
		t.Fatalf("429 attendu, %d reçu", r.Code)
	}
	attente, err := strconv.Atoi(r.Header().Get("Retry-After"))
	if err != nil || attente < 1 {
		t.Fatalf("Retry-After entier ≥ 1 attendu, %q reçu", r.Header().Get("Retry-After"))
	}
	if !strings.HasPrefix(r.Header().Get("Content-Type"), typeProbleme) {
		t.Fatalf("Content-Type %q reçu, %q attendu", r.Header().Get("Content-Type"), typeProbleme)
	}
	var p probleme
	if err := json.Unmarshal(r.Body.Bytes(), &p); err != nil {
		t.Fatalf("Problem Details illisible : %v : %s", err, r.Body.Bytes())
	}
	if p.Status != http.StatusTooManyRequests || p.Title == "" || p.Type == "" {
		t.Fatalf("Problem Details incomplet : %s", r.Body.Bytes())
	}
	return attente
}

func exigerAccepte(t *testing.T, r *httptest.ResponseRecorder) {
	t.Helper()
	if r.Code != http.StatusAccepted {
		t.Fatalf("requête servie attendue, %d reçu : %s", r.Code, r.Body.Bytes())
	}
}

func TestUS027AC1RafaleServiePuis429AvecRetryAfter(t *testing.T) {
	b := nouveauBanc(nominale())
	b.epuiser(t, adresseClient, rafaleNominale)

	exigerRefus(t, b.depuis(adresseClient))

	if b.servies.Load() != rafaleNominale {
		t.Fatalf("%d requêtes transmises au service, %d attendues", b.servies.Load(), rafaleNominale)
	}
}

func TestUS027AC1RetryAfterEnSecondesArrondiesAuSuperieur(t *testing.T) {
	cas := map[string]struct {
		limite  ratelimit.Limit
		attente int
	}{
		"30 par minute : 2 s":          {ratelimit.PerMinute(30, 1), 2},
		"7 par minute : 8,57 s → 9":    {ratelimit.PerMinute(7, 1), 9},
		"1 par minute : 60 s":          {ratelimit.PerMinute(1, 1), 60},
		"6000 par minute : 0,01 s → 1": {ratelimit.PerMinute(6000, 1), 1},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			b := nouveauBanc(ratelimit.Policy{Nominal: c.limite, LoadTest: c.limite})
			b.epuiser(t, adresseClient, 1)

			if attente := exigerRefus(t, b.depuis(adresseClient)); attente != c.attente {
				t.Fatalf("Retry-After %d, %d attendu", attente, c.attente)
			}
		})
	}
}

func TestUS027AC1RetryAfterDecroitAvecLeTemps(t *testing.T) {
	b := nouveauBanc(ratelimit.Policy{Nominal: ratelimit.PerMinute(1, 1), LoadTest: ratelimit.PerMinute(1, 1)})
	b.epuiser(t, adresseClient, 1)

	b.horloge.avancer(45*time.Second + 500*time.Millisecond)

	if attente := exigerRefus(t, b.depuis(adresseClient)); attente != 15 {
		t.Fatalf("Retry-After %d après 45,5 s, 15 attendu", attente)
	}
}

func TestUS027AC1ServieDeNouveauApresRetryAfter(t *testing.T) {
	b := nouveauBanc(nominale())
	b.epuiser(t, adresseClient, rafaleNominale)
	attente := exigerRefus(t, b.depuis(adresseClient))

	b.horloge.avancer(time.Duration(attente) * time.Second)

	exigerAccepte(t, b.depuis(adresseClient))
	exigerRefus(t, b.depuis(adresseClient))
}

func TestUS027AC1DebitSoutenuAuRythmeConfigure(t *testing.T) {
	b := nouveauBanc(nominale())
	b.epuiser(t, adresseClient, rafaleNominale)

	for minute := range 3 {
		for i := range 30 {
			b.horloge.avancer(2 * time.Second)
			if r := b.depuis(adresseClient); r.Code != http.StatusAccepted {
				t.Fatalf("minute %d, requête %d au rythme de 30/min : %d reçu", minute+1, i+1, r.Code)
			}
		}
	}
	exigerRefus(t, b.depuis(adresseClient))
}

func TestUS027AC1RafaleBorneeApresLongueInactivite(t *testing.T) {
	b := nouveauBanc(nominale())
	b.horloge.avancer(24 * time.Hour)

	b.epuiser(t, adresseClient, rafaleNominale)

	exigerRefus(t, b.depuis(adresseClient))
}

func TestUS027AC1AdressesIndependantes(t *testing.T) {
	b := nouveauBanc(nominale())
	b.epuiser(t, adresseClient, rafaleNominale)
	exigerRefus(t, b.depuis(adresseClient))

	b.epuiser(t, autreClient, rafaleNominale)
}

func TestUS027AC1AdresseIPv6LimiteeParSonPrefixe64(t *testing.T) {
	b := nouveauBanc(nominale())
	for i := range rafaleNominale {
		exigerAccepte(t, b.depuis("2001:db8:1:2::"+strconv.FormatInt(int64(i+1), 16)))
	}

	exigerRefus(t, b.depuis("2001:db8:1:2:ffff:ffff:ffff:ffff"))
	exigerAccepte(t, b.depuis("2001:db8:1:3::1"))
}

func TestUS027AC4ForwardedForForgeResteLimite(t *testing.T) {
	forges := map[string]func(i int) []string{
		"valeur unique tournante": func(i int) []string {
			return []string{"192.0.2." + strconv.Itoa(i+1) + ", " + adresseClient}
		},
		"liste tournante": func(i int) []string {
			return []string{"192.0.2." + strconv.Itoa(i+1) + ", 10.0.0." + strconv.Itoa(i+1) + ", 127.0.0.1, " + adresseClient}
		},
		"ligne forgée séparée": func(i int) []string {
			return []string{"192.0.2." + strconv.Itoa(i+1), adresseClient}
		},
		"valeur illisible": func(i int) []string {
			return []string{"inconnu-" + strconv.Itoa(i) + ", " + adresseClient}
		},
		"adresse de l'autre client forgée": func(int) []string {
			return []string{autreClient + ", " + adresseClient}
		},
	}
	for nom, forge := range forges {
		t.Run(nom, func(t *testing.T) {
			b := nouveauBanc(nominale())
			for i := range rafaleNominale {
				exigerAccepte(t, b.envoyer(forge(i)...))
			}

			exigerRefus(t, b.envoyer(forge(rafaleNominale)...))
			exigerRefus(t, b.depuis(adresseClient))
			exigerAccepte(t, b.depuis(autreClient))
		})
	}
}

func TestUS027AC4ForgeApresEpuisementResteLimitee(t *testing.T) {
	b := nouveauBanc(nominale())
	b.epuiser(t, adresseClient, rafaleNominale)

	for _, forge := range []string{"192.0.2.1", "192.0.2.1, 192.0.2.2", "::1", ""} {
		exigerRefus(t, b.envoyer(forge+", "+adresseClient))
	}
}

func TestUS027AC3LimiteReleveePourLaSeuleAdresseDeTir(t *testing.T) {
	b := nouveauBanc(avecTir(nominale(), adresseDeTir))

	b.epuiser(t, adresseDeTir, 200)
	exigerRefus(t, b.depuis(adresseDeTir))

	b.epuiser(t, adresseClient, rafaleNominale)
	exigerRefus(t, b.depuis(adresseClient))
}

func TestUS027AC3AdresseDeTirForgeeNeReleveRien(t *testing.T) {
	forges := map[string][]string{
		"adresse de tir en tête":         {adresseDeTir + ", " + adresseClient},
		"adresse de tir en ligne à part": {adresseDeTir, adresseClient},
		"liste avec l'adresse de tir":    {"192.0.2.1, " + adresseDeTir + ", 192.0.2.2, " + adresseClient},
	}
	for nom, forge := range forges {
		t.Run(nom, func(t *testing.T) {
			b := nouveauBanc(avecTir(nominale(), adresseDeTir))
			for range rafaleNominale {
				exigerAccepte(t, b.envoyer(forge...))
			}

			exigerRefus(t, b.envoyer(forge...))
		})
	}
}

func TestUS027AC3SansAdresseDeTirAucuneLimiteRelevee(t *testing.T) {
	b := nouveauBanc(nominale())

	b.epuiser(t, adresseDeTir, rafaleNominale)

	exigerRefus(t, b.depuis(adresseDeTir))
}

func TestUS027AC5FinDuTirLaLimiteNominaleRedonne429(t *testing.T) {
	session := nouveauBanc(avecTir(nominale(), adresseDeTir))
	session.epuiser(t, adresseDeTir, 150)
	session.epuiser(t, adresseClient, rafaleNominale)
	exigerRefus(t, session.depuis(adresseClient))

	politique, err := ratelimit.LoadPolicy(variables(nil))
	if err != nil {
		t.Fatalf("politique nominale : %v", err)
	}
	apresTir := nouveauBanc(politique)

	apresTir.epuiser(t, adresseDeTir, rafaleNominale)
	exigerRefus(t, apresTir.depuis(adresseDeTir))
	apresTir.epuiser(t, adresseClient, rafaleNominale)
	exigerRefus(t, apresTir.depuis(adresseClient))
}

func TestUS027PurgeEtCapaciteLaLimiteContinueDeSAppliquer(t *testing.T) {
	b := nouveauBanc(nominale())
	b.epuiser(t, adresseClient, rafaleNominale)
	b.horloge.avancer(2 * time.Minute)

	for i := range 60_000 {
		b.depuis("10." + strconv.Itoa(i>>16&255) + "." + strconv.Itoa(i>>8&255) + "." + strconv.Itoa(i&255))
	}
	b.horloge.avancer(2 * time.Minute)

	b.epuiser(t, autreClient, rafaleNominale)
	exigerRefus(t, b.depuis(autreClient))
	b.epuiser(t, adresseClient, rafaleNominale)
	exigerRefus(t, b.depuis(adresseClient))
}

func TestUS027AC1RequetesConcurrentesBorneesALaRafale(t *testing.T) {
	b := nouveauBanc(nominale())
	var servies, refusees atomic.Int64
	var groupe sync.WaitGroup
	for range 64 {
		groupe.Go(func() {
			for range 4 {
				switch b.depuis(adresseClient).Code {
				case http.StatusAccepted:
					servies.Add(1)
				case http.StatusTooManyRequests:
					refusees.Add(1)
				}
			}
			b.depuis("192.0.2." + strconv.Itoa(int(servies.Load()%250)+1))
		})
	}
	groupe.Wait()

	if servies.Load() != rafaleNominale || refusees.Load() != 256-rafaleNominale {
		t.Fatalf("%d servies et %d refusées, %d et %d attendues", servies.Load(), refusees.Load(), rafaleNominale, 256-rafaleNominale)
	}
}
