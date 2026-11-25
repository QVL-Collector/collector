package outbox_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"collector/libs/platform/logging"
	"collector/libs/platform/message"
	"collector/libs/platform/outbox"
)

const (
	traceDeLaRequete       = "4bf92f3577b34da6a3ce929d0e0e4736"
	traceparentDeLaRequete = "00-" + traceDeLaRequete + "-00f067aa0ba902b7-01"
	correlationDeLaRequete = "8d1f6a2e-3b4c-4d5e-9f60-718293a4b5c6"
)

var traceparentW3C = regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`)

func enveloppe(correlation string) message.Envelope {
	return message.Envelope{ID: "6f1c0000-0000-4000-8000-000000000001", Type: "controle.demande", Version: 1, EmittedAt: time.Now().UTC(), CorrelationID: correlation, Data: []byte(`{}`)}
}

func TestUS014AC1EvenementDOutboxGardeLaTraceDeLaRequete(t *testing.T) {
	ctx := logging.ContinueTrace(logging.WithReceivedCorrelationID(context.Background(), correlationDeLaRequete), traceparentDeLaRequete)
	attendu, _ := logging.TraceParent(ctx)

	evenement := outbox.NewEvent(ctx, enveloppe(correlationDeLaRequete), "controle.demandes", "v1", []byte(`{}`))

	if evenement.TraceParent != attendu {
		t.Errorf("traceparent de l'événement %q, %q attendu", evenement.TraceParent, attendu)
	}
	if !traceparentW3C.MatchString(evenement.TraceParent) || evenement.TraceParent[3:35] != traceDeLaRequete {
		t.Errorf("traceparent %q sans la trace %s", evenement.TraceParent, traceDeLaRequete)
	}
	if evenement.CorrelationID != correlationDeLaRequete {
		t.Errorf("correlation_id de l'événement %q, %q attendu", evenement.CorrelationID, correlationDeLaRequete)
	}
}

func TestUS014AC1EvenementDOutboxHorsTraceResteConformeAuW3COuVide(t *testing.T) {
	evenement := outbox.NewEvent(context.Background(), enveloppe(correlationDeLaRequete), "controle.demandes", "v1", []byte(`{}`))

	if evenement.TraceParent != "" && !traceparentW3C.MatchString(evenement.TraceParent) {
		t.Errorf("traceparent %q hors du format W3C (refusé par la contrainte de la colonne)", evenement.TraceParent)
	}
	if evenement.CorrelationID != correlationDeLaRequete {
		t.Errorf("correlation_id de l'événement %q, %q attendu", evenement.CorrelationID, correlationDeLaRequete)
	}
}

func TestUS014AC1EvenementsDUneMemeRequetePartagentLaTrace(t *testing.T) {
	ctx := logging.ContinueTrace(logging.WithReceivedCorrelationID(context.Background(), correlationDeLaRequete), traceparentDeLaRequete)

	premier := outbox.NewEvent(ctx, enveloppe(correlationDeLaRequete), "controle.demandes", "v1", []byte(`{}`))
	second := outbox.NewEvent(ctx, enveloppe(correlationDeLaRequete), "pdm.evenements", "publication.decidee.v1", []byte(`{}`))

	if premier.TraceParent != second.TraceParent {
		t.Errorf("deux événements d'une même requête : %q et %q", premier.TraceParent, second.TraceParent)
	}
}
