package canal_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"collector/contracts"
	"collector/libs/platform/broker"
	"collector/libs/platform/message"
	"collector/services/interactions/internal/notifications"
	"collector/services/interactions/internal/notifications/canal"
)

const (
	dossierDesExemples = "messages/publication.decidee/exemples/v1"
	emetteurMarche     = "marche"
)

var (
	errBaseIndisponible = errors.New("base indisponible")
	issuesFermees       = []string{"publie", "refuse", "a_verifier"}
	motifsFermes        = []string{"annonce_incomplete", "coordonnees_detectees", "compte_non_autorise", "prix_a_confirmer", "vendeur_a_verifier", "categorie_inconnue", "prix_abusif", "article_interdit", "vendeur_non_fiable"}
)

type enregistreurEnMemoire struct {
	mu        sync.Mutex
	decisions []notifications.Decision
	vues      map[string]bool
	panne     error
}

func nouvelEnregistreur() *enregistreurEnMemoire {
	return &enregistreurEnMemoire{vues: map[string]bool{}}
}

func (e *enregistreurEnMemoire) Record(_ context.Context, decision notifications.Decision) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.panne != nil {
		return false, e.panne
	}
	e.decisions = append(e.decisions, decision)
	if e.vues[decision.MessageID] {
		return false, nil
	}
	e.vues[decision.MessageID] = true
	return true, nil
}

func (e *enregistreurEnMemoire) recues() []notifications.Decision {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]notifications.Decision(nil), e.decisions...)
}

func (e *enregistreurEnMemoire) creees() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.vues)
}

type exemple struct {
	nom   string
	corps []byte
}

func exemplesDe(t *testing.T, sorte string) []exemple {
	t.Helper()
	racine := os.DirFS(filepath.Join("..", "..", "..", "..", "..", "contracts"))
	var trouves []exemple
	err := fs.WalkDir(racine, path.Join(dossierDesExemples, sorte), func(p string, entree fs.DirEntry, err error) error {
		if err != nil || entree.IsDir() {
			return err
		}
		corps, err := fs.ReadFile(racine, p)
		trouves = append(trouves, exemple{nom: strings.TrimPrefix(p, dossierDesExemples+"/"), corps: corps})
		return err
	})
	if err != nil || len(trouves) == 0 {
		t.Fatalf("exemples %s introuvables : %v", sorte, err)
	}
	return trouves
}

func traitant(t *testing.T, enregistreur *enregistreurEnMemoire, journal io.Writer) *canal.DecisionHandler {
	t.Helper()
	validateur, err := message.NewValidator(contracts.Messages, message.DefaultMaxSize)
	if err != nil {
		t.Fatalf("validateur : %v", err)
	}
	return canal.NewDecisionHandler(validateur, enregistreur, slog.New(slog.NewJSONHandler(journal, nil)))
}

func livrer(t *testing.T, h *canal.DecisionHandler, corps []byte) (broker.Disposition, error) {
	t.Helper()
	return h.Handle(t.Context(), broker.Delivery{MessageID: uuid.NewString(), UserID: emetteurMarche, Body: corps})
}

type donneesAttendues struct {
	ArticleID string  `json:"article_id"`
	Revision  int     `json:"revision"`
	SellerID  string  `json:"seller_id"`
	Issue     string  `json:"issue"`
	Reason    *string `json:"reason"`
}

type enveloppeAttendue struct {
	ID   string           `json:"id"`
	Data donneesAttendues `json:"data"`
}

func lireEnveloppe(t *testing.T, corps []byte) enveloppeAttendue {
	t.Helper()
	var e enveloppeAttendue
	if err := json.Unmarshal(corps, &e); err != nil {
		t.Fatalf("exemple illisible : %v", err)
	}
	return e
}

func motifDe(d notifications.Decision) *string {
	if d.Reason == nil {
		return nil
	}
	code := string(*d.Reason)
	return &code
}

func memeDecision(a, b notifications.Decision) bool {
	return a.MessageID == b.MessageID && a.Seller == b.Seller && a.Article == b.Article && a.Revision == b.Revision &&
		a.Issue == b.Issue && memeMotif(motifDe(a), motifDe(b)) && a.DecidedAt.Equal(b.DecidedAt)
}

func memeMotif(a, b *string) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func TestUS028AC9_ChaqueDecisionConformeAuContratEstAcceptee(t *testing.T) {
	for _, ex := range exemplesDe(t, "valides") {
		t.Run(ex.nom, func(t *testing.T) {
			enregistreur := nouvelEnregistreur()
			disposition, err := livrer(t, traitant(t, enregistreur, io.Discard), ex.corps)
			if err != nil || disposition != broker.Ack {
				t.Fatalf("décision conforme : disposition %v, erreur %v ; acquittement attendu", disposition, err)
			}
			attendu := lireEnveloppe(t, ex.corps).Data
			recues := enregistreur.recues()
			if len(recues) != 1 {
				t.Fatalf("%d notification(s) enregistrée(s), 1 attendue", len(recues))
			}
			d := recues[0]
			if d.Article.String() != attendu.ArticleID || int(d.Revision) != attendu.Revision || d.Seller.String() != attendu.SellerID ||
				string(d.Issue) != attendu.Issue || !memeMotif(motifDe(d), attendu.Reason) {
				t.Fatalf("décision enregistrée %+v, attendue %+v", d, attendu)
			}
		})
	}
}

func TestUS028AC2_UneDecisionCreeUneSeuleNotification(t *testing.T) {
	enregistreur := nouvelEnregistreur()
	h := traitant(t, enregistreur, io.Discard)
	for _, ex := range exemplesDe(t, "valides") {
		if _, err := livrer(t, h, ex.corps); err != nil {
			t.Fatalf("%s : %v", ex.nom, err)
		}
	}
	if n := enregistreur.creees(); n != len(exemplesDe(t, "valides")) {
		t.Fatalf("%d notification(s) pour %d décision(s) distinctes", n, len(exemplesDe(t, "valides")))
	}
}

func TestUS028AC3_MemeDecisionLivreeDeuxFoisUneSeuleNotification(t *testing.T) {
	corps := exemplesDe(t, "valides")[0].corps
	enregistreur := nouvelEnregistreur()
	journal := &bytes.Buffer{}
	h := traitant(t, enregistreur, journal)

	premiere, errPremiere := livrer(t, h, corps)
	seconde, errSeconde := h.Handle(t.Context(), broker.Delivery{MessageID: uuid.NewString(), UserID: emetteurMarche, Redelivered: true, Body: corps})

	if errPremiere != nil || errSeconde != nil || premiere != broker.Ack || seconde != broker.Ack {
		t.Fatalf("livraisons : %v %v / %v %v ; deux acquittements attendus", premiere, errPremiere, seconde, errSeconde)
	}
	recues := enregistreur.recues()
	if len(recues) != 2 || !memeDecision(recues[0], recues[1]) {
		t.Fatalf("les deux livraisons doivent porter la même décision : %+v", recues)
	}
	if recues[0].MessageID != lireEnveloppe(t, corps).ID {
		t.Fatalf("décision identifiée par %q, attendu l'identifiant de l'événement %q", recues[0].MessageID, lireEnveloppe(t, corps).ID)
	}
	if n := enregistreur.creees(); n != 1 {
		t.Fatalf("%d notification(s), 1 attendue", n)
	}
	if !strings.Contains(journal.String(), "notifications.doublon_ignore") {
		t.Fatalf("doublon non journalisé : %s", journal.String())
	}
}

func TestUS028AC3_DeuxDecisionsDuMemeArticleDonnentDeuxNotifications(t *testing.T) {
	enregistreur := nouvelEnregistreur()
	h := traitant(t, enregistreur, io.Discard)
	modele := exemplesDe(t, "valides")[0].corps
	for range 2 {
		var document map[string]any
		if err := json.Unmarshal(modele, &document); err != nil {
			t.Fatalf("modèle : %v", err)
		}
		document["id"] = uuid.NewString()
		corps, _ := json.Marshal(document)
		if disposition, err := livrer(t, h, corps); err != nil || disposition != broker.Ack {
			t.Fatalf("disposition %v, erreur %v", disposition, err)
		}
	}
	if n := enregistreur.creees(); n != 2 {
		t.Fatalf("%d notification(s) pour deux décisions distinctes, 2 attendues", n)
	}
}

func TestUS028AC5_MessageInvalideEcarteSansNotification(t *testing.T) {
	for _, ex := range exemplesDe(t, "invalides") {
		t.Run(ex.nom, func(t *testing.T) {
			enregistreur := nouvelEnregistreur()
			disposition, err := livrer(t, traitant(t, enregistreur, io.Discard), ex.corps)
			if err != nil || disposition != broker.Reject {
				t.Fatalf("message invalide : disposition %v, erreur %v ; mise à l'écart attendue", disposition, err)
			}
			if n := len(enregistreur.recues()); n != 0 {
				t.Fatalf("%d notification(s) créée(s) pour un message invalide", n)
			}
		})
	}
}

func TestUS028AC5_MessageHorsContratEcarteSansNotification(t *testing.T) {
	valide := exemplesDe(t, "valides")[0].corps
	cas := map[string][]byte{
		"corps vide":                 nil,
		"tableau JSON":               []byte(`[]`),
		"au-delà de 64 Kio":          append(bytes.Clone(valide), bytes.Repeat([]byte(" "), message.DefaultMaxSize)...),
		"demande de contrôle":        []byte(strings.Replace(string(valide), `"type": "publication.decidee"`, `"type": "controle.demande"`, 1)),
		"résultat de contrôle":       []byte(strings.Replace(string(valide), `"type": "publication.decidee"`, `"type": "controle.resultat"`, 1)),
		"issue en majuscules":        []byte(strings.Replace(string(valide), `"issue": "`, `"issue": "X`, 1)),
		"texte saisi par le vendeur": []byte(strings.Replace(string(valide), `"revision":`, `"description": "Appelez-moi au 01 99 00 12 34", "revision":`, 1)),
	}
	for nom, corps := range cas {
		t.Run(nom, func(t *testing.T) {
			enregistreur := nouvelEnregistreur()
			disposition, err := livrer(t, traitant(t, enregistreur, io.Discard), corps)
			if err != nil || disposition != broker.Reject {
				t.Fatalf("disposition %v, erreur %v ; mise à l'écart attendue", disposition, err)
			}
			if n := len(enregistreur.recues()); n != 0 {
				t.Fatalf("%d notification(s) créée(s)", n)
			}
		})
	}
}

func TestUS028AC5_MessageInvalideNeBloquePasLeSuivant(t *testing.T) {
	enregistreur := nouvelEnregistreur()
	h := traitant(t, enregistreur, io.Discard)
	for _, ex := range exemplesDe(t, "invalides") {
		if _, err := livrer(t, h, ex.corps); err != nil {
			t.Fatalf("%s : %v", ex.nom, err)
		}
	}
	disposition, err := livrer(t, h, exemplesDe(t, "valides")[0].corps)
	if err != nil || disposition != broker.Ack || enregistreur.creees() != 1 {
		t.Fatalf("décision suivante : disposition %v, erreur %v, %d notification(s)", disposition, err, enregistreur.creees())
	}
}

func TestUS028AC2_BaseIndisponibleLaDecisionNEstPasEcartee(t *testing.T) {
	enregistreur := nouvelEnregistreur()
	enregistreur.panne = errBaseIndisponible
	disposition, err := livrer(t, traitant(t, enregistreur, io.Discard), exemplesDe(t, "valides")[0].corps)
	if err == nil || disposition == broker.Reject {
		t.Fatalf("panne de base : disposition %v, erreur %v ; nouvel essai attendu, sans mise à l'écart", disposition, err)
	}
}

func TestUS028AC4_DecisionEnregistreeNePorteQueDesCodes(t *testing.T) {
	enregistreur := nouvelEnregistreur()
	h := traitant(t, enregistreur, io.Discard)
	for _, ex := range exemplesDe(t, "valides") {
		if _, err := livrer(t, h, ex.corps); err != nil {
			t.Fatalf("%s : %v", ex.nom, err)
		}
	}
	for _, d := range enregistreur.recues() {
		if !slices.Contains(issuesFermees, string(d.Issue)) {
			t.Fatalf("issue %q hors liste fermée", d.Issue)
		}
		if motif := motifDe(d); motif != nil && !slices.Contains(motifsFermes, *motif) {
			t.Fatalf("motif %q hors liste fermée", *motif)
		}
	}
}
