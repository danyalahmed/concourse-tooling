package sdk

import (
	"testing"
)

func TestToSMBPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo/bar", "foo\\bar"},
		{"/foo/bar/", "foo\\bar"},
		{"foo\\bar", "foo\\bar"},
		{"\\\\foo\\bar\\\\", "foo\\bar"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := ToSMBPath(tt.input); got != tt.expected {
			t.Errorf("ToSMBPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "'foo'"},
		{"foo'bar", "'foo'\\''bar'"},
		{"", "''"},
	}

	for _, tt := range tests {
		if got := ShellQuote(tt.input); got != tt.expected {
			t.Errorf("ShellQuote(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
