package broker

import (
	"context"
	"slices"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"collector/libs/platform/metrics"
)

const (
	meterScope  = "collector/libs/platform/broker"
	messageUnit = "{message}"
	exchangeKey = "echange"
	queueKey    = "file"
	causeKey    = "motif"
)

type channelInstruments struct {
	published metric.Int64Counter
	consumed  metric.Int64Counter
	discarded metric.Int64Counter
	retried   metric.Int64Counter
}

var instruments = newChannelInstruments()

func newChannelInstruments() channelInstruments {
	meter := otel.Meter(meterScope)
	return channelInstruments{
		published: metrics.Counter(meter, "canal.messages.publies", messageUnit, "Messages confirmés et routés par le broker, par échange"),
		consumed:  metrics.Counter(meter, "canal.messages.consommes", messageUnit, "Messages traités puis acquittés, par file"),
		discarded: metrics.Counter(meter, "canal.messages.ecartes", messageUnit, "Messages mis à l'écart par le consommateur, par file et motif"),
		retried:   metrics.Counter(meter, "canal.messages.relances", messageUnit, "Nouveaux essais de traitement après un échec transitoire, par file"),
	}
}

func recordPublished(ctx context.Context, batch []Message, unroutable []string) {
	for _, message := range batch {
		if !slices.Contains(unroutable, message.ID) {
			instruments.published.Add(ctx, 1, metric.WithAttributes(attribute.String(exchangeKey, message.Exchange)))
		}
	}
}

func recordConsumed(ctx context.Context, queue string) {
	instruments.consumed.Add(ctx, 1, metric.WithAttributes(attribute.String(queueKey, queue)))
}

func recordDiscarded(ctx context.Context, queue string, cause discardCause) {
	instruments.discarded.Add(ctx, 1, metric.WithAttributes(
		attribute.String(queueKey, queue),
		attribute.String(causeKey, string(cause)),
	))
}

func recordRetried(ctx context.Context, queue string) {
	instruments.retried.Add(ctx, 1, metric.WithAttributes(attribute.String(queueKey, queue)))
}
