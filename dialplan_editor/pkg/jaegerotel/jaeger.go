package jaegerotel

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

type tracerProvider struct {
	service     string
	environment string
}

func NewJaegerTracerProvider(url string, options ...JaegerTracerProviderOption) (*tracesdk.TracerProvider, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}

	t := &tracerProvider{}

	for _, opt := range options {
		opt(t)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(t.service),
			attribute.String("environment", t.environment),
		)),
	)

	return tp, nil
}

func GetTracer() trace.Tracer {
	return otel.Tracer("aster-bridge-event-parser")
}

func StartNewSpan(name string) (context.Context, trace.Span) {
	return GetTracer().Start(context.Background(), name)
}

func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, name)
}
