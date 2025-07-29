package tracing

import (
	"context"
	"fmt"
	"strings"

	"github.com/pirogoeth/apps/pkg/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc/credentials"
)

type (
	TracerBuilder func(*tracer)
	tracer        struct {
		appName        string
		componentName  string
		cfg            config.TracingConfig
		parentCtx      context.Context
		exporter       *otlptrace.Exporter
		tracerProvider *sdktrace.TracerProvider
	}
)

func WithAppName(appName string) TracerBuilder {
	return func(t *tracer) {
		t.appName = appName
	}
}

func WithComponentName(componentName string) TracerBuilder {
	return func(t *tracer) {
		t.componentName = componentName
	}
}

func WithConfig(cfg config.TracingConfig) TracerBuilder {
	return func(t *tracer) {
		t.cfg = cfg
	}
}

func WithParentContext(ctx context.Context) TracerBuilder {
	return func(t *tracer) {
		t.parentCtx = ctx
	}
}

func makeExporterClient(cfg config.TracingConfig) otlptrace.Client {
	switch strings.ToLower(cfg.ExporterProtocol) {
	case "grpc":
		secureOption := otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")) // config can be passed to configure TLS
		if cfg.ExporterInsecure {
			secureOption = otlptracegrpc.WithInsecure()
		}

		return otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(cfg.ExporterEndpoint),
			otlptracegrpc.WithHeaders(cfg.ExporterHeaders),
		)
	case "http":
		return otlptracehttp.NewClient(otlptracehttp.WithEndpoint(cfg.ExporterEndpoint))
	}

	return nil
}

func Setup(options ...TracerBuilder) *tracer {
	tracer := &tracer{
		parentCtx: context.Background(),
	}
	for _, optionFn := range options {
		optionFn(tracer)
	}

	if tracer.appName == "" || tracer.componentName == "" {
		logrus.Warnf("Tracing disabled because AppName and ComponentName not set")
		return nil
	}

	if !tracer.cfg.Enabled {
		logrus.Debug("Tracing disabled by config")
		return nil
	}

	traceServiceName := fmt.Sprintf("%s/%s", tracer.appName, tracer.componentName)
	logrus.Debugf("Tracing initialized for %s with config: %#v",
		traceServiceName,
		tracer.cfg,
	)

	exporter, err := otlptrace.New(tracer.parentCtx, makeExporterClient(tracer.cfg))
	if err != nil {
		// TODO: Should this be fatal?
		logrus.Errorf("could not create tracing exporter: %v", err)
	}
	tracer.exporter = exporter

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(tracer.cfg.SamplerRate)),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exporter)),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(traceServiceName))),
	)
	tracer.tracerProvider = tracerProvider
	otel.SetTracerProvider(tracerProvider)
	// otel.SetLogger(stdr.New(logrus.StandardLogger().Writer()))
	go tracer.waitForShutdown()

	return tracer
}

func (t *tracer) waitForShutdown() {
	for range t.parentCtx.Done() {
		if err := t.tracerProvider.Shutdown(t.parentCtx); err != nil {
			logrus.Errorf("error shutting down tracer provider: %v", err)
		}
	}
}
