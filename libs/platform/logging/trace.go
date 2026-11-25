package logging

import (
	"context"
	"net/http"
	"regexp"

	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	TraceParentHeader   = "traceparent"
	instrumentationName = "collector/libs/platform/logging"
	spanName            = "collector.traitement"
)

var (
	traceParentFormat = regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`)
	propagator        = propagation.TraceContext{}
	tracer            = sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.NeverSample()))).Tracer(instrumentationName)
)

func ContinueTrace(ctx context.Context, receivedTraceParent string) context.Context {
	if parent, received := remoteSpanContext(receivedTraceParent); received {
		ctx, _ = tracer.Start(trace.ContextWithRemoteSpanContext(ctx, parent), spanName)
		return ctx
	}
	ctx, _ = tracer.Start(ctx, spanName, trace.WithNewRoot())
	return ctx
}

func restoreTrace(ctx context.Context, traceParent string) context.Context {
	if current, valid := remoteSpanContext(traceParent); valid {
		return trace.ContextWithSpanContext(ctx, current)
	}
	return ContinueTrace(ctx, "")
}

func TraceParent(ctx context.Context) (string, bool) {
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)
	traceParent := carrier.Get(TraceParentHeader)
	return traceParent, traceParent != ""
}

func injectTraceParent(ctx context.Context, header http.Header) {
	propagator.Inject(ctx, propagation.HeaderCarrier(header))
}

func currentSpan(ctx context.Context) (trace.SpanContext, bool) {
	current := trace.SpanContextFromContext(ctx)
	return current, current.IsValid()
}

func remoteSpanContext(traceParent string) (trace.SpanContext, bool) {
	if !traceParentFormat.MatchString(traceParent) {
		return trace.SpanContext{}, false
	}
	extracted := propagator.Extract(context.Background(), propagation.MapCarrier{TraceParentHeader: traceParent})
	return currentSpan(extracted)
}
