package result

import (
	"strings"
	"testing"
	"time"

	conf "github.com/andmarios/bashistdb/configuration"
)

func TestFormattedUsesInstanceFormat(t *testing.T) {
	// Create a Result with JSON format
	r := New(conf.FORMAT_JSON)
	r.AddRow(1, "user", "host", "ls", "", "", time.Now())
	out := r.Formatted()
	s := string(out)

	// JSON result should start with "[" and end with "]"
	if !strings.HasPrefix(s, "[") {
		t.Fatalf("JSON formatted result should start with '[', got: %s", s)
	}
	if !strings.HasSuffix(s, "]") {
		t.Fatalf("JSON formatted result should end with ']', got: %s", s)
	}

	// Create a Result with non-JSON format — should NOT have brackets
	r2 := New(conf.FORMAT_COMMAND_LINE)
	r2.AddRow(1, "user", "host", "ls", "", "", time.Now())
	out2 := r2.Formatted()
	s2 := string(out2)

	if strings.HasSuffix(s2, "]") {
		t.Fatalf("Non-JSON formatted result should NOT end with ']', got: %s", s2)
	}
}

func TestFormattedEmptyJSON(t *testing.T) {
	r := New(conf.FORMAT_JSON)
	out := r.Formatted()
	s := string(out)

	// Even with no rows, should produce valid JSON array
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		t.Fatalf("Empty JSON result should be '[]' or '[\\n]', got: %q", s)
	}
}

func TestFormattedNonJSON(t *testing.T) {
	r := New(conf.FORMAT_ALL)
	r.AddRow(1, "user", "host", "pwd", "123", "/home", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	out := r.Formatted()

	if len(out) == 0 {
		t.Fatal("Formatted returned empty for non-JSON format with data")
	}
}
