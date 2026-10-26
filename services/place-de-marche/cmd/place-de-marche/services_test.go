package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"collector/libs/platform/logging"
)

func adresseLibre(t *testing.T) string {
	t.Helper()
	ecoute, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("port libre : %v", err)
	}
	adresse := ecoute.Addr().String()
	_ = ecoute.Close()
	return adresse
}

func attendre(t *testing.T, delai time.Duration, quoi string, condition func() bool) time.Duration {
	t.Helper()
	debut := time.Now()
	for time.Since(debut) < delai {
		if condition() {
			return time.Since(debut)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("%s : non atteint en %s", quoi, delai)
	return delai
}

func repond(client *http.Client, adresse string) bool {
	reponse, err := client.Get(adresse)
	if err != nil {
		return false
	}
	_ = reponse.Body.Close()
	return reponse.StatusCode == http.StatusOK
}

func lancerPlaceDeMarche(t *testing.T, canal *canalDeTest, adresseBase string, emetteur emetteurDeJetons) (*environnement, *journalPartage) {
	t.Helper()
	api, gestion := adresseLibre(t), adresseLibre(t)
	variables := canal.environnementDeService(identiteMarche)
	variables["BASE_DE_DONNEES_URL"] = adresseBase
	variables["BASE_DE_DONNEES_TLS_EXIGE"] = "false"
	variables["OIDC_EMETTEUR"] = emetteurAttendu
	variables["OIDC_JEU_DE_CLES_URL"] = emetteur.jwks.URL
	variables["CANAL_EMETTEUR_ATTENDU"] = identiteControle
	variables["ADRESSE_API"] = api
	variables["ADRESSE_GESTION"] = gestion
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
			t.Fatalf("la Place de marché s'est arrêtée : %v", err)
		default:
		}
		return repond(client, "http://"+gestion+"/disponibilite")
	})
	return emetteur.environnementSur(t, "http://"+api, client), journal
}

func construireControle(t *testing.T) string {
	t.Helper()
	source, err := filepath.Abs(filepath.Join("..", "..", "..", "controle", "cmd", "controle"))
	if err != nil {
		t.Fatalf("sources du Contrôle : %v", err)
	}
	repertoire := t.TempDir()
	binaire := filepath.Join(repertoire, "controle")
	if runtime.GOOS == "windows" {
		binaire += ".exe"
	}
	construction := exec.Command("go", "install", ".")
	construction.Dir = source
	construction.Env = append(os.Environ(), "GOBIN="+repertoire)
	if sortie, err := construction.CombinedOutput(); err != nil {
		t.Fatalf("construction du Contrôle : %v\n%s", err, sortie)
	}
	return binaire
}

type processusControle struct {
	journal *journalPartage
	gestion string
	termine chan error
	arret   *exec.Cmd
	une     sync.Once
}

func demarrerProcessusControle(t *testing.T, binaire string, canal *canalDeTest) *processusControle {
	t.Helper()
	p := &processusControle{journal: &journalPartage{}, gestion: adresseLibre(t), termine: make(chan error, 1)}
	p.arret = exec.Command(binaire)
	p.arret.Env = os.Environ()
	for nom, valeur := range canal.environnementDeService(identiteControle) {
		p.arret.Env = append(p.arret.Env, nom+"="+valeur)
	}
	p.arret.Env = append(p.arret.Env, "CANAL_EMETTEUR_ATTENDU="+identiteMarche, "ADRESSE_GESTION="+p.gestion)
	p.arret.Stdout, p.arret.Stderr = p.journal, p.journal
	if err := p.arret.Start(); err != nil {
		t.Fatalf("lancement du Contrôle : %v", err)
	}
	go func() { p.termine <- p.arret.Wait() }()
	t.Cleanup(p.arreter)
	return p
}

func (p *processusControle) arreter() {
	p.une.Do(func() {
		_ = p.arret.Process.Kill()
		select {
		case erreur := <-p.termine:
			p.journal.consigner("arrêt : " + fmt.Sprint(erreur))
		case <-time.After(10 * time.Second):
		}
	})
}

func (p *processusControle) disponible(client *http.Client) bool {
	return repond(client, "http://"+p.gestion+"/disponibilite")
}

func lancerControle(t *testing.T, binaire string, canal *canalDeTest) *journalPartage {
	t.Helper()
	p := demarrerProcessusControle(t, binaire, canal)
	client := &http.Client{Timeout: 2 * time.Second}
	attendre(t, 30*time.Second, "démarrage du Contrôle", func() bool {
		select {
		case err := <-p.termine:
			t.Fatalf("le Contrôle s'est arrêté : %v", err)
		default:
		}
		return p.disponible(client)
	})
	return p.journal
}
