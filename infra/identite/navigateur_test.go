package identite

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

const (
	redirectionDuSite      = "http://localhost:5173/rappel"
	cheminDAutorisation    = "/realms/" + Realm + "/protocol/openid-connect/auth"
	cheminDuJetonDuRealm   = "/realms/" + Realm + "/protocol/openid-connect/token"
	methodeDeDefi          = "S256"
	formulaireDeConnexion  = "kc-form-login"
	formulaireDeSaisieOTP  = "kc-otp-login-form"
	formulaireDeConfigOTP  = "kc-totp-settings-form"
	actionDeConfigOTP      = "CONFIGURE_TOTP"
	limiteDeRedirections   = 10
	typeFormulaireEncodeEn = "application/x-www-form-urlencoded"
)

var (
	baliseFormulaire = regexp.MustCompile(`(?is)<form\b[^>]*>`)
	baliseChamp      = regexp.MustCompile(`(?is)<input\b[^>]*>`)
	attributHTML     = regexp.MustCompile(`(?is)\b([a-z][a-z0-9_-]*)\s*=\s*"([^"]*)"`)
)

type navigateur struct {
	kc           *keycloakDeTest
	client       *http.Client
	verificateur string
	etat         string
}

type etape struct {
	statut  int
	page    string
	adresse *url.URL
	retour  *url.URL
}

func (e etape) code() string {
	if e.retour == nil {
		return ""
	}
	return e.retour.Query().Get("code")
}

func nouveauNavigateur(t *testing.T, kc *keycloakDeTest) *navigateur {
	t.Helper()
	pot, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("pot à cookies : %v", err)
	}
	return &navigateur{
		kc: kc,
		client: &http.Client{
			Jar: pot,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		verificateur: valeurAleatoire() + valeurAleatoire(),
		etat:         valeurAleatoire(),
	}
}

func (n *navigateur) defi() string {
	empreinte := sha256.Sum256([]byte(n.verificateur))
	return base64.RawURLEncoding.EncodeToString(empreinte[:])
}

func (n *navigateur) parametresDAutorisation() url.Values {
	return url.Values{
		"client_id":             {ClientDuSite},
		"response_type":         {"code"},
		"redirect_uri":          {redirectionDuSite},
		"scope":                 {"openid profile"},
		"state":                 {n.etat},
		"code_challenge":        {n.defi()},
		"code_challenge_method": {methodeDeDefi},
	}
}

func (n *navigateur) ouvrir(t *testing.T, parametres url.Values) etape {
	t.Helper()
	return n.aller(t, http.MethodGet, n.kc.base+cheminDAutorisation+"?"+parametres.Encode(), nil)
}

func (n *navigateur) seConnecter(t *testing.T, compte utilisateurDeTest) etape {
	t.Helper()
	page := n.ouvrir(t, n.parametresDAutorisation())
	return n.soumettre(t, page, formulaireDeConnexion, url.Values{
		"username":     {compte.nom},
		"password":     {compte.secret},
		"credentialId": {""},
	})
}

func (n *navigateur) soumettre(t *testing.T, precedente etape, idDuFormulaire string, champs url.Values) etape {
	t.Helper()
	if precedente.statut != http.StatusOK {
		t.Fatalf("formulaire %s attendu, statut %d", idDuFormulaire, precedente.statut)
	}
	action := actionDuFormulaire(t, precedente.page, idDuFormulaire)
	cible, err := precedente.adresse.Parse(action)
	if err != nil {
		t.Fatalf("action du formulaire %s : %v", idDuFormulaire, err)
	}
	return n.aller(t, http.MethodPost, cible.String(), champs)
}

func (n *navigateur) aller(t *testing.T, methode, adresse string, champs url.Values) etape {
	t.Helper()
	for range limiteDeRedirections {
		cible, err := url.Parse(adresse)
		if err != nil {
			t.Fatalf("adresse %q : %v", adresse, err)
		}
		statut, page, suivante := n.requete(t, methode, cible, champs)
		if suivante == nil {
			return etape{statut: statut, page: page, adresse: cible}
		}
		if strings.HasPrefix(suivante.String(), redirectionDuSite) {
			return etape{statut: statut, adresse: cible, retour: suivante}
		}
		methode, adresse, champs = http.MethodGet, suivante.String(), nil
	}
	t.Fatalf("plus de %d redirections", limiteDeRedirections)
	return etape{}
}

func (n *navigateur) requete(t *testing.T, methode string, cible *url.URL, champs url.Values) (int, string, *url.URL) {
	t.Helper()
	var corps io.Reader
	if champs != nil {
		corps = strings.NewReader(champs.Encode())
	}
	requete, err := http.NewRequestWithContext(n.kc.ctx, methode, cible.String(), corps)
	if err != nil {
		t.Fatalf("%s %s : %v", methode, cible.Path, err)
	}
	if champs != nil {
		requete.Header.Set("Content-Type", typeFormulaireEncodeEn)
	}
	reponse, err := n.client.Do(requete)
	if err != nil {
		t.Fatalf("%s %s : %v", methode, cible.Path, err)
	}
	defer func() { _ = reponse.Body.Close() }()
	contenu, err := io.ReadAll(reponse.Body)
	if err != nil {
		t.Fatalf("%s %s : lecture : %v", methode, cible.Path, err)
	}
	emplacement := reponse.Header.Get("Location")
	if reponse.StatusCode < 300 || reponse.StatusCode > 399 || emplacement == "" {
		return reponse.StatusCode, string(contenu), nil
	}
	suivante, err := cible.Parse(emplacement)
	if err != nil {
		t.Fatalf("redirection illisible depuis %s : %v", cible.Path, err)
	}
	return reponse.StatusCode, string(contenu), suivante
}

func (n *navigateur) echangerLeCode(t *testing.T, code string) map[string]any {
	t.Helper()
	formulaire := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {ClientDuSite},
		"redirect_uri":  {redirectionDuSite},
		"code":          {code},
		"code_verifier": {n.verificateur},
	}
	var reponse struct {
		AccessToken string `json:"access_token"`
	}
	n.kc.envoyer(t, http.MethodPost, cheminDuJetonDuRealm, typeFormulaireEncodeEn, strings.NewReader(formulaire.Encode()), "", http.StatusOK, &reponse)
	return revendications(t, reponse.AccessToken)
}

func revendications(t *testing.T, jeton string) map[string]any {
	t.Helper()
	parties := strings.Split(jeton, ".")
	if len(parties) != 3 {
		t.Fatalf("jeton d'accès mal formé")
	}
	brut, err := base64.RawURLEncoding.DecodeString(parties[1])
	if err != nil {
		t.Fatalf("charge du jeton : %v", err)
	}
	var charge map[string]any
	if err := json.Unmarshal(brut, &charge); err != nil {
		t.Fatalf("charge du jeton : %v", err)
	}
	return charge
}

func attributs(balise string) map[string]string {
	valeurs := map[string]string{}
	for _, correspondance := range attributHTML.FindAllStringSubmatch(balise, -1) {
		valeurs[strings.ToLower(correspondance[1])] = html.UnescapeString(correspondance[2])
	}
	return valeurs
}

func baliseAvec(page string, motif *regexp.Regexp, nom, valeur string) (map[string]string, bool) {
	for _, balise := range motif.FindAllString(page, -1) {
		if a := attributs(balise); a[nom] == valeur {
			return a, true
		}
	}
	return nil, false
}

func contientLeFormulaire(page, idDuFormulaire string) bool {
	_, trouve := baliseAvec(page, baliseFormulaire, "id", idDuFormulaire)
	return trouve
}

func actionDuFormulaire(t *testing.T, page, idDuFormulaire string) string {
	t.Helper()
	formulaire, trouve := baliseAvec(page, baliseFormulaire, "id", idDuFormulaire)
	if !trouve || formulaire["action"] == "" {
		t.Fatalf("formulaire %s absent de la page", idDuFormulaire)
	}
	return formulaire["action"]
}

func valeurDuChamp(t *testing.T, page, nom string) string {
	t.Helper()
	champ, trouve := baliseAvec(page, baliseChamp, "name", nom)
	if !trouve {
		t.Fatalf("champ %s absent de la page", nom)
	}
	return champ["value"]
}
