package logging_test

import (
	"bytes"
	"context"
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
	jetonPiege       = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9" + "." + "eyJzdWIiOiJwaWVnZSJ9" + "." + "c2lnbmF0dXJlLXBpZWdlZQ"
	opaquePiege      = "opaque-piege-7f3a9c"
	piegeDeConnexion = "ValeurPiegeeDeConnexion42"
	chaineConnexion  = "postgres://marche_execution:" + piegeDeConnexion + "@base.collector.test:5432/marche?sslmode=verify-full"
	valeursInterdite = "valeur piégée"
	signaturePiegee  = "signature-presignee-piegee"
	identitePiegee   = "identite-presignee-piegee"
	sessionPiegee    = "session-presignee-piegee"
	urlPresignee     = "https://photos.collector.test/a.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=" + identitePiegee + "&X-Amz-Signature=" + signaturePiegee
	urlMinuscules    = "https://photos.collector.test/b.jpg?x-amz-credential=" + identitePiegee + "&x-amz-security-token=" + sessionPiegee + "&x-amz-signature=" + signaturePiegee
)

var pieges = []string{jetonPiege, opaquePiege, piegeDeConnexion, signaturePiegee, identitePiegee, sessionPiegee}

func journal(t *testing.T) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	sortie := &bytes.Buffer{}
	return logging.New("place-de-marche", sortie, slog.LevelDebug), sortie
}

func exigerSansPiege(t *testing.T, sortie *bytes.Buffer) {
	t.Helper()
	if sortie.Len() == 0 {
		t.Fatal("aucune ligne journalisée")
	}
	for _, p := range pieges {
		if strings.Contains(sortie.String(), p) {
			t.Fatalf("%s présente dans le journal : %s", valeursInterdite, sortie)
		}
	}
}

func TestMasquageDesValeursPiegees(t *testing.T) {
	erreurURL := &url.Error{Op: "Get", URL: chaineConnexion, Err: errors.New("connexion refusée")}
	cas := map[string]func(*slog.Logger){
		"jeton dans un attribut":              func(l *slog.Logger) { l.Info("appel", "valeur", jetonPiege) },
		"en-tête Authorization":               func(l *slog.Logger) { l.Info("appel", "Authorization", "Bearer "+opaquePiege) },
		"jeton dans le message":               func(l *slog.Logger) { l.Info("jeton reçu Bearer " + jetonPiege) },
		"chaîne de connexion en attribut":     func(l *slog.Logger) { l.Info("connexion", "base", chaineConnexion) },
		"chaîne de connexion dans le message": func(l *slog.Logger) { l.Error("échec " + chaineConnexion) },
		"erreur url.Error":                    func(l *slog.Logger) { l.Error("échec", "erreur", erreurURL) },
		"erreur enveloppée": func(l *slog.Logger) {
			l.Error("échec", "erreur", fmt.Errorf("vérification : %w", errors.New("jeton "+jetonPiege)))
		},
		"URL présignée en attribut":     func(l *slog.Logger) { l.Info("photo", "url", urlPresignee) },
		"URL présignée en minuscules":   func(l *slog.Logger) { l.Info("photo", "url", urlMinuscules) },
		"URL présignée dans le message": func(l *slog.Logger) { l.Info("téléversement vers " + urlMinuscules) },
		"URL présignée dans une url.Error": func(l *slog.Logger) {
			l.Error("échec", "erreur", &url.Error{Op: "Put", URL: urlPresignee, Err: errors.New("délai dépassé")})
		},
		"groupe imbriqué": func(l *slog.Logger) {
			l.Info("requête", slog.Group("http", slog.Group("entetes", slog.String("Authorization", "Bearer "+opaquePiege), slog.String("valeur", jetonPiege))))
		},
		"journal dérivé avec groupe": func(l *slog.Logger) {
			l.WithGroup("base").With("url", chaineConnexion).Info("connexion", "erreur", erreurURL)
		},
	}
	for nom, journaliser := range cas {
		t.Run(nom, func(t *testing.T) {
			l, sortie := journal(t)
			journaliser(l)
			exigerSansPiege(t, sortie)
		})
	}
}

func TestMasquageConserveLeMessageUtile(t *testing.T) {
	l, sortie := journal(t)
	l.Error("connexion impossible", "base", chaineConnexion)
	for _, attendu := range []string{"connexion impossible", "place-de-marche", "base.collector.test"} {
		if !strings.Contains(sortie.String(), attendu) {
			t.Errorf("%q absent du journal : %s", attendu, sortie)
		}
	}
}

func TestMasquageGardeLesNomsDeParametresPresignes(t *testing.T) {
	l, sortie := journal(t)
	l.Info("photo", "url", urlMinuscules)
	for _, attendu := range []string{"photos.collector.test", "x-amz-credential", "x-amz-security-token", "x-amz-signature"} {
		if !strings.Contains(sortie.String(), attendu) {
			t.Errorf("%q absent du journal : %s", attendu, sortie)
		}
	}
	exigerSansPiege(t, sortie)
}

func TestIdentifiantDeCorrelationJournalise(t *testing.T) {
	l, sortie := journal(t)
	gestionnaire := logging.Middleware(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id, ok := logging.CorrelationID(r.Context()); !ok || id == "" {
			w.WriteHeader(http.StatusTeapot)
			return
		}
		l.InfoContext(r.Context(), "traitement")
		w.WriteHeader(http.StatusNoContent)
	}))
	requete := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", nil)
	requete.Header.Set(logging.CorrelationHeader, "8d1f6a2e-3b4c-4d5e-9f60-718293a4b5c6")
	requete.Header.Set("Authorization", "Bearer "+jetonPiege)
	enregistreur := httptest.NewRecorder()
	gestionnaire.ServeHTTP(enregistreur, requete)
	if enregistreur.Code != http.StatusNoContent {
		t.Fatalf("identifiant de corrélation absent du contexte : %d", enregistreur.Code)
	}
	if !strings.Contains(sortie.String(), "correlation_id") {
		t.Errorf("correlation_id absent du journal : %s", sortie)
	}
	exigerSansPiege(t, sortie)
}

func TestIdentifiantDeCorrelationDuContexte(t *testing.T) {
	ctx := logging.WithCorrelationID(context.Background(), "abc")
	if id, ok := logging.CorrelationID(ctx); !ok || id != "abc" {
		t.Fatalf("identifiant %q %v", id, ok)
	}
	if _, ok := logging.CorrelationID(context.Background()); ok {
		t.Fatal("identifiant inventé sur un contexte vide")
	}
}

func TestNiveauDeJournal(t *testing.T) {
	for brut, attendu := range map[string]slog.Level{"debug": slog.LevelDebug, "info": slog.LevelInfo, "warn": slog.LevelWarn, "error": slog.LevelError} {
		if niveau, err := logging.ParseLevel(brut); err != nil || niveau != attendu {
			t.Errorf("%s : %v, %v", brut, niveau, err)
		}
	}
	if _, err := logging.ParseLevel("bavard"); err == nil {
		t.Error("niveau inconnu accepté")
	}
}
