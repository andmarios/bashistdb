package timestamp

import (
	"testing"
)

func TestConvertTrailingTimestamp(t *testing.T) {
	// When the last line of the history is a timestamp with no following command,
	// Convert should handle it gracefully without panicking.
	input := []byte("#1234567890\n")
	// This should not panic
	result := Convert(input, 12)
	if result == nil {
		t.Fatal("Convert returned nil")
	}
}

func TestConvertNormal(t *testing.T) {
	// Regression: normal timestamped history should still work.
	input := []byte("#1234567890\nls\n#1234567895\npwd\n")
	result := Convert(input, 12)
	if result == nil {
		t.Fatal("Convert returned nil")
	}
	s := string(result)
	if len(s) == 0 {
		t.Fatal("Convert returned empty result for valid input")
	}
}

func TestConvertEmptyInput(t *testing.T) {
	// Empty input should not panic; nil or empty output is fine.
	input := []byte("")
	_ = Convert(input, 12)
}
