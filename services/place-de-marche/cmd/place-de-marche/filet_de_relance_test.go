package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"collector/libs/platform/logging"
	"collector/services/place-de-marche/internal/annonces/controle"
	"collector/services/place-de-marche/internal/annonces/stockage"
	comptesstockage "collector/services/place-de-marche/internal/comptes/stockage"
)

const (
	balayageRapide     = 50 * time.Millisecond
	attenteSansRelance = time.Second
	delaiDeRelance     = 5 * time.Second
	toleranceDHorloge  = 50 * time.Millisecond
)

type bancDuFilet struct {
	*environnement
	base  basePostgreSQL
	pool  *pgxpool.Pool
	store *stockage.Store
	jeton string
}

type relanceJournalisee struct {
	relances      int
	prochaineDans time.Duration
	le            time.Time
}

func demarrerBancDuFilet(t *testing.T) *bancDuFilet {
	t.Helper()
	base, pool, service := servicePostgreSQL(t)
	e := demarrer(t, service)
	return &bancDuFilet{
		environnement: e,
		base:          base,
		pool:          pool,
		store:         stockage.NewStore(pool, controle.NewRequestEncoder(validateurDeContrats(t)), comptesstockage.NewStore(pool)),
		jeton:         e.vendeur(),
	}
}

func rythmeDeProductionBalayeVite() controle.SafeguardTiming {
	rythme := productionSafeguardTiming
	rythme.SweepEvery = balayageRapide
	return rythme
}

func (b *bancDuFilet) lancerLeFilet(t *testing.T, rythme controle.SafeguardTiming) *journalPartage {
	t.Helper()
	journal := &journalPartage{}
	filet, err := controle.NewSafeguard(b.store, rythme, logging.New(serviceName, journal, slog.LevelInfo))
	if err != nil {
		t.Fatalf("filet de relance : %v", err)
	}
	ctx, arreter := context.WithCancel(context.Background())
	termine := make(chan error, 1)
	go func() { termine <- filet.Run(ctx) }()
	t.Cleanup(func() {
		arreter()
		if err := <-termine; err != nil {
			t.Errorf("arrêt du filet : %v", err)
		}
	})
	return journal
}

func (b *bancDuFilet) soumettreUnArticle(t *testing.T, titre string) articleRecu {
	t.Helper()
	return b.soumettreAvecSucces(t, b.jeton, b.creer(t, b.jeton, corpsComplet(titre, categorieSure, prixCourant, 0)))
}

func (b *bancDuFilet) dater(t *testing.T, id string, relances int, depuis time.Duration) {
	t.Helper()
	_, err := b.base.maitre.Exec(t.Context(), `
		UPDATE article SET relances = $2, demande_controle_le = now() - make_interval(secs => $3) WHERE id = $1`,
		id, relances, depuis.Seconds())
	if err != nil {
		t.Fatalf("datation de la demande : %v", err)
	}
}

func (b *bancDuFilet) relancesEnBase(t *testing.T, id string) int {
	t.Helper()
	var relances int
	if err := b.base.maitre.QueryRow(t.Context(), `SELECT relances FROM article WHERE id = $1`, id).Scan(&relances); err != nil {
		t.Fatalf("relances : %v", err)
	}
	return relances
}

func (b *bancDuFilet) demandesEnOutbox(t *testing.T, id string) []demandeRecue {
	t.Helper()
	lignes, err := b.base.maitre.Query(t.Context(), `
		SELECT corps FROM outbox WHERE convert_from(corps, 'UTF8')::jsonb -> 'data' ->> 'article_id' = $1 ORDER BY cree_le`, id)
	if err != nil {
		t.Fatalf("outbox : %v", err)
	}
	defer lignes.Close()
	validateur := validateurDeContrats(t)
	var demandes []demandeRecue
	for lignes.Next() {
		var corps []byte
		if err := lignes.Scan(&corps); err != nil {
			t.Fatalf("outbox : %v", err)
		}
		var d demandeRecue
		if _, err := validateur.Decode(corps, &d); err != nil {
			t.Fatalf("demande relancée non conforme au contrat v1 : %v", err)
		}
		demandes = append(demandes, d)
	}
	return demandes
}

func relancesJournalisees(t *testing.T, journal *journalPartage, id string) []relanceJournalisee {
	t.Helper()
	var relances []relanceJournalisee
	for _, e := range evenementsPortant(journal, evenementRelance, id) {
		nombre, _ := e["relances"].(float64)
		prochaine, _ := e["prochaine_dans"].(float64)
		le, err := time.Parse(time.RFC3339Nano, e["time"].(string))
		if err != nil {
			t.Fatalf("horodatage du journal : %v", err)
		}
		relances = append(relances, relanceJournalisee{relances: int(nombre), prochaineDans: time.Duration(prochaine), le: le})
	}
	return relances
}

func exigerAucuneRelance(t *testing.T, journal *journalPartage, id string, deja int) {
	t.Helper()
	time.Sleep(attenteSansRelance)
	if n := len(relancesJournalisees(t, journal, id)); n != deja {
		t.Fatalf("%d relance(s) journalisée(s), %d attendue(s) : relance avant l'échéance", n, deja)
	}
}

func exigerRelance(t *testing.T, journal *journalPartage, id string, rang int, prochaine time.Duration) {
	t.Helper()
	attendre(t, delaiDeRelance, fmt.Sprintf("relance de rang %d", rang), func() bool {
		return len(relancesJournalisees(t, journal, id)) >= rang
	})
	relances := relancesJournalisees(t, journal, id)
	if len(relances) != rang {
		t.Fatalf("%d relance(s) journalisée(s), %d attendue(s)", len(relances), rang)
	}
	derniere := relances[rang-1]
	if derniere.prochaineDans != prochaine {
		t.Errorf("relance %d : prochaine dans %s, %s attendu", rang, derniere.prochaineDans, prochaine)
	}
}

func TestUS024FiletDelaiCroissantAuxValeursDeProduction(t *testing.T) {
	b := demarrerBancDuFilet(t)
	journal := b.lancerLeFilet(t, rythmeDeProductionBalayeVite())
	article := b.soumettreUnArticle(t, "Filet de production")
	t.Run("US-024-AC1 pas de relance avant 2 min", func(t *testing.T) {
		b.dater(t, article.ID, 0, 2*time.Minute-time.Second)
		exigerAucuneRelance(t, journal, article.ID, 0)
		if n := len(b.demandesEnOutbox(t, article.ID)); n != 1 || b.relancesEnBase(t, article.ID) != 0 {
			t.Fatalf("%d demande(s), %d relance(s) en base", n, b.relancesEnBase(t, article.ID))
		}
	})
	t.Run("US-024-AC1 relance à 2 min pour la même révision, prochaine dans 4 min", func(t *testing.T) {
		b.dater(t, article.ID, 0, 2*time.Minute+time.Second)
		exigerRelance(t, journal, article.ID, 1, 4*time.Minute)
		demandes := b.demandesEnOutbox(t, article.ID)
		if len(demandes) != 2 || demandes[1].ArticleID != article.ID || demandes[1].Revision != article.Revision || b.relancesEnBase(t, article.ID) != 1 {
			t.Fatalf("demandes %+v, %d relance(s) en base", demandes, b.relancesEnBase(t, article.ID))
		}
		exigerEtat(t, b.relire(t, b.jeton, article.ID), etatEnControle, "")
	})
	t.Run("US-024-AC1 pas de seconde relance avant 4 min, puis prochaine dans 8 min", func(t *testing.T) {
		b.dater(t, article.ID, 1, 4*time.Minute-time.Second)
		exigerAucuneRelance(t, journal, article.ID, 1)
		b.dater(t, article.ID, 1, 4*time.Minute+time.Second)
		exigerRelance(t, journal, article.ID, 2, 8*time.Minute)
	})
	t.Run("US-024-AC1 plafond à 1 h dès 5 relances", func(t *testing.T) {
		b.dater(t, article.ID, 4, 32*time.Minute+time.Second)
		exigerRelance(t, journal, article.ID, 3, time.Hour)
		if n := b.relancesEnBase(t, article.ID); n != 5 {
			t.Fatalf("%d relance(s) en base, 5 attendues", n)
		}
		b.dater(t, article.ID, 5, time.Hour-time.Minute)
		exigerAucuneRelance(t, journal, article.ID, 3)
		b.dater(t, article.ID, 5, time.Hour+time.Minute)
		exigerRelance(t, journal, article.ID, 4, time.Hour)
	})
	t.Run("US-024-AC1 relances sans arrêt au-delà du plafond", func(t *testing.T) {
		b.dater(t, article.ID, 1000, time.Hour+time.Minute)
		exigerRelance(t, journal, article.ID, 5, time.Hour)
		if n := b.relancesEnBase(t, article.ID); n != 1001 {
			t.Fatalf("%d relance(s) en base, 1001 attendues", n)
		}
	})
}

func TestUS024FiletRemiseAZeroDesRelances(t *testing.T) {
	b := demarrerBancDuFilet(t)
	journal := b.lancerLeFilet(t, rythmeDeProductionBalayeVite())
	t.Run("US-024-AC1 recontrôle après modification : relances remises à 0", func(t *testing.T) {
		article := b.soumettreUnArticle(t, "Filet recontrôlé")
		b.dater(t, article.ID, 3, 16*time.Minute+time.Second)
		exigerRelance(t, journal, article.ID, 1, 32*time.Minute)
		var modifie articleRecu
		b.modifier(t, b.jeton, article.ID, corpsRevision(article.Revision, corpsComplet("Filet recontrôlé modifié", categorieSure, prixCourant, 0))).exiger(t, http.StatusOK).lire(t, &modifie)
		if n := b.relancesEnBase(t, article.ID); n != 0 {
			t.Fatalf("%d relance(s) après le recontrôle, 0 attendue", n)
		}
		b.dater(t, article.ID, 0, 2*time.Minute+time.Second)
		exigerRelance(t, journal, article.ID, 2, 4*time.Minute)
		if demandes := b.demandesEnOutbox(t, article.ID); demandes[len(demandes)-1].Revision != modifie.Revision {
			t.Fatalf("relance en révision %d, %d attendue", demandes[len(demandes)-1].Revision, modifie.Revision)
		}
	})
	t.Run("US-024-AC1 nouvelle soumission après refus : relances remises à 0", func(t *testing.T) {
		article := b.soumettreUnArticle(t, "Filet resoumis")
		b.dater(t, article.ID, 2, 8*time.Minute+time.Second)
		exigerRelance(t, journal, article.ID, 1, 16*time.Minute)
		if _, err := b.base.maitre.Exec(t.Context(), `UPDATE article SET etat = 'refuse', motif = 'annonce_incomplete' WHERE id = $1`, article.ID); err != nil {
			t.Fatalf("refus : %v", err)
		}
		b.soumettreAvecSucces(t, b.jeton, b.relire(t, b.jeton, article.ID))
		if n := b.relancesEnBase(t, article.ID); n != 0 {
			t.Fatalf("%d relance(s) après la nouvelle soumission, 0 attendue", n)
		}
	})
}

func TestUS024FiletRythmeCourtInjecte(t *testing.T) {
	b := demarrerBancDuFilet(t)
	rythme := controle.SafeguardTiming{FirstDelay: 100 * time.Millisecond, MaxDelay: 3 * time.Second, SweepEvery: 20 * time.Millisecond}
	article := b.soumettreUnArticle(t, "Filet au rythme court")
	soumisLe := time.Now()
	journal := b.lancerLeFilet(t, rythme)
	attendus := []time.Duration{200 * time.Millisecond, 400 * time.Millisecond, 800 * time.Millisecond, 1600 * time.Millisecond, 3 * time.Second, 3 * time.Second}
	attendre(t, 30*time.Second, "six relances", func() bool { return len(relancesJournalisees(t, journal, article.ID)) >= len(attendus) })
	relances := relancesJournalisees(t, journal, article.ID)[:len(attendus)]
	precedente, attente := soumisLe, rythme.FirstDelay
	for i, r := range relances {
		if r.relances != i+1 || r.prochaineDans != attendus[i] {
			t.Errorf("relance %d : relances %d, prochaine dans %s ; attendu %d et %s", i+1, r.relances, r.prochaineDans, i+1, attendus[i])
		}
		if ecart := r.le.Sub(precedente); ecart < attente-toleranceDHorloge {
			t.Errorf("relance %d %s après la précédente, échéance %s", i+1, ecart, attente)
		}
		precedente, attente = r.le, r.prochaineDans
	}
	for _, d := range b.demandesEnOutbox(t, article.ID) {
		if d.Revision != article.Revision {
			t.Errorf("demande en révision %d, %d attendue", d.Revision, article.Revision)
		}
	}
}

func TestUS024FiletBalayageAuDemarrage(t *testing.T) {
	b := demarrerBancDuFilet(t)
	article := b.soumettreUnArticle(t, "Filet au démarrage")
	b.dater(t, article.ID, 0, 3*time.Minute)
	lance := time.Now()
	journal := b.lancerLeFilet(t, controle.SafeguardTiming{FirstDelay: 2 * time.Minute, MaxDelay: time.Hour, SweepEvery: time.Hour})
	exigerRelance(t, journal, article.ID, 1, 4*time.Minute)
	t.Logf("relance %s après le démarrage du filet", time.Since(lance).Round(time.Millisecond))
}

func TestUS024FiletRythmesInvalidesRefuses(t *testing.T) {
	invalides := map[string]controle.SafeguardTiming{
		"tout à zéro":                   {},
		"délai initial nul":             {FirstDelay: 0, MaxDelay: time.Hour, SweepEvery: time.Second},
		"délai initial négatif":         {FirstDelay: -time.Minute, MaxDelay: time.Hour, SweepEvery: time.Second},
		"plafond sous le délai initial": {FirstDelay: time.Hour, MaxDelay: time.Minute, SweepEvery: time.Second},
		"période nulle":                 {FirstDelay: time.Minute, MaxDelay: time.Hour},
		"période négative":              {FirstDelay: time.Minute, MaxDelay: time.Hour, SweepEvery: -time.Second},
	}
	for nom, rythme := range invalides {
		t.Run(nom, func(t *testing.T) {
			if filet, err := controle.NewSafeguard(nil, rythme, journalMuet()); err == nil || filet != nil {
				t.Fatalf("rythme %+v accepté", rythme)
			}
		})
	}
	valides := map[string]controle.SafeguardTiming{
		"production":                    productionSafeguardTiming,
		"plafond égal au délai initial": {FirstDelay: time.Minute, MaxDelay: time.Minute, SweepEvery: time.Second},
	}
	for nom, rythme := range valides {
		t.Run(nom, func(t *testing.T) {
			if filet, err := controle.NewSafeguard(nil, rythme, journalMuet()); err != nil || filet == nil {
				t.Fatalf("rythme %+v refusé : %v", rythme, err)
			}
		})
	}
}
