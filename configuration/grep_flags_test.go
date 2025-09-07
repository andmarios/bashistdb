package configuration

import (
	"os"
	"reflect"
	"testing"
)

func TestPreprocessGrepStyleFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Single grep-style -A flag",
			input:    []string{"bashistdb", "-A5", "test"},
			expected: []string{"bashistdb", "-A", "5", "test"},
		},
		{
			name:     "Single grep-style -B flag",
			input:    []string{"bashistdb", "-B10", "query"},
			expected: []string{"bashistdb", "-B", "10", "query"},
		},
		{
			name:     "Single grep-style -C flag",
			input:    []string{"bashistdb", "-C3", "search"},
			expected: []string{"bashistdb", "-C", "3", "search"},
		},
		{
			name:     "Multiple grep-style flags",
			input:    []string{"bashistdb", "-A2", "-B3", "-C4", "test"},
			expected: []string{"bashistdb", "-A", "2", "-B", "3", "-C", "4", "test"},
		},
		{
			name:     "Mixed grep-style and regular flags",
			input:    []string{"bashistdb", "-A5", "-verbose", "2", "-B10", "test"},
			expected: []string{"bashistdb", "-A", "5", "-verbose", "2", "-B", "10", "test"},
		},
		{
			name:     "Traditional format unchanged",
			input:    []string{"bashistdb", "-A", "5", "test"},
			expected: []string{"bashistdb", "-A", "5", "test"},
		},
		{
			name:     "Other single-letter flags unchanged",
			input:    []string{"bashistdb", "-g", "-R", "-u", "test"},
			expected: []string{"bashistdb", "-g", "-R", "-u", "test"},
		},
		{
			name:     "Large numbers in grep-style",
			input:    []string{"bashistdb", "-A100", "-B999", "test"},
			expected: []string{"bashistdb", "-A", "100", "-B", "999", "test"},
		},
		{
			name:     "Don't split non-ABC flags with numbers",
			input:    []string{"bashistdb", "-D5", "-E10", "test"},
			expected: []string{"bashistdb", "-D5", "-E10", "test"},
		},
		{
			name:     "Empty args after program name",
			input:    []string{"bashistdb"},
			expected: []string{"bashistdb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original os.Args
			origArgs := os.Args
			defer func() { os.Args = origArgs }()

			// Set test input
			os.Args = tt.input

			// Run preprocessing
			preprocessGrepStyleFlags()

			// Check result
			if !reflect.DeepEqual(os.Args, tt.expected) {
				t.Errorf("preprocessGrepStyleFlags() got = %v, want %v", os.Args, tt.expected)
			}
		})
	}
}
