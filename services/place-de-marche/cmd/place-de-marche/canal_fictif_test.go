package main

import (
	"bytes"
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"collector/libs/platform/broker"
	"collector/libs/platform/message"
	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/annonces/controle"
)

const (
	identiteControle      = "controle"
	echangeDemandes       = "controle.demandes"
	echangeResultats      = "controle.resultats"
	cleDeRoutageV1        = "v1"
	typeDemande           = "controle.demande"
	typeResultat          = "controle.resultat"
	issuePublie           = "publie"
	issueRefuse           = "refuse"
	issueAVerifier        = "a_verifier"
	motifIncomplet        = "annonce_incomplete"
	motifCategorieInconnu = "categorie_inconnue"
	versionDesRegles      = 1
)

var categoriesConnuesDuControle = []string{"figurines", "timbres"}

type journalPartage struct {
	mu     sync.Mutex
	tampon bytes.Buffer
}

func (j *journalPartage) Write(octets []byte) (int, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.tampon.Write(octets)
}

func (j *journalPartage) contient(fragment string) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return strings.Contains(j.tampon.String(), fragment)
}

type annonceDemandee struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	CategoryCode *string `json:"category_code"`
	PriceCents   *int64  `json:"price_cents"`
	PhotoCount   int     `json:"photo_count"`
}

type demandeRecue struct {
	ArticleID   string          `json:"article_id"`
	Revision    int             `json:"revision"`
	SubmittedAt string          `json:"submitted_at"`
	Listing     annonceDemandee `json:"listing"`
}

type resultatEmis struct {
	ArticleID    string  `json:"article_id"`
	Revision     int     `json:"revision"`
	RulesVersion int     `json:"rules_version"`
	Issue        string  `json:"issue"`
	Reason       *string `json:"reason,omitempty"`
}

type livraison struct {
	broker.Delivery
	disposition broker.Disposition
}

type bancEnMemoire struct {
	*environnement
	stockage   *stockageEnMemoire
	validateur *message.Validator
	encodeur   *controle.RequestEncoder
	resultats  *controle.ResultHandler
	journal    *journalPartage
}

func demarrerBancEnMemoire(t *testing.T) *bancEnMemoire {
	t.Helper()
	stockage := nouveauStockageEnMemoire(jeuDeCategoriesFictif)
	service := annonces.NewService(stockage, stockage)
	validateur := validateurDeContrats(t)
	journal := &journalPartage{}
	return &bancEnMemoire{
		environnement: demarrer(t, service),
		stockage:      stockage,
		validateur:    validateur,
		encodeur:      controle.NewRequestEncoder(validateur),
		resultats:     controle.NewResultHandler(validateur, service, slog.New(slog.NewJSONHandler(journal, nil))),
		journal:       journal,
	}
}

func verdictAttendu(annonce annonceDemandee) (string, *string) {
	incomplete := strings.TrimSpace(annonce.Title) == "" || strings.TrimSpace(annonce.Description) == "" ||
		annonce.CategoryCode == nil || strings.TrimSpace(*annonce.CategoryCode) == "" || annonce.PriceCents == nil
	if incomplete {
		motif := motifIncomplet
		return issueRefuse, &motif
	}
	if !slices.Contains(categoriesConnuesDuControle, *annonce.CategoryCode) {
		motif := motifCategorieInconnu
		return issueAVerifier, &motif
	}
	return issuePublie, nil
}

func (b *bancEnMemoire) emettreLesDemandes(t *testing.T) []demandeRecue {
	t.Helper()
	var recues []demandeRecue
	for _, demande := range b.stockage.retirerDemandes() {
		evenement, err := b.encodeur.Encode(t.Context(), demande)
		if err != nil {
			t.Fatalf("demande de contrôle non encodable : %v", err)
		}
		if evenement.Exchange != echangeDemandes || evenement.RoutingKey != cleDeRoutageV1 || evenement.Type != typeDemande {
			t.Fatalf("demande routée vers %q/%q (type %q), attendu %q/%q (type %q)",
				evenement.Exchange, evenement.RoutingKey, evenement.Type, echangeDemandes, cleDeRoutageV1, typeDemande)
		}
		var recue demandeRecue
		enveloppe, err := b.validateur.Decode(evenement.Body, &recue)
		if err != nil {
			t.Fatalf("demande non conforme au contrat v1 : %v", err)
		}
		if enveloppe.ID != evenement.ID || enveloppe.CorrelationID != evenement.CorrelationID {
			t.Fatalf("identifiants de l'outbox (%s, %s) différents de ceux du message (%s, %s)",
				evenement.ID, evenement.CorrelationID, enveloppe.ID, enveloppe.CorrelationID)
		}
		recues = append(recues, recue)
	}
	return recues
}

func (b *bancEnMemoire) resultatPour(t *testing.T, demande demandeRecue) broker.Delivery {
	t.Helper()
	issue, motif := verdictAttendu(demande.Listing)
	return b.resultatSigne(t, resultatEmis{
		ArticleID:    demande.ArticleID,
		Revision:     demande.Revision,
		RulesVersion: versionDesRegles,
		Issue:        issue,
		Reason:       motif,
	})
}

func (b *bancEnMemoire) resultatSigne(t *testing.T, resultat resultatEmis) broker.Delivery {
	t.Helper()
	enveloppe, err := message.NewEnvelope(message.Contract{Type: typeResultat, Version: 1}, "0b6f8a4e-6f43-4a0e-9a3c-5b2f7f0c1d2e", resultat, time.Now())
	if err != nil {
		t.Fatalf("enveloppe du résultat : %v", err)
	}
	corps, err := b.validateur.Encode(enveloppe)
	if err != nil {
		t.Fatalf("résultat non conforme au contrat v1 : %v", err)
	}
	return broker.Delivery{MessageID: enveloppe.ID, UserID: identiteControle, Body: corps}
}

func (b *bancEnMemoire) livrer(t *testing.T, delivery broker.Delivery) livraison {
	t.Helper()
	disposition, err := b.resultats.Handle(context.WithoutCancel(t.Context()), delivery)
	if err != nil {
		t.Fatalf("traitement du résultat %s : %v", delivery.MessageID, err)
	}
	return livraison{Delivery: delivery, disposition: disposition}
}

func (b *bancEnMemoire) controler(t *testing.T) []livraison {
	t.Helper()
	var livrees []livraison
	for _, demande := range b.emettreLesDemandes(t) {
		livrees = append(livrees, b.livrer(t, b.resultatPour(t, demande)))
	}
	return livrees
}
