package coordonnees_test

import (
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"collector/services/controle/internal/coordonnees"
)

const (
	longueurMaxTitre       = 200
	longueurMaxDescription = 5000
	budgetParAnnonce       = 50 * time.Millisecond
	mesuresParTexte        = 7
)

type cas struct {
	nom   string
	texte string
}

var emailsEnClair = []cas{
	{"email simple", "jean.dupont@example.com"},
	{"email en majuscules", "JEAN.DUPONT@EXAMPLE.COM"},
	{"email dans une phrase", "Écrivez-moi à jean.dupont@example.com pour en discuter"},
	{"email avec plus et chiffres", "collec+tintin75@example.org"},
	{"email sous-domaine", "vente@mail.example.fr"},
}

var emailsMasques = []cas{
	{"arobase et point en toutes lettres", "jean point dupont arobase example point com"},
	{"arobase avec majuscules", "Jean Point Dupont AROBASE Example Point Com"},
	{"arobase accentuée", "jean.dupont arobasé example.com"},
	{"at entre parenthèses", "jean.dupont (at) example (dot) com"},
	{"at entre crochets", "jean.dupont [at] example [dot] com"},
	{"arobase entourée d'espaces", "jean.dupont @ example . com"},
	{"arobase et point com séparés", "jean.dupont arobase example . com"},
	{"point com en toutes lettres", "jean.dupont@example point com"},
	{"pleine chasse", "ｊｅａｎ．ｄｕｐｏｎｔ＠ｅｘａｍｐｌｅ．ｃｏｍ"},
	{"espace insécable invisible", "jean.dupont@exam\u200bple.com"},
	{"caractères invisibles multiples", "jean\u200d.dupont\u2060@\ufeffexample\u00ad.com"},
	{"arobase pleine chasse seule", "jean.dupont＠example.com"},
}

var liensEnClair = []cas{
	{"lien https", "Voir https://example.com/annonce/42"},
	{"lien http", "http://example.org"},
	{"lien www", "photos sur www.example.com"},
	{"domaine nu", "toutes les photos sur example.com"},
	{"domaine nu en fr", "boutique-collector.fr"},
	{"domaine masqué point com", "rendez-vous sur example point com"},
	{"lien en pleine chasse", "ｗｗｗ．ｅｘａｍｐｌｅ．ｃｏｍ"},
}

var pseudosEtReseaux = []cas{
	{"pseudo arobase", "suivez @collector_tintin75"},
	{"instagram", "plus de photos sur Instagram"},
	{"whatsapp", "contactez-moi sur WhatsApp"},
	{"facebook", "retrouvez-moi sur facebook"},
	{"snapchat", "mon snapchat : tintin75"},
	{"telegram", "dispo sur Telegram"},
}

var telephonesEnClair = []cas{
	{"fixe collé", "0199001234"},
	{"fixe espacé par paires", "01 99 00 12 34"},
	{"fixe avec points", "01.99.00.12.34"},
	{"fixe avec tirets", "01-99-00-12-34"},
	{"fixe séparateurs mélangés", "01 99.00-12 34"},
	{"mobile espacé", "appelez le 06 39 98 12 34 après 18h"},
	{"mobile 07", "07 00 00 12 34"},
	{"indicatif plus 33", "+33 1 99 00 12 34"},
	{"indicatif plus 33 collé", "+33199001234"},
	{"indicatif plus 33 avec zéro entre parenthèses", "+33 (0)1 99 00 12 34"},
	{"indicatif 0033", "0033 1 99 00 12 34"},
	{"indicatif 0033 collé", "0033199001234"},
	{"numéro en fin de phrase", "Mon numéro : 0639981234."},
	{"numéro entre parenthèses", "(01) 99 00 12 34"},
	{"numéro avec barres obliques", "06/39/98/12/34"},
	{"numéro avec barres obliques espacées", "06 / 39 / 98 / 12 / 34"},
	{"numéro avec un groupe ressemblant à une année", "06 2015 3456"},
}

var telephonesMasques = []cas{
	{"chiffres séparés par des espaces", "0 1 9 9 0 0 1 2 3 4"},
	{"chiffres séparés par plusieurs espaces", "06  39  98  12  34"},
	{"chiffres par groupes irréguliers", "063 998 123 4"},
	{"zéro six en lettres puis chiffres", "zéro six 39 98 12 34"},
	{"tout en lettres par nombres", "zéro six trente-neuf quatre-vingt-dix-huit douze trente-quatre"},
	{"tout en lettres sans tirets", "zero un quatre vingt dix neuf zero zero douze trente quatre"},
	{"tout en lettres chiffre par chiffre", "zéro six trois neuf neuf huit un deux trois quatre"},
	{"lettres en majuscules", "ZÉRO SIX TRENTE-NEUF QUATRE-VINGT-DIX-HUIT DOUZE TRENTE-QUATRE"},
	{"chiffres pleine chasse", "０６ ３９ ９８ １２ ３４"},
	{"chiffres avec caractères invisibles", "06\u200b39\u200b98\u200b12\u200b34"},
	{"plus 33 en lettres", "plus trente-trois six trente-neuf quatre-vingt-dix-huit douze trente-quatre"},
}

var textesSansCoordonnees = []cas{
	{"prix en euros", "Prix : 45 €, frais de port 6,50 €"},
	{"prix avec milliers", "Vendu 1 250,00 € ferme"},
	{"prix élevé espacé", "Estimation 12 500 € à 15 000 €"},
	{"dimensions avec x", "Dimensions 30 x 40 cm"},
	{"dimensions collées", "Format 30x40cm"},
	{"dimensions à trois côtés", "Boîte 21 x 29,7 x 3 cm"},
	{"poids", "Poids 1.250 kg"},
	{"année seule", "Édition de 1987"},
	{"plage d'années", "Production de 1960 à 1975"},
	{"suite d'années", "Lot millésimes 1999 2000 2001 2002 2003"},
	{"date avec barres", "Acheté le 12/05/1987"},
	{"date avec points", "Acheté le 12.05.1987"},
	{"date avec tirets", "Acheté le 12-05-1987"},
	{"date en lettres", "Acheté le douze mai mille neuf cent quatre-vingt-sept"},
	{"plage de dates", "Exposée du 01.01.1990 au 31.12.1999"},
	{"date et heure", "Mis en vente le 01/02/2003 à 14:30"},
	{"date et heure seules", "01/02/2003 à 14:30"},
	{"tirage puis date", "Edition 01/50, 12/05/2003"},
	{"date au format iso", "Sortie le 1987-05-12"},
	{"date pointée seule", "12.05.1987"},
	{"point suivi d'une abréviation de langue", "English. FR : description"},
	{"référence isbn 13", "ISBN 978-2-203-00101-8"},
	{"référence isbn 10", "ISBN 2-203-00101-6"},
	{"édition originale et tirage", "Édition originale 1954, tirage de tête n° 125/300"},
	{"dépôt légal", "Casterman, dépôt légal 4e trimestre 1946, B12"},
	{"référence montre", "Omega Speedmaster réf. 145.022-69"},
	{"numéro de série chiffres", "Numéro de série : 45781236"},
	{"numéro de série alphanumérique", "SN AB-12345-CD"},
	{"numéro de série tiré", "Exemplaire 0042/1000"},
	{"tome", "Tintin tome 12, état neuf"},
	{"tome à deux chiffres zéro", "Astérix tome 06"},
	{"point collé après une phrase", "Bon état.De plus la boîte est d'origine"},
	{"point collé avant comme", "Belle pièce.Comme neuve"},
	{"mot point dans une phrase", "Le point fort de cette pièce est sa patine"},
	{"vinyle 33 tours", "Vinyle 33 tours de 1975"},
	{"pièce de monnaie", "Pièce de 5 francs 1960"},
	{"nombres en lettres courants", "Lot de deux figurines et trois cartes"},
	{"texte vide", ""},
	{"texte en espaces", "   "},
}

func exigerDetection(t *testing.T, liste []cas, attendu bool) {
	t.Helper()
	for _, c := range liste {
		t.Run(c.nom, func(t *testing.T) {
			if recu := coordonnees.Detected(c.texte); recu != attendu {
				t.Errorf("Detected(%q) = %v, attendu %v", c.texte, recu, attendu)
			}
		})
	}
}

func TestUS018AC1EmailEnClairDetecte(t *testing.T) {
	exigerDetection(t, emailsEnClair, true)
}

func TestUS018AC1LienExterneEnClairDetecte(t *testing.T) {
	exigerDetection(t, liensEnClair, true)
}

func TestUS018AC1TelephoneFrancaisEnClairDetecte(t *testing.T) {
	exigerDetection(t, telephonesEnClair, true)
}

func TestUS018AC1PseudoEtReseauSocialDetectes(t *testing.T) {
	exigerDetection(t, pseudosEtReseaux, true)
}

func TestUS018AC2EmailMasqueDetecte(t *testing.T) {
	exigerDetection(t, emailsMasques, true)
}

func TestUS018AC2TelephoneMasqueDetecte(t *testing.T) {
	exigerDetection(t, telephonesMasques, true)
}

func TestUS018AC3TexteSansCoordonneesNonDetecte(t *testing.T) {
	exigerDetection(t, textesSansCoordonnees, false)
}

func repeter(motif string, longueur int) string {
	var texte strings.Builder
	for utf8.RuneCountInString(texte.String()) < longueur {
		texte.WriteString(motif)
	}
	return string([]rune(texte.String())[:longueur])
}

func textesPiegesDeLongueur(longueur int) map[string]string {
	return map[string]string{
		"chiffres espacés":           repeter("0 ", longueur),
		"chiffres collés":            repeter("0", longueur),
		"zéros en lettres":           repeter("zéro ", longueur),
		"nombres composés":           repeter("quatre-vingt-dix-", longueur),
		"arobases":                   repeter("a@", longueur),
		"arobase en lettres":         repeter("arobase point ", longueur),
		"points et lettres":          repeter("a.", longueur),
		"www répétés":                repeter("www.", longueur),
		"plus 33 répétés":            repeter("+33 ", longueur),
		"années répétées":            repeter("1999 ", longueur),
		"chiffres et séparateurs":    repeter("0.1-2 (3) ", longueur),
		"pleine chasse et invisible": repeter("０\u200b＠\u200b．", longueur),
		"texte réaliste":             repeter("Figurine Tintin en bon état, édition 1987, 30 x 40 cm. ", longueur),
	}
}

func meilleureDuree(texte string) time.Duration {
	durees := make([]time.Duration, mesuresParTexte)
	for i := range durees {
		debut := time.Now()
		coordonnees.Detected(texte)
		durees[i] = time.Since(debut)
	}
	return slices.Min(durees)
}

func TestUS018AC6TexteMaximalPiegeAnalyseEnMoinsDe50ms(t *testing.T) {
	titres := textesPiegesDeLongueur(longueurMaxTitre)
	for nom, description := range textesPiegesDeLongueur(longueurMaxDescription) {
		t.Run(nom, func(t *testing.T) {
			duree := meilleureDuree(titres[nom]) + meilleureDuree(description)
			t.Logf("annonce piégée %q : %s", nom, duree)
			if duree >= budgetParAnnonce {
				t.Errorf("détection en %s, budget %s", duree, budgetParAnnonce)
			}
		})
	}
}

func BenchmarkUS018AC6AnnonceMaximalePiegee(b *testing.B) {
	titres := textesPiegesDeLongueur(longueurMaxTitre)
	for nom, description := range textesPiegesDeLongueur(longueurMaxDescription) {
		b.Run(nom, func(b *testing.B) {
			for b.Loop() {
				coordonnees.Detected(titres[nom])
				coordonnees.Detected(description)
			}
		})
	}
}

func FuzzDetected(f *testing.F) {
	for _, liste := range [][]cas{emailsEnClair, emailsMasques, liensEnClair, pseudosEtReseaux, telephonesEnClair, telephonesMasques, textesSansCoordonnees} {
		for _, c := range liste {
			f.Add(c.texte)
		}
	}
	f.Fuzz(func(_ *testing.T, texte string) {
		coordonnees.Detected(texte)
	})
}
