package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"collector/libs/platform/httpserver"
)

const adresseAjouteeParLALB = "203.0.113.7"

func requeteDepuis(source string, lignesForwardedFor ...string) *http.Request {
	requete := httptest.NewRequest(http.MethodPost, "/marche/v1/vendeur/articles/a/soumission", nil)
	requete.RemoteAddr = source
	for _, ligne := range lignesForwardedFor {
		requete.Header.Add("X-Forwarded-For", ligne)
	}
	return requete
}

func exigerAdresse(t *testing.T, requete *http.Request, attendue string) {
	t.Helper()
	adresse, ok := httpserver.ClientAddress(requete)
	if !ok {
		t.Fatalf("adresse %s attendue, aucune adresse lue", attendue)
	}
	if adresse != netip.MustParseAddr(attendue) {
		t.Fatalf("adresse %s lue, %s attendue", adresse, attendue)
	}
}

func TestUS027AC2DerniereValeurAjouteeParLALB(t *testing.T) {
	cas := map[string][]string{
		"valeur unique":                  {adresseAjouteeParLALB},
		"valeur forgée puis ALB":         {"198.51.100.1, " + adresseAjouteeParLALB},
		"liste forgée puis ALB":          {"198.51.100.1, 198.51.100.2, 10.0.0.1, " + adresseAjouteeParLALB},
		"valeur forgée non IP puis ALB":  {"nimporte-quoi, " + adresseAjouteeParLALB},
		"espaces autour des valeurs":     {"  198.51.100.1 ,   " + adresseAjouteeParLALB + "  "},
		"sans espace après la virgule":   {"198.51.100.1," + adresseAjouteeParLALB},
		"deux lignes d'en-tête":          {"198.51.100.1", "198.51.100.2, " + adresseAjouteeParLALB},
		"première ligne avec l'adresse":  {adresseAjouteeParLALB + ", 198.51.100.1", "198.51.100.9, " + adresseAjouteeParLALB},
		"trois lignes, dernière seule":   {"198.51.100.1", "198.51.100.2", adresseAjouteeParLALB},
		"forgée avec l'adresse de l'ALB": {adresseAjouteeParLALB + ", 198.51.100.1, " + adresseAjouteeParLALB},
	}
	for nom, lignes := range cas {
		t.Run(nom, func(t *testing.T) {
			exigerAdresse(t, requeteDepuis("10.0.1.20:41000", lignes...), adresseAjouteeParLALB)
		})
	}
}

func TestUS027AC2DerniereLigneDifferenteDeLaPremiere(t *testing.T) {
	exigerAdresse(t, requeteDepuis("10.0.1.20:41000", "198.51.100.1, 198.51.100.2", "198.51.100.3"), "198.51.100.3")
}

func TestUS027AC2AdresseIPv6AjouteeParLALB(t *testing.T) {
	cas := map[string]string{
		"compacte":     "2001:db8::5",
		"développée":   "2001:0db8:0000:0000:0000:0000:0000:0005",
		"après forgée": "198.51.100.1, 2001:db8::5",
	}
	for nom, ligne := range cas {
		t.Run(nom, func(t *testing.T) {
			exigerAdresse(t, requeteDepuis("10.0.1.20:41000", ligne), "2001:db8::5")
		})
	}
}

func TestUS027AC2SansForwardedForAdresseSourceSansPort(t *testing.T) {
	cas := map[string]struct {
		source   string
		attendue string
	}{
		"IPv4 avec port": {"192.0.2.10:52000", "192.0.2.10"},
		"IPv6 avec port": {"[2001:db8::7]:443", "2001:db8::7"},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			exigerAdresse(t, requeteDepuis(c.source), c.attendue)
		})
	}
}

func TestUS027AC2MemeCleAvecOuSansForgeEnTete(t *testing.T) {
	sansForge, _ := httpserver.ClientAddress(requeteDepuis("10.0.1.20:41000", adresseAjouteeParLALB))
	for _, forge := range []string{"198.51.100.1", "198.51.100.1, 198.51.100.2", "::1", "127.0.0.1"} {
		avecForge, _ := httpserver.ClientAddress(requeteDepuis("10.0.1.20:41000", forge+", "+adresseAjouteeParLALB))
		if avecForge != sansForge {
			t.Fatalf("forge %q : adresse %s, %s sans forge", forge, avecForge, sansForge)
		}
	}
}

func TestUS027AC2AdresseSourceIllisibleSansAdresse(t *testing.T) {
	if adresse, ok := httpserver.ClientAddress(requeteDepuis("pas-une-adresse")); ok {
		t.Fatalf("aucune adresse attendue, %s lue", adresse)
	}
}
