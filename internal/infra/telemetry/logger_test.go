package telemetry_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Vllad3/tripgo-avito/internal/infra/telemetry"
)

func TestNewLogger_WritesJSONWithService(t *testing.T) {
	var buf bytes.Buffer
	log, err := telemetry.NewLogger(&buf, "info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	log.Info("hello")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log line is not JSON: %v: %q", err, buf.String())
	}
	if rec["service"] != "trip-service" || rec["msg"] != "hello" || rec["level"] != "INFO" {
		t.Errorf("record = %v", rec)
	}
}

func TestNewLogger_RespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	log, err := telemetry.NewLogger(&buf, "warn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	log.Info("must be dropped")

	if buf.Len() != 0 {
		t.Errorf("info record written at level warn: %s", buf.String())
	}
}

func TestNewLogger_InvalidLevel(t *testing.T) {
	if _, err := telemetry.NewLogger(&bytes.Buffer{}, "loud"); err == nil {
		t.Fatal("expected an error for an unknown level")
	}
}
