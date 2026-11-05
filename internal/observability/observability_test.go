package observability

import (
	"context"
	"testing"
)

func TestNew_CodeInternalError_UnreachableCollector(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:59999")

	provider, err := New(context.Background(), "trip-service")
	if err != nil {
		t.Fatalf("New не должен падать без коллектора: %v", err)
	}

	if err := provider.Shutdown(context.Background()); err != nil {
		t.Logf("shutdown вернул ошибку (допустимо): %v", err)
	}
}
