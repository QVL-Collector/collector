package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/libs/platform/message"
)

const delaiDeSignalement = 45 * time.Second

type vendeurDemande struct {
	AccountID   string `json:"account_id"`
	MemberSince string `json:"member_since"`
	Suspended   bool   `json:"suspended"`
}

type donneesDeDemande struct {
	ArticleID   string          `json:"article_id"`
	Revision    int             `json:"revision"`
	SubmittedAt string          `json:"submitted_at"`
	Seller      vendeurDemande  `json:"seller"`
	Listing     annonceDemandee `json:"listing"`
}

func demandeConforme(t *testing.T) []byte {
	t.Helper()
	categorie, prix := categorieSure, int64(1500)
	donnees := donneesDeDemande{
		ArticleID:   uuid.NewString(),
		Revision:    2,
		SubmittedAt: "2026-10-26T09:00:00Z",
		Seller:      vendeurDemande{AccountID: uuid.NewString(), MemberSince: "2025-01-15"},
		Listing:     annonceDemandee{Title: "Figurine", Description: "Belle", CategoryCode: &categorie, PriceCents: &prix},
	}
	validateur := validateurDeContrats(t)
	enveloppe, err := message.NewEnvelope(message.Contract{Type: typeDemande, Version: 1}, uuid.NewString(), donnees, time.Now())
	if err != nil {
		t.Fatalf("enveloppe de la demande : %v", err)
	}
	corps, err := validateur.Encode(enveloppe)
	if err != nil {
		t.Fatalf("demande non conforme : %v", err)
	}
	return corps
}

func signaleSaPanne(t *testing.T, p *processusControle) string {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	disponibleUneFois := false
	limite := time.Now().Add(delaiDeSignalement)
	for time.Now().Before(limite) {
		select {
		case err := <-p.termine:
			if err == nil {
				t.Fatal("le Contrôle s'est arrêté sans erreur")
			}
			return "arrêt en erreur : " + err.Error()
		default:
		}
		disponible := p.disponible(client)
		if disponibleUneFois && !disponible {
			return "non disponible"
		}
		disponibleUneFois = disponibleUneFois || disponible
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("le Contrôle ni arrêté ni signalé indisponible en %s malgré une publication refusée", delaiDeSignalement)
	return ""
}

func TestUS013ControleSansDroitDePublicationSArreteOuSeSignaleIndisponible(t *testing.T) {
	canal := demarrerRabbitMQ(t, sansDroitDEcriture(identiteControle))
	binaire := construireControle(t)
	demande := publication{echange: echangeDemandes, cle: cleDeRoutageV1, id: uuid.NewString(), emetteur: identiteMarche, corps: demandeConforme(t)}
	if err := canal.publier(t, identiteMarche, demande); err != nil {
		t.Fatalf("publication de la demande : %v", err)
	}
	p := demarrerProcessusControle(t, binaire, canal)
	t.Logf("Contrôle : %s", signaleSaPanne(t, p))
	p.arreter()
	attendre(t, delaiDeTraitement, "demande conservée sans acquittement", func() bool {
		return canal.compter(t, fileDemandes) == 1 && canal.compter(t, fileDemandesEcartees) == 0
	})
}
