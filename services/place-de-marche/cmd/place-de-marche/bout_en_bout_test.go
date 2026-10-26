package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"collector/libs/platform/message"
	"collector/services/place-de-marche/internal/migrations"
)

const (
	delaiDePublication   = 10 * time.Second
	delaiDeTraitement    = 20 * time.Second
	categorieNonReconnue = "timbres"
)

type ligneOutbox struct {
	echange string
	cle     string
	typ     string
	corps   []byte
	publiee bool
}

type resultatEnregistre struct {
	revision int
	issue    string
	motif    *string
	version  int
}

type resultatCapture struct {
	livraison amqp.Delivery
	donnees   resultatEmis
	enveloppe message.Envelope
}

type bancComplet struct {
	*environnement
	canal      *canalDeTest
	base       basePostgreSQL
	journal    *journalPartage
	validateur *message.Validator
	jeton      string
	captures   map[string][]resultatCapture

	enAttente       articleRecu
	refuse          articleRecu
	corrige         articleRecu
	demandeUsurpee  string
	journalControle *journalPartage
}

func demarrerBancComplet(t *testing.T) *bancComplet {
	t.Helper()
	ctx := contexteBorne(t, 3*time.Minute)
	base := amorcer(ctx, t, demarrerPostgreSQL(t))
	if err := migrations.Apply(ctx, journalMuet(), base.migration); err != nil {
		t.Fatalf("migrations : %v", err)
	}
	canal := demarrerRabbitMQ(t)
	canal.poserUnEspion(t)
	e, journal := lancerPlaceDeMarche(t, canal, base.execution, nouvelEmetteurDeJetons(t))
	return &bancComplet{
		environnement: e,
		canal:         canal,
		base:          base,
		journal:       journal,
		validateur:    validateurDeContrats(t),
		jeton:         e.vendeur(),
		captures:      map[string][]resultatCapture{},
	}
}

func (b *bancComplet) attendreEtat(t *testing.T, id, etat string, delai time.Duration) (articleRecu, time.Duration) {
	t.Helper()
	var dernier articleRecu
	duree := attendre(t, delai, fmt.Sprintf("article %s %s", id, etat), func() bool {
		dernier = b.relire(t, b.jeton, id)
		return dernier.Etat == etat
	})
	return dernier, duree
}

func (b *bancComplet) outboxDe(t *testing.T, id string) []ligneOutbox {
	t.Helper()
	lignes, err := b.base.maitre.Query(t.Context(), `
		SELECT echange, cle_routage, type_message, corps, publie_le IS NOT NULL
		FROM outbox WHERE convert_from(corps, 'UTF8')::jsonb -> 'data' ->> 'article_id' = $1
		ORDER BY cree_le`, id)
	if err != nil {
		t.Fatalf("lecture de l'outbox : %v", err)
	}
	defer lignes.Close()
	var resultat []ligneOutbox
	for lignes.Next() {
		var l ligneOutbox
		if err := lignes.Scan(&l.echange, &l.cle, &l.typ, &l.corps, &l.publiee); err != nil {
			t.Fatalf("ligne d'outbox : %v", err)
		}
		resultat = append(resultat, l)
	}
	return resultat
}

func (b *bancComplet) resultatsEnregistres(t *testing.T, id string) []resultatEnregistre {
	t.Helper()
	lignes, err := b.base.maitre.Query(t.Context(), `
		SELECT revision, issue, motif, version_regles FROM resultat_controle WHERE article_id = $1 ORDER BY revision`, id)
	if err != nil {
		t.Fatalf("lecture des résultats : %v", err)
	}
	defer lignes.Close()
	var resultat []resultatEnregistre
	for lignes.Next() {
		var r resultatEnregistre
		if err := lignes.Scan(&r.revision, &r.issue, &r.motif, &r.version); err != nil {
			t.Fatalf("ligne de résultat : %v", err)
		}
		resultat = append(resultat, r)
	}
	return resultat
}

func (b *bancComplet) releverLesResultats(t *testing.T) {
	t.Helper()
	for _, livraison := range b.canal.releverLEspion(t) {
		var donnees resultatEmis
		enveloppe, err := b.validateur.Decode(livraison.Body, &donnees)
		if err != nil {
			t.Errorf("résultat publié par le Contrôle non conforme au contrat v1 : %v", err)
			continue
		}
		if livraison.UserId != identiteControle || livraison.RoutingKey != cleDeRoutageV1 {
			t.Errorf("résultat publié en %q avec la clé %q", livraison.UserId, livraison.RoutingKey)
		}
		b.captures[donnees.ArticleID] = append(b.captures[donnees.ArticleID], resultatCapture{livraison: livraison, donnees: donnees, enveloppe: enveloppe})
	}
}

func (b *bancComplet) capture(t *testing.T, id string, revision int) resultatCapture {
	t.Helper()
	b.releverLesResultats(t)
	for _, c := range b.captures[id] {
		if c.donnees.Revision == revision {
			return c
		}
	}
	t.Fatalf("aucun résultat capturé pour %s en révision %d", id, revision)
	return resultatCapture{}
}

func (b *bancComplet) creerEtSoumettre(t *testing.T, corps string) articleRecu {
	t.Helper()
	return b.soumettreAvecSucces(t, b.jeton, b.creer(t, b.jeton, corps))
}

func (b *bancComplet) exigerMessages(t *testing.T, file string, attendus int) {
	t.Helper()
	attendre(t, delaiDeTraitement, fmt.Sprintf("%d message(s) dans %s", attendus, file), func() bool {
		return b.canal.compter(t, file) == attendus
	})
}

func (b *bancComplet) exigerJournalise(t *testing.T, journal *journalPartage, id string) {
	t.Helper()
	attendre(t, 5*time.Second, "journalisation de "+id, func() bool { return journal.contient(id) })
}

func resultatForge(article articleRecu, issue string, motif *string) []byte {
	donnees, _ := json.Marshal(resultatEmis{ArticleID: article.ID, Revision: article.Revision, RulesVersion: 1, Issue: issue, Reason: motif})
	return []byte(fmt.Sprintf(`{"id":%q,"type":"controle.resultat","version":1,"emitted_at":"2026-10-26T09:00:00Z","correlation_id":%q,"data":%s}`,
		uuid.NewString(), uuid.NewString(), donnees))
}

func versLesResultats(emetteur string, corps []byte) publication {
	return publication{echange: echangeResultats, cle: cleDeRoutageV1, id: uuid.NewString(), emetteur: emetteur, corps: corps}
}

func TestUS013BoutEnBoutPlaceDeMarcheEtControle(t *testing.T) {
	b := demarrerBancComplet(t)
	binaire := construireControle(t)
	b.enAttente = b.creerEtSoumettre(t, corpsComplet("Figurine Tintin", categorieSure, 4500, 600))
	b.demandeUsurpee = uuid.NewString()
	t.Run("AC5 demande écrite dans l'outbox avec l'état en contrôle puis publiée sur le canal", b.ac5DemandeEcriteDansLOutboxPuisPubliee)
	t.Run("AC5 échec d'écriture dans l'outbox annule la soumission", b.ac5EchecDeLOutboxAnnuleLaSoumission)
	t.Run("AC7 résultat publié par une autre identité écarté et journalisé", b.ac7ResultatDUneAutreIdentiteEcarte)
	t.Run("AC7 user-id controle usurpé refusé par le broker", b.ac7UserIDUsurpeRefuseParLeBroker)
	t.Run("AC8 résultat non conforme au contrat écarté et journalisé", b.ac8ResultatNonConformeEcarte)
	t.Run("AC7 AC8 demandes usurpée ou non conforme publiées avant le démarrage du Contrôle", b.ac7ac8DemandesUsurpeeOuNonConformeEcartees)
	b.canal.releverLEspion(t)
	b.journalControle = lancerControle(t, binaire, b.canal)
	t.Run("AC1 demande en attente publiée dès le démarrage du Contrôle", b.ac1DemandeEnAttentePublieeAuDemarrageDuControle)
	t.Run("AC1 brouillon complet en contrôle puis publié en moins de 10 secondes", b.ac1PublieEnMoinsDe10Secondes)
	t.Run("AC1 brouillon complet publié dans chaque catégorie proposée", b.ac1PublieDansChaqueCategorie)
	t.Run("AC2 brouillon sans description refusé pour annonce incomplète", b.ac2SansDescriptionRefuse)
	t.Run("AC3 article refusé corrigé et resoumis repasse en contrôle puis publié", b.ac3RefuseCorrigeResoumisPublie)
	t.Run("AC4 catégorie inconnue du Contrôle à vérifier, jamais publiée", b.ac4CategorieInconnueAVerifier)
	t.Run("AC5 AC8 chaque résultat porte la version des règles v1 et respecte le contrat", b.ac5ac8ResultatsEnVersionDeReglesV1)
	t.Run("AC6 résultat livré deux fois ou de révision antérieure sans effet", b.ac6DoublonEtRevisionAnterieureSansEffet)
	t.Run("AC9 soumission par un autre vendeur introuvable", b.ac9AutreVendeurIntrouvable)
}

func (b *bancComplet) ac5DemandeEcriteDansLOutboxPuisPubliee(t *testing.T) {
	exigerEtat(t, b.relire(t, b.jeton, b.enAttente.ID), etatEnControle, "")
	lignes := b.outboxDe(t, b.enAttente.ID)
	if len(lignes) != 1 || lignes[0].echange != echangeDemandes || lignes[0].cle != cleDeRoutageV1 || lignes[0].typ != typeDemande {
		t.Fatalf("outbox de %s : %+v", b.enAttente.ID, lignes)
	}
	var demande demandeRecue
	if _, err := b.validateur.Decode(lignes[0].corps, &demande); err != nil {
		t.Fatalf("demande de l'outbox non conforme au contrat v1 : %v", err)
	}
	if demande.ArticleID != b.enAttente.ID || demande.Revision != b.enAttente.Revision {
		t.Fatalf("demande %+v pour l'article %s en révision %d", demande, b.enAttente.ID, b.enAttente.Revision)
	}
	attendre(t, delaiDeTraitement, "publication de la demande", func() bool { return b.outboxDe(t, b.enAttente.ID)[0].publiee })
	b.exigerMessages(t, fileDemandes, 1)
}

func (b *bancComplet) ac5EchecDeLOutboxAnnuleLaSoumission(t *testing.T) {
	article := b.creer(t, b.jeton, corpsComplet("Figurine Milou", categorieSure, 1000, 0))
	for _, instruction := range []string{
		`CREATE FUNCTION test_outbox_indisponible() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'outbox indisponible'; END $$`,
		`CREATE TRIGGER test_outbox_indisponible BEFORE INSERT ON outbox FOR EACH ROW EXECUTE FUNCTION test_outbox_indisponible()`,
	} {
		if _, err := b.base.maitre.Exec(t.Context(), instruction); err != nil {
			t.Fatalf("panne simulée de l'outbox : %v", err)
		}
	}
	statut := b.soumettre(t, b.jeton, article.ID, article.Revision).statut
	if _, err := b.base.maitre.Exec(t.Context(), `DROP TRIGGER test_outbox_indisponible ON outbox; DROP FUNCTION test_outbox_indisponible()`); err != nil {
		t.Fatalf("fin de la panne simulée : %v", err)
	}
	if statut < http.StatusInternalServerError {
		t.Fatalf("soumission acceptée (%d) alors que la demande n'a pas pu être écrite", statut)
	}
	relu := b.relire(t, b.jeton, article.ID)
	exigerEtat(t, relu, etatBrouillon, "")
	if relu.Revision != article.Revision || len(b.outboxDe(t, article.ID)) != 0 {
		t.Fatalf("soumission partielle : révision %d, outbox %d ligne(s)", relu.Revision, len(b.outboxDe(t, article.ID)))
	}
}

func (b *bancComplet) ac7ResultatDUneAutreIdentiteEcarte(t *testing.T) {
	usurpation := versLesResultats(identiteAdmin, resultatForge(b.enAttente, issuePublie, nil))
	if err := b.canal.publier(t, identiteAdmin, usurpation); err != nil {
		t.Fatalf("publication sous sa propre identité refusée : %v", err)
	}
	b.exigerMessages(t, fileResultatsEcartes, 1)
	b.exigerJournalise(t, b.journal, usurpation.id)
	exigerEtat(t, b.relire(t, b.jeton, b.enAttente.ID), etatEnControle, "")
}

func (b *bancComplet) ac7UserIDUsurpeRefuseParLeBroker(t *testing.T) {
	err := b.canal.publier(t, identiteAdmin, versLesResultats(identiteControle, resultatForge(b.enAttente, issuePublie, nil)))
	if codeDeRefus(err) != refusParLeBroker {
		t.Fatalf("publication avec un user-id usurpé : %v, refus %d attendu", err, refusParLeBroker)
	}
	b.exigerMessages(t, fileResultatsEcartes, 1)
	exigerEtat(t, b.relire(t, b.jeton, b.enAttente.ID), etatEnControle, "")
}

func (b *bancComplet) ac8ResultatNonConformeEcarte(t *testing.T) {
	motif := motifIncomplet
	nonConformes := []publication{
		versLesResultats(identiteControle, resultatForge(b.enAttente, issuePublie, &motif)),
		versLesResultats(identiteControle, []byte(`{"id":"tronqué"`)),
		versLesResultats(identiteControle, []byte(fmt.Sprintf(`{"id":%q,"type":"controle.resultat","version":2,"emitted_at":"2026-10-26T09:00:00Z","correlation_id":%q,"data":{}}`, uuid.NewString(), uuid.NewString()))),
	}
	for i, p := range nonConformes {
		if err := b.canal.publier(t, identiteControle, p); err != nil {
			t.Fatalf("publication du message non conforme %d : %v", i, err)
		}
		b.exigerMessages(t, fileResultatsEcartes, 2+i)
		b.exigerJournalise(t, b.journal, p.id)
	}
	exigerEtat(t, b.relire(t, b.jeton, b.enAttente.ID), etatEnControle, "")
}

func (b *bancComplet) ac7ac8DemandesUsurpeeOuNonConformeEcartees(t *testing.T) {
	corps := b.outboxDe(t, b.enAttente.ID)[0].corps
	var demande map[string]any
	if err := json.Unmarshal(corps, &demande); err != nil {
		t.Fatalf("demande : %v", err)
	}
	demande["id"] = uuid.NewString()
	demande["data"].(map[string]any)["article_id"] = b.demandeUsurpee
	usurpee, _ := json.Marshal(demande)
	if err := b.canal.publier(t, identiteAdmin, publication{echange: echangeDemandes, cle: cleDeRoutageV1, id: uuid.NewString(), emetteur: identiteAdmin, corps: usurpee}); err != nil {
		t.Fatalf("demande usurpée : %v", err)
	}
	if err := b.canal.publier(t, identiteMarche, publication{echange: echangeDemandes, cle: cleDeRoutageV1, id: uuid.NewString(), emetteur: identiteMarche, corps: []byte(`{}`)}); err != nil {
		t.Fatalf("demande non conforme : %v", err)
	}
	b.exigerMessages(t, fileDemandes, 3)
}

func (b *bancComplet) ac1DemandeEnAttentePublieeAuDemarrageDuControle(t *testing.T) {
	b.attendreEtat(t, b.enAttente.ID, etatPublie, delaiDeTraitement)
	b.exigerMessages(t, fileDemandesEcartees, 2)
	b.releverLesResultats(t)
	if len(b.captures[b.demandeUsurpee]) != 0 {
		t.Fatal("le Contrôle a répondu à une demande usurpée")
	}
	if b.journalControle.contient("Figurine Tintin") {
		t.Fatal("le Contrôle journalise le contenu des demandes en clair")
	}
}

func (b *bancComplet) ac1PublieEnMoinsDe10Secondes(t *testing.T) {
	debut := time.Now()
	article := b.creerEtSoumettre(t, corpsComplet("Figurine Haddock", categorieSure, 3200, 500))
	exigerEtat(t, b.dansMesArticles(t, b.jeton, article.ID), etatEnControle, "")
	publie, _ := b.attendreEtat(t, article.ID, etatPublie, delaiDePublication)
	if duree := time.Since(debut); duree >= delaiDePublication {
		t.Fatalf("publié en %s", duree)
	}
	exigerEtat(t, publie, etatPublie, "")
	exigerEtat(t, b.dansMesArticles(t, b.jeton, article.ID), etatPublie, "")
}

func (b *bancComplet) ac1PublieDansChaqueCategorie(t *testing.T) {
	var soumis []articleRecu
	for _, c := range b.categories(t) {
		soumis = append(soumis, b.creerEtSoumettre(t, corpsComplet("Pièce "+c.Libelle, c.ID, 1500, 0)))
	}
	for _, a := range soumis {
		b.attendreEtat(t, a.ID, etatPublie, delaiDeTraitement)
	}
}

func (b *bancComplet) ac2SansDescriptionRefuse(t *testing.T) {
	soumis := b.creerEtSoumettre(t, corpsSansDescription(categorieSure))
	b.refuse, _ = b.attendreEtat(t, soumis.ID, etatRefuse, delaiDeTraitement)
	exigerEtat(t, b.refuse, etatRefuse, motifIncomplet)
	exigerEtat(t, b.dansMesArticles(t, b.jeton, soumis.ID), etatRefuse, motifIncomplet)
}

func (b *bancComplet) ac3RefuseCorrigeResoumisPublie(t *testing.T) {
	var modifie articleRecu
	b.modifier(t, b.jeton, b.refuse.ID, corpsRevision(b.refuse.Revision, corpsComplet("Figurine Astérix", categorieSure, 2500, 0))).exiger(t, http.StatusOK).lire(t, &modifie)
	resoumis := b.soumettreAvecSucces(t, b.jeton, modifie)
	b.corrige, _ = b.attendreEtat(t, resoumis.ID, etatPublie, delaiDeTraitement)
	exigerEtat(t, b.corrige, etatPublie, "")
}

func (b *bancComplet) ac4CategorieInconnueAVerifier(t *testing.T) {
	if _, err := b.base.maitre.Exec(t.Context(), `INSERT INTO categorie (id, libelle) VALUES ($1, 'Timbres')`, categorieNonReconnue); err != nil {
		t.Fatalf("catégorie ajoutée côté Place de marché : %v", err)
	}
	soumis := b.creerEtSoumettre(t, corpsComplet("Timbre rare", categorieNonReconnue, 900, 0))
	aVerifier, _ := b.attendreEtat(t, soumis.ID, etatAVerifier, delaiDeTraitement)
	exigerEtat(t, aVerifier, etatAVerifier, motifCategorieInconnu)
	time.Sleep(2 * time.Second)
	exigerEtat(t, b.relire(t, b.jeton, soumis.ID), etatAVerifier, motifCategorieInconnu)
}

func (b *bancComplet) ac5ac8ResultatsEnVersionDeReglesV1(t *testing.T) {
	b.releverLesResultats(t)
	if len(b.captures) == 0 {
		t.Fatal("aucun résultat capturé")
	}
	for id, captures := range b.captures {
		for _, c := range captures {
			if c.donnees.RulesVersion != versionDesRegles {
				t.Errorf("résultat de %s en version de règles %d", id, c.donnees.RulesVersion)
			}
		}
		for _, r := range b.resultatsEnregistres(t, id) {
			if r.version != versionDesRegles {
				t.Errorf("résultat enregistré de %s en version de règles %d", id, r.version)
			}
		}
	}
	lignes := b.outboxDe(t, b.enAttente.ID)
	var demande message.Envelope
	if err := json.Unmarshal(lignes[0].corps, &demande); err != nil {
		t.Fatalf("demande : %v", err)
	}
	if c := b.capture(t, b.enAttente.ID, b.enAttente.Revision); c.enveloppe.CorrelationID != demande.CorrelationID {
		t.Errorf("corrélation du résultat %s, demande %s", c.enveloppe.CorrelationID, demande.CorrelationID)
	}
}

func (b *bancComplet) ac6DoublonEtRevisionAnterieureSansEffet(t *testing.T) {
	publieAvant := b.relire(t, b.jeton, b.enAttente.ID)
	corrigeAvant := b.relire(t, b.jeton, b.corrige.ID)
	doublon := b.capture(t, b.enAttente.ID, b.enAttente.Revision)
	perime := b.capture(t, b.refuse.ID, b.refuse.Revision)
	for _, c := range []resultatCapture{doublon, perime} {
		rejeu := publication{echange: echangeResultats, cle: cleDeRoutageV1, id: c.livraison.MessageId, emetteur: identiteControle, corps: c.livraison.Body}
		if err := b.canal.publier(t, identiteControle, rejeu); err != nil {
			t.Fatalf("rejeu de %s : %v", c.livraison.MessageId, err)
		}
	}
	temoin := b.creerEtSoumettre(t, corpsComplet("Figurine témoin", categorieSure, 100, 0))
	b.attendreEtat(t, temoin.ID, etatPublie, delaiDeTraitement)
	publieApres, corrigeApres := b.relire(t, b.jeton, b.enAttente.ID), b.relire(t, b.jeton, b.corrige.ID)
	if publieApres.Etat != publieAvant.Etat || publieApres.ModifieLe != publieAvant.ModifieLe || publieApres.Revision != publieAvant.Revision {
		t.Errorf("doublon avec effet : %+v devenu %+v", publieAvant, publieApres)
	}
	if corrigeApres.Etat != etatPublie || corrigeApres.Motif != nil || corrigeApres.ModifieLe != corrigeAvant.ModifieLe {
		t.Errorf("résultat périmé avec effet : %+v devenu %+v", corrigeAvant, corrigeApres)
	}
	if n := len(b.resultatsEnregistres(t, b.enAttente.ID)); n != 1 {
		t.Errorf("%d résultat(s) enregistré(s) pour %s, 1 attendu", n, b.enAttente.ID)
	}
	if n := b.canal.compter(t, fileResultatsEcartes); n != 4 {
		t.Errorf("%d message(s) écarté(s), 4 attendus : un doublon n'est pas une faute", n)
	}
}

func (b *bancComplet) ac9AutreVendeurIntrouvable(t *testing.T) {
	article := b.creer(t, b.jeton, corpsComplet("Figurine Dupond", categorieSure, 100, 0))
	b.soumettre(t, b.vendeur(), article.ID, article.Revision).probleme(t, http.StatusNotFound)
	exigerEtat(t, b.relire(t, b.jeton, article.ID), etatBrouillon, "")
	if len(b.outboxDe(t, article.ID)) != 0 {
		t.Fatal("la tentative d'un autre vendeur a écrit une demande")
	}
}
