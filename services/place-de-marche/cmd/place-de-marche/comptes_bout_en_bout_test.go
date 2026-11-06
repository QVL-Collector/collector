package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const (
	commandeSuspendre      = "suspendre"
	commandeRetablir       = "retablir"
	evenementSuspension    = "comptes.suspension"
	motifCompteNonAutorise = "compte_non_autorise"
	prixSousLaVigilance    = 4500
	prixAuDessusVigilance  = 30_001
	ancienneteEtablie      = 40 * 24 * time.Hour
)

var identiteCivileDuVendeur = []string{"Jeanne", "Collectionneuse", "jeanne.collectionneuse@exemple.fr", "jcollec"}

type vendeurDuBanc struct {
	sujet string
	jeton string
}

type vendeurTransmis struct {
	AccountID   string `json:"account_id"`
	MemberSince string `json:"member_since"`
	Suspended   bool   `json:"suspended"`
}

type demandeAvecVendeur struct {
	Revision int             `json:"revision"`
	Seller   json.RawMessage `json:"seller"`
}

type sortieDeCommande struct {
	code   int
	sortie string
}

type bancDesComptes struct {
	*bancComplet
	commande     string
	enRegle      vendeurDuBanc
	publie       articleRecu
	refuse       articleRecu
	debut        time.Time
	suspenduDate time.Time
}

func avecIdentiteCivile(_, charge map[string]any) {
	charge["email"] = "jeanne.collectionneuse@exemple.fr"
	charge["name"] = "Jeanne Collectionneuse"
	charge["given_name"] = "Jeanne"
	charge["family_name"] = "Collectionneuse"
	charge["preferred_username"] = "jcollec"
}

func vendeurAuSujet(e *environnement, sujet string) vendeurDuBanc {
	return vendeurDuBanc{sujet: sujet, jeton: bearer(e.jetons.emettre(sujet, []string{roleVendeur}, avecIdentiteCivile))}
}

func nouveauVendeurIdentifie(e *environnement) vendeurDuBanc {
	return vendeurAuSujet(e, uuid.NewString())
}

func construirePlaceDeMarche(t *testing.T) string {
	t.Helper()
	repertoire := t.TempDir()
	binaire := filepath.Join(repertoire, serviceName)
	if runtime.GOOS == "windows" {
		binaire += ".exe"
	}
	construction := exec.Command("go", "install", ".")
	construction.Env = append(os.Environ(), "GOBIN="+repertoire)
	if sortie, err := construction.CombinedOutput(); err != nil {
		t.Fatalf("construction de la Place de marché : %v\n%s", err, sortie)
	}
	return binaire
}

func jourUTC(instant time.Time) string {
	return instant.UTC().Format(time.DateOnly)
}

func (b *bancComplet) vendeurTransmis(t *testing.T, article articleRecu) vendeurTransmis {
	t.Helper()
	lignes := b.outboxDe(t, article.ID)
	if len(lignes) == 0 {
		t.Fatalf("aucune demande en outbox pour %s", article.ID)
	}
	corps := lignes[len(lignes)-1].corps
	var demande demandeAvecVendeur
	if _, err := b.validateur.Decode(corps, &demande); err != nil {
		t.Fatalf("demande de l'outbox non conforme au contrat v1 : %v", err)
	}
	if demande.Revision != article.Revision {
		t.Fatalf("dernière demande en révision %d, %d attendue", demande.Revision, article.Revision)
	}
	var attributs map[string]json.RawMessage
	if err := json.Unmarshal(demande.Seller, &attributs); err != nil {
		t.Fatalf("vendeur de la demande : %v", err)
	}
	if len(attributs) != 3 || attributs["account_id"] == nil || attributs["member_since"] == nil || attributs["suspended"] == nil {
		t.Fatalf("attributs du vendeur transmis au Contrôle : %s", demande.Seller)
	}
	for _, revele := range identiteCivileDuVendeur {
		if strings.Contains(string(corps), revele) {
			t.Errorf("la demande révèle %q : %s", revele, corps)
		}
	}
	var vendeur vendeurTransmis
	if err := json.Unmarshal(demande.Seller, &vendeur); err != nil {
		t.Fatalf("vendeur de la demande : %v", err)
	}
	return vendeur
}

func exigerVendeurTransmis(t *testing.T, recu, attendu vendeurTransmis) {
	t.Helper()
	if recu != attendu {
		t.Fatalf("vendeur transmis %+v, attendu %+v", recu, attendu)
	}
}

func (b *bancDesComptes) exigerInscritDepuisLeDebut(t *testing.T, article articleRecu, v vendeurDuBanc, suspendu bool) {
	t.Helper()
	recu := b.vendeurTransmis(t, article)
	if recu.MemberSince < jourUTC(b.debut) || recu.MemberSince > jourUTC(time.Now()) {
		t.Fatalf("inscription transmise %q, attendue entre %s et %s (UTC)", recu.MemberSince, jourUTC(b.debut), jourUTC(time.Now()))
	}
	exigerVendeurTransmis(t, recu, vendeurTransmis{AccountID: v.sujet, MemberSince: recu.MemberSince, Suspended: suspendu})
}

func lancerUneCommande(t *testing.T, binaire, adresseBase string, arguments []string) sortieDeCommande {
	t.Helper()
	commande := exec.Command(binaire)
	commande.Args = append(commande.Args, arguments...)
	commande.Env = append(os.Environ(), "BASE_DE_DONNEES_URL="+adresseBase, "BASE_DE_DONNEES_TLS_EXIGE=false")
	sortie, err := commande.CombinedOutput()
	var fin *exec.ExitError
	if err != nil && !errors.As(err, &fin) {
		t.Fatalf("lancement de %v : %v", arguments, err)
	}
	return sortieDeCommande{code: commande.ProcessState.ExitCode(), sortie: string(sortie)}
}

func (b *bancDesComptes) commander(t *testing.T, arguments ...string) sortieDeCommande {
	t.Helper()
	return lancerUneCommande(t, b.commande, b.base.migration, arguments)
}

func (b *bancDesComptes) exigerCommandeReussie(t *testing.T, commande, sujet string) {
	t.Helper()
	s := b.commander(t, commande, sujet)
	if s.code != 0 {
		t.Fatalf("%s %s : code %d\n%s", commande, sujet, s.code, s.sortie)
	}
	if !strings.Contains(s.sortie, evenementSuspension) || !strings.Contains(s.sortie, sujet) {
		t.Fatalf("%s %s : journal sans %s pour le compte\n%s", commande, sujet, evenementSuspension, s.sortie)
	}
}

func (b *bancDesComptes) dateDeSuspension(t *testing.T, sujet string) time.Time {
	t.Helper()
	var date *time.Time
	if err := b.base.maitre.QueryRow(t.Context(), `SELECT suspendu_le FROM compte WHERE id = $1`, sujet).Scan(&date); err != nil {
		t.Fatalf("compte %s : %v", sujet, err)
	}
	if date == nil {
		return time.Time{}
	}
	return *date
}

func (b *bancDesComptes) comptesAuSujet(t *testing.T, sujet string) int {
	t.Helper()
	var nombre int
	if err := b.base.maitre.QueryRow(t.Context(), `SELECT count(*) FROM compte WHERE id::text = $1`, sujet).Scan(&nombre); err != nil {
		t.Fatalf("comptage des comptes : %v", err)
	}
	return nombre
}

func (b *bancDesComptes) soumettrePour(t *testing.T, v vendeurDuBanc, titre string, prix int64) articleRecu {
	t.Helper()
	return b.soumettreAvecSucces(t, v.jeton, b.creer(t, v.jeton, corpsComplet(titre, categorieSure, prix, 0)))
}

func (b *bancDesComptes) attendreEtatDe(t *testing.T, v vendeurDuBanc, id, etat, motif string) articleRecu {
	t.Helper()
	var dernier articleRecu
	attendre(t, delaiDeTraitement, "article "+id+" "+etat, func() bool {
		dernier = b.relire(t, v.jeton, id)
		return dernier.Etat == etat
	})
	exigerEtat(t, dernier, etat, motif)
	exigerEtat(t, b.dansMesArticles(t, v.jeton, id), etat, motif)
	return dernier
}

func (b *bancDesComptes) modifierPour(t *testing.T, v vendeurDuBanc, article articleRecu, titre string) articleRecu {
	t.Helper()
	var modifie articleRecu
	b.modifier(t, v.jeton, article.ID, corpsRevision(article.Revision, corpsComplet(titre, categorieSure, prixSousLaVigilance, 0))).exiger(t, http.StatusOK).lire(t, &modifie)
	return modifie
}

func (b *bancDesComptes) revaliderPour(t *testing.T, v vendeurDuBanc, publie articleRecu, titre string) articleRecu {
	t.Helper()
	modifie := b.modifierPour(t, v, publie, titre)
	exigerNouvelleRevisionEnControle(t, modifie, publie)
	return modifie
}

func TestUS019Lot2ComptesSurPostgreSQLRabbitMQEtControle(t *testing.T) {
	b := &bancDesComptes{bancComplet: demarrerBancComplet(t), commande: construirePlaceDeMarche(t), debut: time.Now()}
	b.journalControle = lancerControle(t, construireControle(t), b.canal)
	t.Run("AC4 vendeur neuf : demande portant l'identifiant opaque, inscrit le jour même en UTC, non suspendu, sans email ni nom", b.ac4VendeurNeufInscritCeJourNonSuspendu)
	t.Run("AC2 brouillon puis vendeur suspendu par l'exploitation : soumission refusée pour compte non autorisé", b.ac2BrouillonPuisSuspensionSoumissionRefusee)
	t.Run("AC2 US-021 vendeur suspendu : revalidation de son article publié refusée pour compte non autorisé", b.ac2SuspenduRevalidationRefusee)
	t.Run("AC2 suspension rejouée : sans erreur, date de suspension conservée, soumission toujours refusée", b.ac2SuspensionIdempotente)
	t.Run("AC2 AC5 vendeur rétabli deux fois : soumission et resoumission après correction non suspendues, publiées", b.ac2RetablissementIdempotentPublie)
	t.Run("AC2 compte jamais suspendu rétabli : sans erreur, soumission publiée", b.ac2RetablirUnCompteEnRegle)
	t.Run("AC2 identifiant invalide, compte inconnu, argument absent ou en trop : arrêt en erreur sans compte créé", b.ac2CommandesEnErreur)
	t.Run("AC3 vendeur inscrit le jour même au-dessus du seuil de vigilance : à vérifier", b.ac3VendeurNeufAuDessusDeLaVigilanceAVerifier)
	t.Run("AC3 vendeur inscrit depuis 40 jours au-dessus du seuil de vigilance : demande à sa date d'inscription, publié", b.ac3VendeurEtabliAuDessusDeLaVigilancePublie)
}

func (b *bancDesComptes) ac4VendeurNeufInscritCeJourNonSuspendu(t *testing.T) {
	b.enRegle = nouveauVendeurIdentifie(b.environnement)
	soumis := b.soumettrePour(t, b.enRegle, "Figurine Tintin en règle", prixSousLaVigilance)
	b.exigerInscritDepuisLeDebut(t, soumis, b.enRegle, false)
	b.publie = b.attendreEtatDe(t, b.enRegle, soumis.ID, etatPublie, "")
}

func (b *bancDesComptes) ac2BrouillonPuisSuspensionSoumissionRefusee(t *testing.T) {
	v := nouveauVendeurIdentifie(b.environnement)
	brouillon := b.creer(t, v.jeton, corpsComplet("Figurine Milou d'un vendeur suspendu", categorieSure, prixSousLaVigilance, 0))
	b.exigerCommandeReussie(t, commandeSuspendre, v.sujet)
	soumis := b.soumettreAvecSucces(t, v.jeton, brouillon)
	b.exigerInscritDepuisLeDebut(t, soumis, v, true)
	b.attendreEtatDe(t, v, soumis.ID, etatRefuse, motifCompteNonAutorise)
}

func (b *bancDesComptes) ac2SuspenduRevalidationRefusee(t *testing.T) {
	b.exigerCommandeReussie(t, commandeSuspendre, b.enRegle.sujet)
	b.suspenduDate = b.dateDeSuspension(t, b.enRegle.sujet)
	modifie := b.revaliderPour(t, b.enRegle, b.publie, "Figurine Tintin modifiée pendant la suspension")
	b.exigerInscritDepuisLeDebut(t, modifie, b.enRegle, true)
	b.refuse = b.attendreEtatDe(t, b.enRegle, modifie.ID, etatRefuse, motifCompteNonAutorise)
	b.exigerHorsCatalogue(t, b.refuse)
}

func (b *bancDesComptes) ac2SuspensionIdempotente(t *testing.T) {
	b.exigerCommandeReussie(t, commandeSuspendre, b.enRegle.sujet)
	if date := b.dateDeSuspension(t, b.enRegle.sujet); b.suspenduDate.IsZero() || !date.Equal(b.suspenduDate) {
		t.Fatalf("date de suspension %v après la seconde suspension, %v attendue", date, b.suspenduDate)
	}
	soumis := b.soumettrePour(t, b.enRegle, "Figurine Haddock pendant la suspension", prixSousLaVigilance)
	b.exigerInscritDepuisLeDebut(t, soumis, b.enRegle, true)
	b.attendreEtatDe(t, b.enRegle, soumis.ID, etatRefuse, motifCompteNonAutorise)
}

func (b *bancDesComptes) ac2RetablissementIdempotentPublie(t *testing.T) {
	b.exigerCommandeReussie(t, commandeRetablir, b.enRegle.sujet)
	b.exigerCommandeReussie(t, commandeRetablir, b.enRegle.sujet)
	soumis := b.soumettrePour(t, b.enRegle, "Figurine Tournesol après rétablissement", prixSousLaVigilance)
	b.exigerInscritDepuisLeDebut(t, soumis, b.enRegle, false)
	b.attendreEtatDe(t, b.enRegle, soumis.ID, etatPublie, "")
	resoumis := b.soumettreAvecSucces(t, b.enRegle.jeton, b.modifierPour(t, b.enRegle, b.refuse, "Figurine Tintin corrigée après rétablissement"))
	b.exigerInscritDepuisLeDebut(t, resoumis, b.enRegle, false)
	b.attendreEtatDe(t, b.enRegle, resoumis.ID, etatPublie, "")
}

func (b *bancDesComptes) ac2RetablirUnCompteEnRegle(t *testing.T) {
	v := nouveauVendeurIdentifie(b.environnement)
	brouillon := b.creer(t, v.jeton, corpsComplet("Figurine Dupont jamais suspendu", categorieSure, prixSousLaVigilance, 0))
	b.exigerCommandeReussie(t, commandeRetablir, v.sujet)
	soumis := b.soumettreAvecSucces(t, v.jeton, brouillon)
	b.exigerInscritDepuisLeDebut(t, soumis, v, false)
	b.attendreEtatDe(t, v, soumis.ID, etatPublie, "")
}

func (b *bancDesComptes) ac2CommandesEnErreur(t *testing.T) {
	inconnu := uuid.NewString()
	cas := map[string][]string{
		"suspendre un compte inconnu":                    {commandeSuspendre, inconnu},
		"rétablir un compte inconnu":                     {commandeRetablir, inconnu},
		"suspendre un identifiant non UUID":              {commandeSuspendre, "pas-un-identifiant"},
		"rétablir un identifiant non UUID":               {commandeRetablir, "42"},
		"suspendre un identifiant vide":                  {commandeSuspendre, ""},
		"suspendre un UUID tronqué":                      {commandeSuspendre, inconnu[:35]},
		"suspendre sans identifiant":                     {commandeSuspendre},
		"rétablir sans identifiant":                      {commandeRetablir},
		"suspendre deux identifiants":                    {commandeSuspendre, inconnu, uuid.NewString()},
		"suspendre un compte existant suivi d'un intrus": {commandeSuspendre, b.enRegle.sujet, "intrus"},
	}
	for nom, arguments := range cas {
		t.Run(nom, func(t *testing.T) {
			s := b.commander(t, arguments...)
			if s.code == 0 {
				t.Fatalf("%v accepté (code 0)\n%s", arguments, s.sortie)
			}
			if strings.Contains(s.sortie, evenementSuspension) {
				t.Fatalf("%v journalise une suspension\n%s", arguments, s.sortie)
			}
		})
	}
	if n := b.comptesAuSujet(t, inconnu); n != 0 {
		t.Fatalf("%d compte(s) créé(s) pour l'identifiant inconnu", n)
	}
	soumis := b.soumettrePour(t, b.enRegle, "Figurine Lampion après commandes en erreur", prixSousLaVigilance)
	b.exigerInscritDepuisLeDebut(t, soumis, b.enRegle, false)
	b.attendreEtatDe(t, b.enRegle, soumis.ID, etatPublie, "")
	v := vendeurAuSujet(b.environnement, inconnu)
	soumis = b.soumettrePour(t, v, "Figurine Séraphin d'un compte inconnu de l'exploitation", prixSousLaVigilance)
	b.exigerInscritDepuisLeDebut(t, soumis, v, false)
	b.attendreEtatDe(t, v, soumis.ID, etatPublie, "")
}

func (b *bancDesComptes) ac3VendeurNeufAuDessusDeLaVigilanceAVerifier(t *testing.T) {
	v := nouveauVendeurIdentifie(b.environnement)
	soumis := b.soumettrePour(t, v, "Figurine Tintin de collection d'un nouveau vendeur", prixAuDessusVigilance)
	b.exigerInscritDepuisLeDebut(t, soumis, v, false)
	b.attendreEtatDe(t, v, soumis.ID, etatAVerifier, motifVendeurAVerifier)
}

func (b *bancDesComptes) ac3VendeurEtabliAuDessusDeLaVigilancePublie(t *testing.T) {
	v := nouveauVendeurIdentifie(b.environnement)
	brouillon := b.creer(t, v.jeton, corpsComplet("Figurine Tintin de collection d'un vendeur établi", categorieSure, prixAuDessusVigilance, 0))
	inscription := time.Now().Add(-ancienneteEtablie)
	if _, err := b.base.maitre.Exec(t.Context(), `UPDATE compte SET inscrit_le = $2 WHERE id = $1`, v.sujet, inscription); err != nil {
		t.Fatalf("ancienneté du compte : %v", err)
	}
	soumis := b.soumettreAvecSucces(t, v.jeton, brouillon)
	exigerVendeurTransmis(t, b.vendeurTransmis(t, soumis), vendeurTransmis{AccountID: v.sujet, MemberSince: jourUTC(inscription), Suspended: false})
	b.attendreEtatDe(t, v, soumis.ID, etatPublie, "")
}
