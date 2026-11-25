package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/services/place-de-marche/internal/migrations"
)

const (
	evenementSuspensionDeCompte = "comptes.suspension"
	auteurExploitation          = "exploitation"
)

type demandeTracee struct {
	correlation string
	trace       string
}

func (b *bancDuFilet) soumettreTrace(t *testing.T, titre string, entetes map[string]string) (articleRecu, string) {
	t.Helper()
	cree := b.creer(t, b.jeton, corpsComplet(titre, categorieSure, prixCourant, 0))
	requete, err := http.NewRequestWithContext(t.Context(), http.MethodPost, b.adresse+cheminSoumission(cree.ID), strings.NewReader(fmt.Sprintf(`{"revision":%d}`, cree.Revision)))
	if err != nil {
		t.Fatalf("requête : %v", err)
	}
	requete.Header.Set("Content-Type", typeJSON)
	requete.Header.Set("Authorization", b.jeton)
	for nom, valeur := range entetes {
		requete.Header.Set(nom, valeur)
	}
	var soumis articleRecu
	r := b.executer(t, requete).exiger(t, http.StatusAccepted)
	r.lire(t, &soumis)
	return soumis, r.entetes.Get(enteteCorrelation)
}

func (b *bancDuFilet) demandesTracees(t *testing.T, id string) []demandeTracee {
	t.Helper()
	lignes, err := b.base.maitre.Query(t.Context(), `
		SELECT correlation_id::text, coalesce(traceparent, '') FROM outbox
		WHERE type_message = $2 AND convert_from(corps, 'UTF8')::jsonb -> 'data' ->> 'article_id' = $1 ORDER BY cree_le`, id, typeDemande)
	if err != nil {
		t.Fatalf("outbox : %v", err)
	}
	defer lignes.Close()
	var demandes []demandeTracee
	for lignes.Next() {
		var d demandeTracee
		var traceparent string
		if err := lignes.Scan(&d.correlation, &traceparent); err != nil {
			t.Fatalf("outbox : %v", err)
		}
		if morceaux := formatTraceparent.FindStringSubmatch(traceparent); morceaux != nil {
			d.trace = morceaux[1]
		}
		demandes = append(demandes, d)
	}
	return demandes
}

func exigerRelanceTracee(t *testing.T, journal *journalPartage, id string, rang int, attendu demandeTracee) {
	t.Helper()
	attendre(t, delaiDeRelance, fmt.Sprintf("relance de rang %d", rang), func() bool { return len(evenementsPortant(journal, evenementRelance, id)) >= rang })
	relance := evenementsPortant(journal, evenementRelance, id)[rang-1]
	if relance["correlation_id"] != attendu.correlation || relance["trace_id"] != attendu.trace {
		t.Errorf("journal de relance : correlation %v, trace %v ; %s et %s attendus : %v", relance["correlation_id"], relance["trace_id"], attendu.correlation, attendu.trace, relance)
	}
}

func exigerDemandesTracees(t *testing.T, demandes []demandeTracee, attendu demandeTracee, nombre int) {
	t.Helper()
	if len(demandes) != nombre {
		t.Fatalf("%d demande(s) en outbox, %d attendue(s) : %+v", len(demandes), nombre, demandes)
	}
	for i, d := range demandes {
		if d != attendu {
			t.Errorf("demande %d : %+v, %+v attendue", i+1, d, attendu)
		}
	}
}

func TestUS014RelanceDuFiletGardeLaCorrelationEtLaTraceDeLaSoumission(t *testing.T) {
	b := demarrerBancDuFilet(t)
	journal := b.lancerLeFilet(t, rythmeDeProductionBalayeVite())

	t.Run("relance de la soumission : même correlation_id et même trace dans l'outbox et le journal", func(t *testing.T) {
		attendu := demandeTracee{correlation: uuid.NewString(), trace: nouvelleTrace()}
		soumis, renvoyee := b.soumettreTrace(t, "Figurine Bianca relancée", entetesDeTrace(attendu.correlation, attendu.trace))
		if renvoyee != attendu.correlation {
			t.Fatalf("X-Correlation-Id renvoyé %q, %q attendu", renvoyee, attendu.correlation)
		}
		b.dater(t, soumis.ID, 0, 3*time.Minute)
		exigerRelanceTracee(t, journal, soumis.ID, 1, attendu)
		exigerDemandesTracees(t, b.demandesTracees(t, soumis.ID), attendu, 2)
		b.dater(t, soumis.ID, 1, 5*time.Minute)
		exigerRelanceTracee(t, journal, soumis.ID, 2, attendu)
		exigerDemandesTracees(t, b.demandesTracees(t, soumis.ID), attendu, 3)
	})
	t.Run("soumission sans en-tête : la relance reprend les identifiants tirés à la soumission", func(t *testing.T) {
		soumis, correlation := b.soumettreTrace(t, "Figurine Séraphin relancée", nil)
		demandes := b.demandesTracees(t, soumis.ID)
		if len(demandes) != 1 || demandes[0].correlation != correlation || demandes[0].trace == "" {
			t.Fatalf("demande de la soumission %+v, corrélation %s attendue avec une trace", demandes, correlation)
		}
		b.dater(t, soumis.ID, 0, 3*time.Minute)
		exigerRelanceTracee(t, journal, soumis.ID, 1, demandes[0])
		exigerDemandesTracees(t, b.demandesTracees(t, soumis.ID), demandes[0], 2)
	})
	t.Run("article soumis avant la migration 00011 : identifiants tirés à la première relance puis gardés", func(t *testing.T) {
		soumis, _ := b.soumettreTrace(t, "Figurine Nestor héritée", nil)
		if _, err := b.base.maitre.Exec(t.Context(), `UPDATE article SET demande_controle_correlation_id = NULL, demande_controle_traceparent = NULL WHERE id = $1`, soumis.ID); err != nil {
			t.Fatalf("article hérité : %v", err)
		}
		b.dater(t, soumis.ID, 0, 3*time.Minute)
		attendre(t, delaiDeRelance, "première relance de l'article hérité", func() bool { return len(evenementsPortant(journal, evenementRelance, soumis.ID)) >= 1 })
		demandes := b.demandesTracees(t, soumis.ID)
		premiere := demandes[len(demandes)-1]
		if _, err := uuid.Parse(premiere.correlation); err != nil || premiere.trace == "" {
			t.Fatalf("première relance d'un article hérité sans identifiants valides : %+v", premiere)
		}
		exigerRelanceTracee(t, journal, soumis.ID, 1, premiere)
		b.dater(t, soumis.ID, 1, 5*time.Minute)
		exigerRelanceTracee(t, journal, soumis.ID, 2, premiere)
		if derniere := b.demandesTracees(t, soumis.ID); derniere[len(derniere)-1] != premiere {
			t.Errorf("seconde relance %+v, identifiants de la première %+v attendus", derniere[len(derniere)-1], premiere)
		}
	})
}

type bancDesSuspensions struct {
	base     basePostgreSQL
	commande string
	sujet    string
}

func lignesDeSortie(sortie string) []map[string]any {
	journal := &journalPartage{}
	journal.consigner(sortie)
	lignes, _ := lignesJSON(journal)
	return lignes
}

func (b *bancDesSuspensions) inscrire(t *testing.T) string {
	t.Helper()
	sujet := uuid.NewString()
	if _, err := b.base.maitre.Exec(t.Context(), `INSERT INTO compte (id, inscrit_le) VALUES ($1, now())`, sujet); err != nil {
		t.Fatalf("compte : %v", err)
	}
	return sujet
}

func (b *bancDesSuspensions) auditsDe(t *testing.T, action, sujet string) ([]map[string]any, sortieDeCommande) {
	t.Helper()
	s := lancerUneCommande(t, b.commande, b.base.migration, []string{action, sujet})
	var audits []map[string]any
	for _, l := range lignesDeSortie(s.sortie) {
		if l["event"] == evenementSuspensionDeCompte {
			audits = append(audits, l)
		}
	}
	return audits, s
}

func exigerAuditDeCompte(t *testing.T, audits []map[string]any, sujet, action string, suspendu bool) {
	t.Helper()
	if len(audits) != 1 {
		t.Fatalf("%d événement(s) %s, un attendu : %v", len(audits), evenementSuspensionDeCompte, audits)
	}
	a := audits[0]
	if a["audit"] != true || a["compte_id"] != sujet || a["auteur"] != auteurExploitation || a["action"] != action || a["suspendu"] != suspendu || a["service"] != serviceName {
		t.Errorf("audit de compte incomplet (audit, compte_id, auteur, action, suspendu, service) : %v", a)
	}
	if _, err := time.Parse(time.RFC3339Nano, fmt.Sprint(a["time"])); err != nil {
		t.Errorf("audit non horodaté : %v", a)
	}
	if _, err := uuid.Parse(fmt.Sprint(a["correlation_id"])); err != nil || !formatTrace.MatchString(fmt.Sprint(a["trace_id"])) {
		t.Errorf("audit de compte sans correlation_id ni trace_id : %v", a)
	}
	for _, cle := range []string{"kubernetes", "namespace", "pod", "origine"} {
		if _, present := a[cle]; present {
			t.Errorf("champ d'origine %q écrit par le service : %v", cle, a)
		}
	}
}

func (b *bancDesSuspensions) actionAuditee(action string, suspendu bool) func(*testing.T) {
	return func(t *testing.T) {
		audits, s := b.auditsDe(t, action, b.sujet)
		if s.code != 0 {
			t.Fatalf("%s : code %d\n%s", action, s.code, s.sortie)
		}
		exigerAuditDeCompte(t, audits, b.sujet, action, suspendu)
	}
}

func (b *bancDesSuspensions) compteInconnuSansAudit(t *testing.T) {
	for _, action := range []string{commandeSuspendre, commandeRetablir} {
		audits, s := b.auditsDe(t, action, uuid.NewString())
		if s.code == 0 {
			t.Errorf("%s d'un compte inconnu : code 0", action)
		}
		if len(audits) != 0 {
			t.Errorf("%s d'un compte inconnu : audit émis %v", action, audits)
		}
	}
}

func (b *bancDesSuspensions) sortieSansSecretEtEnJSON(t *testing.T) {
	_, s := b.auditsDe(t, commandeSuspendre, b.sujet)
	for _, secret := range motsDePasseDesAdresses(t, b.base.migration) {
		if strings.Contains(s.sortie, secret) {
			t.Errorf("mot de passe de marche_migration présent dans la sortie")
		}
	}
	lecteur := bufio.NewScanner(strings.NewReader(s.sortie))
	for lecteur.Scan() {
		if ligne := lecteur.Text(); ligne != "" && !strings.HasPrefix(ligne, "{") {
			t.Errorf("ligne non JSON dans la sortie de suspendre : %q", ligne)
		}
	}
}

func TestUS014AuditDeSuspensionEtDeRetablissementDUnCompte(t *testing.T) {
	ctx := contexteBorne(t, 3*time.Minute)
	b := &bancDesSuspensions{base: amorcer(ctx, t, demarrerPostgreSQL(t)), commande: construirePlaceDeMarche(t)}
	if err := migrations.Apply(ctx, journalMuet(), b.base.migration); err != nil {
		t.Fatalf("migrations : %v", err)
	}
	b.sujet = b.inscrire(t)

	t.Run("suspendre : audit horodaté avec compte, auteur, action, sans champ d'origine", b.actionAuditee(commandeSuspendre, true))
	t.Run("suspendre de nouveau : audit de nouveau émis", b.actionAuditee(commandeSuspendre, true))
	t.Run("rétablir : audit horodaté avec compte, auteur, action", b.actionAuditee(commandeRetablir, false))
	t.Run("rétablir de nouveau : audit de nouveau émis", b.actionAuditee(commandeRetablir, false))
	t.Run("compte inconnu : arrêt en erreur sans audit", b.compteInconnuSansAudit)
	t.Run("aucun secret de la base, sortie entièrement JSON", b.sortieSansSecretEtEnJSON)
}
