package metrics

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

func Counter(meter metric.Meter, name, unit, description string) metric.Int64Counter {
	counter, err := meter.Int64Counter(name, metric.WithUnit(unit), metric.WithDescription(description))
	if err != nil {
		otel.Handle(err)
		return noop.Int64Counter{}
	}
	return counter
}

func Histogram(meter metric.Meter, name, unit, description string, boundaries []float64) metric.Float64Histogram {
	histogram, err := meter.Float64Histogram(name,
		metric.WithUnit(unit),
		metric.WithDescription(description),
		metric.WithExplicitBucketBoundaries(boundaries...),
	)
	if err != nil {
		otel.Handle(err)
		return noop.Float64Histogram{}
	}
	return histogram
}
