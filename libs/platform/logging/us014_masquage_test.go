package logging_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"collector/libs/platform/logging"
)

const (
	jwtDeVendeur      = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImsxIn0" + "." + "eyJzdWIiOiJ2ZW5kZXVyLXBpZWdlIiwiYXVkIjoibWFyY2hlIn0" + "." + "U2lnbmF0dXJlUGllZ2VlVmVuZGV1cjAxMjM0NTY3ODk"
	jwtUtilisateur    = "eyJ0eXAiOiJKV1QiLCJhbGciOiJFUzI1NiJ9" + "." + "eyJzdWIiOiJhY2hldGV1ci1waWVnZSJ9" + "." + "QWNoZXRldXJQaWVnZVNpZ25hdHVyZTk4NzY1NDMyMQ"
	jetonOpaque       = "opq-US014-4f2b8e1d9c7a"
	piegeDeBase       = "MdpBaseUS014-x9Q"
	piegeDuCanal      = "MdpCanalUS014-k7Z"
	identifiantBasic  = "vendeur-us014:MdpBasicUS014-p3W"
	signatureAWS      = "f3a1c9e7b5d3a1f9e7c5b3a19f7e5d3c1b9a7f5e3d1c9b7a5f3e1d9c7b5a3f1e"
	jetonDeSessionAWS = "IQoJb3JpZ2luX2VjUS014PiegeSessionToken0123456789abcdef"
	secretClient      = "SecretClientUS014-r8T"
	cookieDeSession   = "SessionUS014-c5V"
)

var cleAccesAWS = strings.ToUpper("akia" + "us014piege7example")

var valeursPiegees = []string{jwtDeVendeur, jwtUtilisateur, jetonOpaque, piegeDeBase, piegeDuCanal, base64.StdEncoding.EncodeToString([]byte(identifiantBasic)), "MdpBasicUS014-p3W", cleAccesAWS, signatureAWS, jetonDeSessionAWS, secretClient, cookieDeSession}

func urlPresigneeS3() string {
	return "https://collector-photos.s3.eu-west-3.amazonaws.com/articles/0b7e/photo-1.jpg" +
		"?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
		"&X-Amz-Credential=" + cleAccesAWS + "%2F20261120%2Feu-west-3%2Fs3%2Faws4_request" +
		"&X-Amz-Date=20261120T101502Z&X-Amz-Expires=900" +
		"&X-Amz-Security-Token=" + jetonDeSessionAWS +
		"&X-Amz-SignedHeaders=host" +
		"&X-Amz-Signature=" + signatureAWS
}

func urlDeBase() string {
	return "postgres://marche_execution:" + piegeDeBase + "@postgres.collector.svc:5432/marche?sslmode=require"
}

func urlDuCanal() string {
	return "amqps://marche:" + piegeDuCanal + "@rabbitmq.collector.svc:5671/collector"
}

func exigerAucuneValeurPiegee(t *testing.T, sortie string) {
	t.Helper()
	for _, piege := range valeursPiegees {
		if strings.Contains(sortie, piege) {
			t.Errorf("valeur piégée %q présente dans le journal : %s", piege, sortie)
		}
	}
}

func journaliserPuisExigerMasque(t *testing.T, journaliser func(*slog.Logger)) {
	t.Helper()
	logger, sortie := journalJSON(t, serviceDeTest)

	journaliser(logger)

	lignes := lignesDe(t, sortie)
	if len(lignes) == 0 {
		t.Fatalf("aucune ligne journalisée : le masquage ne doit pas supprimer l'événement")
	}
	exigerAucuneValeurPiegee(t, sortie.String())
}

func TestUS014AC5ValeursPiegeesMasqueesALaSource(t *testing.T) {
	presignee, err := url.Parse(urlPresigneeS3())
	if err != nil {
		t.Fatalf("URL présignée : %v", err)
	}
	base, err := url.Parse(urlDeBase())
	if err != nil {
		t.Fatalf("URL de base : %v", err)
	}
	cas := map[string]func(*slog.Logger){
		"en-tête Authorization Bearer JWT":    func(l *slog.Logger) { l.Info("requête", "Authorization", "Bearer "+jwtDeVendeur) },
		"en-tête authorization Bearer opaque": func(l *slog.Logger) { l.Info("requête", "authorization", "Bearer "+jetonOpaque) },
		"en-tête Authorization Basic": func(l *slog.Logger) {
			l.Info("requête", "Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(identifiantBasic)))
		},
		"en-tête X-Collector-User-Token":   func(l *slog.Logger) { l.Info("relais", "X-Collector-User-Token", jwtUtilisateur) },
		"clé x_collector_user_token":       func(l *slog.Logger) { l.Info("relais", "x_collector_user_token", jwtUtilisateur) },
		"clé jeton_utilisateur":            func(l *slog.Logger) { l.Info("relais", "jeton_utilisateur", jetonOpaque) },
		"clé access_token":                 func(l *slog.Logger) { l.Info("relais", "access_token", jetonOpaque) },
		"clé Cookie":                       func(l *slog.Logger) { l.Info("requête", "Cookie", "session="+cookieDeSession) },
		"clé client_secret":                func(l *slog.Logger) { l.Info("oidc", "client_secret", secretClient) },
		"clé CANAL_MOT_DE_PASSE":           func(l *slog.Logger) { l.Info("configuration", "CANAL_MOT_DE_PASSE", piegeDuCanal) },
		"clé BASE_DE_DONNEES_URL":          func(l *slog.Logger) { l.Info("configuration", "BASE_DE_DONNEES_URL", urlDeBase()) },
		"clé password":                     func(l *slog.Logger) { l.Info("connexion", "password", piegeDeBase) },
		"clé X-Amz-Signature":              func(l *slog.Logger) { l.Info("s3", "X-Amz-Signature", signatureAWS) },
		"JWT nu dans le message":           func(l *slog.Logger) { l.Info("jeton reçu " + jwtDeVendeur + " refusé") },
		"Bearer dans le message":           func(l *slog.Logger) { l.Info("appel avec Bearer " + jetonOpaque) },
		"JWT nu dans une valeur libre":     func(l *slog.Logger) { l.Info("refus", "detail", "jeton "+jwtUtilisateur+" expiré") },
		"URL présignée S3 en texte":        func(l *slog.Logger) { l.Info("photo", "url", urlPresigneeS3()) },
		"URL présignée S3 dans le message": func(l *slog.Logger) { l.Info("téléversement vers " + urlPresigneeS3()) },
		"URL présignée S3 en *url.URL":     func(l *slog.Logger) { l.Info("photo", "url", presignee) },
		"URL présignée S3 dans une *url.Error": func(l *slog.Logger) {
			l.Error("échec", "err", &url.Error{Op: "Put", URL: urlPresigneeS3(), Err: errors.New("connexion refusée")})
		},
		"URL présignée S3 dans une erreur enrobée": func(l *slog.Logger) {
			l.Error("échec", "err", fmt.Errorf("téléversement : %w", &url.Error{Op: "Put", URL: urlPresigneeS3(), Err: errors.New("délai")}))
		},
		"postgres://u:mdp@ en valeur libre": func(l *slog.Logger) { l.Info("connexion", "adresse", urlDeBase()) },
		"postgres://u:mdp@ en *url.URL":     func(l *slog.Logger) { l.Info("connexion", "adresse", base) },
		"postgres://u:mdp@ dans une erreur": func(l *slog.Logger) {
			l.Error("échec", "err", fmt.Errorf("ouverture de %s : connexion refusée", urlDeBase()))
		},
		"amqps://u:mdp@ dans le message": func(l *slog.Logger) { l.Error("canal injoignable " + urlDuCanal()) },
		"DSN clé-valeur password=": func(l *slog.Logger) {
			l.Error("échec", "detail", "host=pg user=marche password="+piegeDeBase+" dbname=marche")
		},
		"http.Header complet":   func(l *slog.Logger) { l.Info("requête", "entetes", enTetesPieges()) },
		"url.Values présignées": func(l *slog.Logger) { l.Info("photo", "parametres", presignee.Query()) },
		"url.Values token=": func(l *slog.Logger) {
			l.Info("rappel", "parametres", url.Values{"token": {jetonOpaque}, "access_token": {jwtDeVendeur}})
		},
		"groupe imbriqué": func(l *slog.Logger) {
			l.Info("requête", slog.Group("http", slog.Group("entetes", slog.String("Authorization", "Bearer "+jwtDeVendeur))))
		},
		"attribut posé par With":    func(l *slog.Logger) { l.With("authorization", "Bearer "+jwtDeVendeur).Info("requête") },
		"groupe posé par WithGroup": func(l *slog.Logger) { l.WithGroup("requete").Info("appel", "X-Collector-User-Token", jwtUtilisateur) },
		"niveau Debug":              func(l *slog.Logger) { l.Debug("détail", "jeton", jetonOpaque, "url", urlPresigneeS3()) },
		"LogAttrs avec contexte": func(l *slog.Logger) {
			l.LogAttrs(context.Background(), slog.LevelWarn, "avertissement", slog.String("Authorization", "Bearer "+jwtDeVendeur))
		},
		"audit avec valeur piégée": func(l *slog.Logger) {
			logging.Audit(context.Background(), l, "décision", "annonces.decision_admin", slog.String("detail", "Bearer "+jwtDeVendeur))
		},
		"valeur Stringer non URL portant un JWT": func(l *slog.Logger) { l.Info("jeton", "valeur", porteurDeJeton{}) },
	}
	for nom, journaliser := range cas {
		t.Run(nom, func(t *testing.T) { journaliserPuisExigerMasque(t, journaliser) })
	}
}

type porteurDeJeton struct{}

func (porteurDeJeton) String() string { return "porteur " + jwtDeVendeur }

func enTetesPieges() http.Header {
	return http.Header{
		"Authorization":          {"Bearer " + jwtDeVendeur},
		"X-Collector-User-Token": {jwtUtilisateur},
		"Cookie":                 {"session=" + cookieDeSession},
		"Proxy-Authorization":    {"Basic " + base64.StdEncoding.EncodeToString([]byte(identifiantBasic))},
		"Accept":                 {"application/json"},
	}
}

func TestUS014AC5RequeteHTTPPiegeeAucuneValeurDansLeJournal(t *testing.T) {
	logger, sortie := journalJSON(t, serviceDeTest)
	traiteur := logging.Middleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.InfoContext(r.Context(), messageDuTraiteur, "entetes", r.Header, "url", r.URL, "parametres", r.URL.Query())
		w.WriteHeader(http.StatusOK)
	}))
	presignee, err := url.Parse(urlPresigneeS3())
	if err != nil {
		t.Fatalf("URL présignée : %v", err)
	}
	requete := httptest.NewRequest(http.MethodGet, "/marche/v1/articles/0b7e/photo?"+presignee.RawQuery+"&token="+jetonOpaque, nil)
	for nom, valeurs := range enTetesPieges() {
		requete.Header[nom] = valeurs
	}

	traiteur.ServeHTTP(httptest.NewRecorder(), requete)

	if len(lignesDe(t, sortie)) == 0 {
		t.Fatal("aucune ligne journalisée")
	}
	exigerAucuneValeurPiegee(t, sortie.String())
}

func TestUS014AC5MasquageGardeLeJournalExploitable(t *testing.T) {
	logger, sortie := journalJSON(t, serviceDeTest)
	ctx := logging.ContinueTrace(logging.WithReceivedCorrelationID(context.Background(), correlationRecue), traceparentRecu)

	logger.ErrorContext(ctx, "téléversement en échec", "article_id", "0b7e", "err", &url.Error{Op: "Put", URL: urlPresigneeS3(), Err: errors.New("délai dépassé")})

	ligne := ligneDuMessage(t, sortie, "téléversement en échec")
	exigerChampsDeCorrelation(t, ligne, serviceDeTest, correlationRecue, traceRecue)
	if texte(ligne, "article_id") != "0b7e" || !strings.Contains(texte(ligne, "err"), "délai dépassé") {
		t.Errorf("information utile perdue par le masquage : %v", ligne)
	}
	exigerAucuneValeurPiegee(t, sortie.String())
}

func TestUS014AC5BearerEnMinusculesObservation(t *testing.T) {
	logger, sortie := journalJSON(t, serviceDeTest)

	logger.Info("appel", "detail", "en-tête reçu : bearer "+jetonOpaque)

	t.Logf("bearer en minuscules dans une valeur libre, jeton présent : %v", strings.Contains(sortie.String(), jetonOpaque))
}
