package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	gcpexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

var tracer = otel.Tracer("github.com/pecolynx/go-otel/main")

func initTracerExporter(exporterType, jaegerEndPoint string) (sdktrace.SpanExporter, error) {
	switch exporterType {
	case "jaeger":
		// Create the Jaeger exporter
		return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndPoint)))
	case "gcp":
		projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
		return gcpexporter.New(gcpexporter.WithProjectID(projectID))
	case "stdout":
		return stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
			stdouttrace.WithWriter(os.Stderr),
		)
	case "none":
		return stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
			stdouttrace.WithWriter(io.Discard),
		)
	default:
		return nil, fmt.Errorf("unsupported exporter: %s", exporterType)
	}
}

func initTracerProvider(exporter sdktrace.SpanExporter, serviceName string) (*sdktrace.TracerProvider, error) {
	tp := sdktrace.NewTracerProvider(
		// Always be sure to batch in production.
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		// Record information about this application in a Resource.
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	return tp, nil
}

func a(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "a")
	defer span.End()
	time.Sleep(100 * time.Millisecond)
	b(ctx)
}

func b(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "b")
	defer span.End()
	time.Sleep(200 * time.Millisecond)
	c(ctx)
}

func c(ctx context.Context) {
	_, span := tracer.Start(ctx, "c")
	defer span.End()
	time.Sleep(300 * time.Millisecond)
}

func main() {
	// exporterType := "none"
	// exporterType := "stdout"
	exporterType := "jaeger"
	jaegerEndpoint := "http://localhost:14268/api/traces"
	ctx := context.Background()
	exp, err := initTracerExporter(exporterType, jaegerEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	tp, err := initTracerProvider(exp, "go-otel")
	otel.SetTracerProvider(tp)
	if err != nil {
		log.Fatal(err)
	}
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	a(ctx)

	defer tp.ForceFlush(ctx) // flushes any pending spans
}
