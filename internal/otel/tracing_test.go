package otel

import (
	"context"
	"os"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

func TestInitDisabled(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "true")

	shutdown, err := Init(context.Background(), time.UTC)
	if err != nil {
		t.Fatalf("expected no error when OTEL_SDK_DISABLED is true, got: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("expected shutdown to succeed, got: %v", err)
	}

	propagator := otel.GetTextMapPropagator()
	if propagator == nil {
		t.Fatal("expected propagator to be registered")
	}
}

func TestInitInvalidProtocolGracefulDegradation(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "false")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "invalid-protocol")

	shutdown, err := Init(context.Background(), time.UTC)
	if err != nil {
		t.Fatalf("expected graceful degradation without error on invalid protocol, got: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("expected shutdown to succeed, got: %v", err)
	}
}

func TestGetSampler(t *testing.T) {
	tests := []struct {
		name    string
		sampler string
		arg     string
	}{
		{"always_on", "always_on", ""},
		{"always_off", "always_off", ""},
		{"traceidratio", "traceidratio", "0.5"},
		{"parentbased_always_on", "parentbased_always_on", ""},
		{"parentbased_always_off", "parentbased_always_off", ""},
		{"parentbased_traceidratio", "parentbased_traceidratio", "0.25"},
		{"default_fallback", "unknown_sampler", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OTEL_TRACES_SAMPLER", tt.sampler)
			t.Setenv("OTEL_TRACES_SAMPLER_ARG", tt.arg)

			s := getSampler()
			if s == nil {
				t.Fatalf("expected sampler to be created for %s", tt.name)
			}
			desc := s.Description()
			if desc == "" {
				t.Fatalf("expected non-empty description for sampler %s", tt.name)
			}
		})
	}
}

func TestLogStartupAndErrorNilLoc(t *testing.T) {
	// Should not panic with nil loc
	logStartup(time.UTC, false, "", "", "", "")
	logStartup(time.UTC, true, "grpc", "localhost:4317", "parentbased_traceidratio", "1.0")
	logError(time.UTC, os.ErrNotExist)
}
