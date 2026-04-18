package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	echootel "github.com/labstack/echo-opentelemetry"
	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	logotel "go.opentelemetry.io/otel/log"
	logotelglobal "go.opentelemetry.io/otel/log/global"
	logotelnoop "go.opentelemetry.io/otel/log/noop"
	metricotel "go.opentelemetry.io/otel/metric"
	metricotelnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	logsdk "go.opentelemetry.io/otel/sdk/log"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	traceotel "go.opentelemetry.io/otel/trace"
	traceotelnoop "go.opentelemetry.io/otel/trace/noop"

	logotelbridge "go.opentelemetry.io/contrib/bridges/otelslog"
	attributeotel "go.opentelemetry.io/otel/attribute"
)

func newOtelBridgeHandler() slog.Handler {
	bridgeHandler := logotelbridge.NewHandler(
		"slog",
		logotelbridge.WithLoggerProvider(logotelglobal.GetLoggerProvider()),
		logotelbridge.WithAttributes(attributeotel.String("source", "slog")),
	)

	return &fullTextMessageHandler{next: bridgeHandler}
}

type fullTextMessageHandler struct {
	next   slog.Handler
	attrs  []slog.Attr
	groups []string
}

func (h *fullTextMessageHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *fullTextMessageHandler) Handle(ctx context.Context, r slog.Record) error {
	parts := make([]string, 0)
	for _, attr := range h.attrs {
		parts = appendAttrParts(parts, h.groups, attr)
	}
	r.Attrs(func(attr slog.Attr) bool {
		parts = appendAttrParts(parts, h.groups, attr)
		return true
	})

	message := r.Message
	if len(parts) > 0 {
		message = fmt.Sprintf("%s %s", message, strings.Join(parts, " "))
	}

	record := slog.NewRecord(r.Time, r.Level, message, r.PC)
	r.Attrs(func(attr slog.Attr) bool {
		record.AddAttrs(attr)
		return true
	})

	return h.next.Handle(ctx, record)
}

func (h *fullTextMessageHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	combined := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	combined = append(combined, h.attrs...)
	combined = append(combined, attrs...)

	groups := make([]string, len(h.groups))
	copy(groups, h.groups)

	return &fullTextMessageHandler{
		next:   h.next.WithAttrs(attrs),
		attrs:  combined,
		groups: groups,
	}
}

func (h *fullTextMessageHandler) WithGroup(name string) slog.Handler {
	attrs := make([]slog.Attr, len(h.attrs))
	copy(attrs, h.attrs)

	groups := make([]string, 0, len(h.groups)+1)
	groups = append(groups, h.groups...)
	groups = append(groups, name)

	return &fullTextMessageHandler{
		next:   h.next.WithGroup(name),
		attrs:  attrs,
		groups: groups,
	}
}

func appendAttrParts(parts []string, groups []string, attr slog.Attr) []string {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return parts
	}

	if attr.Value.Kind() == slog.KindGroup {
		nextGroups := groups
		if attr.Key != "" {
			nextGroups = append(append([]string(nil), groups...), attr.Key)
		}
		for _, nestedAttr := range attr.Value.Group() {
			parts = appendAttrParts(parts, nextGroups, nestedAttr)
		}
		return parts
	}

	keyParts := groups
	if attr.Key != "" {
		keyParts = append(append([]string(nil), groups...), attr.Key)
	}
	key := strings.Join(keyParts, ".")
	if key == "" {
		return parts
	}

	return append(parts, fmt.Sprintf("%s=%s", key, formatAttrValue(attr.Value)))
}

func formatAttrValue(value slog.Value) string {
	switch value.Kind() {
	case slog.KindString:
		stringValue := value.String()
		if strings.ContainsAny(stringValue, " \t\n\r\"=") {
			return strconv.Quote(stringValue)
		}
		return stringValue
	default:
		return value.String()
	}
}

func setupOtel(ctx context.Context) (func(context.Context) error, error) {
	res, err := resource.New(
		ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String("kyoo.transcoder")),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Configuring OTEL")

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	var le logsdk.Exporter
	var me metricsdk.Exporter
	var te tracesdk.SpanExporter
	switch {
	case strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")) == "":
		slog.InfoContext(ctx, "Using OLTP type", "type", "noop")
		le = nil
		me = nil
		te = nil
	case strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"))) == "grpc":
		slog.InfoContext(ctx, "Using OLTP type", "type", "grpc")
		le, err = otlploggrpc.New(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed setting up OLTP", "err", err)
			return nil, err
		}
		me, err = otlpmetricgrpc.New(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed setting up OLTP", "err", err)
			return nil, err
		}
		te, err = otlptracegrpc.New(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed setting up OLTP", "err", err)
			return nil, err
		}
	default:
		slog.InfoContext(ctx, "Using OLTP type", "type", "http")
		le, err = otlploghttp.New(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed setting up OLTP", "err", err)
			return nil, err
		}
		me, err = otlpmetrichttp.New(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed setting up OLTP", "err", err)
			return nil, err
		}
		te, err = otlptracehttp.New(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed setting up OLTP", "err", err)
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}

	// default to noop providers
	var lp logotel.LoggerProvider = logotelnoop.NewLoggerProvider()
	var mp metricotel.MeterProvider = metricotelnoop.NewMeterProvider()
	var tp traceotel.TracerProvider = traceotelnoop.NewTracerProvider()

	// use exporter if configured
	if le != nil {
		lp = logsdk.NewLoggerProvider(
			logsdk.WithProcessor(logsdk.NewBatchProcessor(le)),
			logsdk.WithResource(res),
		)
	}

	if me != nil {
		mp = metricsdk.NewMeterProvider(
			metricsdk.WithReader(
				metricsdk.NewPeriodicReader(me),
			),
			metricsdk.WithResource(res),
		)
	}

	if te != nil {
		tp = tracesdk.NewTracerProvider(
			tracesdk.WithBatcher(te),
			tracesdk.WithResource(res),
		)
	}

	// set providers
	logotelglobal.SetLoggerProvider(lp)
	otel.SetMeterProvider(mp)
	otel.SetTracerProvider(tp)

	// configure shutting down
	// noop providers do not have a Shudown method
	log_shutdown := func(ctx context.Context) error {
		if otelprovider, ok := lp.(*logsdk.LoggerProvider); ok && otelprovider != nil {
			return otelprovider.Shutdown(ctx)
		}
		return nil
	}

	metric_shutdown := func(ctx context.Context) error {
		if otelprovider, ok := mp.(*metricsdk.MeterProvider); ok && otelprovider != nil {
			return otelprovider.Shutdown(ctx)
		}
		return nil
	}

	trace_shutdown := func(ctx context.Context) error {
		if otelprovider, ok := tp.(*tracesdk.TracerProvider); ok && otelprovider != nil {
			return otelprovider.Shutdown(ctx)
		}
		return nil
	}

	return func(ctx context.Context) error {
		slog.InfoContext(ctx, "Shutting down OTEL")

		// run shutdowns and collect errors
		var errs []error
		if err := trace_shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
		if err := metric_shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
		if err := log_shutdown(ctx); err != nil {
			errs = append(errs, err)
		}

		if len(errs) == 0 {
			return nil
		}
		return errors.Join(errs...)
	}, nil
}

func instrument(e *echo.Echo) {
	e.Use(echootel.NewMiddlewareWithConfig(echootel.Config{
		ServerName: "kyoo.transcoder",
		Skipper: func(c *echo.Context) bool {
			return (c.Path() == "/video/health" ||
				c.Path() == "/video/ready")
		},
	}))
}
