package metrics

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"collector/libs/platform/httpserver"
)

const (
	scrapePath        = "GET /metrics"
	serviceNameKey    = "service.name"
	failureEvent      = "metriques.echec"
	failureLogMessage = "mesure des métriques en échec"
)

func Install(serviceName string, logger *slog.Logger) (http.Handler, error) {
	registry := prometheus.NewRegistry()
	reader, err := otelprometheus.New(otelprometheus.WithRegisterer(registry))
	if err != nil {
		return nil, fmt.Errorf("exportateur Prometheus : %w", err)
	}
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Warn(failureLogMessage, slog.String("event", failureEvent), slog.Any("erreur", err))
	}))
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(resource.NewSchemaless(attribute.String(serviceNameKey, serviceName))),
	))
	return scrapeHandler(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})), nil
}

func scrapeHandler(metrics http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(scrapePath, metrics)
	mux.HandleFunc("/", httpserver.NotFound)
	return mux
}
