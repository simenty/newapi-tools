// NewAPI Tools - Docker management platform for newapi
package ui

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/Bonus520/newapi-tools/internal/core"
)

func TestSetupLoggerTextFormat(t *testing.T) {
	cfg := &core.LogConfig{Level: "info", Format: "text"}
	SetupLogger(cfg)

	logger := slog.Default()
	if logger == nil {
		t.Fatal("logger should not be nil after SetupLogger")
	}
}

func TestSetupLoggerJSONFormat(t *testing.T) {
	cfg := &core.LogConfig{Level: "debug", Format: "json"}
	SetupLogger(cfg)

	logger := slog.Default()
	if logger == nil {
		t.Fatal("logger should not be nil after SetupLogger")
	}
}

func TestSetupLoggerInvalidLevel(t *testing.T) {
	cfg := &core.LogConfig{Level: "invalid", Format: "text"}
	SetupLogger(cfg)

	// Should default to info level, not panic
	logger := slog.Default()
	if logger == nil {
		t.Fatal("logger should not be nil even with invalid level")
	}
}

func TestL(t *testing.T) {
	logger := L()
	if logger == nil {
		t.Fatal("L() should return non-nil logger")
	}
}

func TestProgressAdd(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, "test", 10)
	p.Add(5)
	output := buf.String()
	if !strings.Contains(output, "50%") {
		t.Errorf("expected 50%% progress, got: %s", output)
	}
}

func TestProgressDone(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, "test", 10)
	p.Add(10)
	p.Done()
	output := buf.String()
	if !strings.Contains(output, "100%") {
		t.Errorf("expected 100%% progress, got: %s", output)
	}
}

func TestProgressOverTotal(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, "test", 10)
	p.Add(15) // exceeds total
	if p.current != 10 {
		t.Errorf("expected current to be capped at 10, got %d", p.current)
	}
}

func TestTableRender(t *testing.T) {
	var buf bytes.Buffer
	table := NewTable("Name", "Status")
	table.SetWriter(&buf)
	table.AddRow("new-api", "running")
	table.AddRow("redis", "stopped")
	table.Render()

	output := buf.String()
	if !strings.Contains(output, "Name") {
		t.Error("table should contain header 'Name'")
	}
	if !strings.Contains(output, "new-api") {
		t.Error("table should contain row 'new-api'")
	}
	if !strings.Contains(output, "running") {
		t.Error("table should contain row 'running'")
	}
}

func TestTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	table := NewTable()
	table.SetWriter(&buf)
	table.Render() // Should not panic
}

func TestTableShortRow(t *testing.T) {
	var buf bytes.Buffer
	table := NewTable("A", "B", "C")
	table.SetWriter(&buf)
	table.AddRow("only-one") // shorter than headers
	table.Render()

	output := buf.String()
	if !strings.Contains(output, "only-one") {
		t.Error("table should contain the short row value")
	}
}

func TestKeyValue(t *testing.T) {
	var buf bytes.Buffer
	KeyValue(&buf, "port", 3000)

	output := buf.String()
	if !strings.Contains(output, "port:") {
		t.Error("should contain key with colon")
	}
	if !strings.Contains(output, "3000") {
		t.Error("should contain value")
	}
}

func TestInfoBox(t *testing.T) {
	var buf bytes.Buffer
	InfoBox(&buf, "Status", "Container: running", "Port: 3000")

	output := buf.String()
	if !strings.Contains(output, "Status") {
		t.Error("info box should contain title")
	}
	if !strings.Contains(output, "Container: running") {
		t.Error("info box should contain lines")
	}
	if !strings.Contains(output, "┌") {
		t.Error("info box should contain border characters")
	}
}
