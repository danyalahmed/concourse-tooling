package sdk

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestPrepareSSHKeyFile(t *testing.T) {
	ctx := context.Background()
	sshKey := "test-key"

	// Test without passphrase
	keyPath, cleanup, err := PrepareSSHKeyFile(ctx, sshKey, "")
	if err != nil {
		t.Fatalf("PrepareSSHKeyFile failed: %v", err)
	}
	defer cleanup()

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Errorf("Key file does not exist at %s", keyPath)
	}

	content, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to read key file: %v", err)
	}

	if strings.TrimSpace(string(content)) != sshKey {
		t.Errorf("Expected content %q, got %q", sshKey, string(content))
	}
}

func TestToSMBPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"path/to/file", "path\\to\\file"},
		{"/path/to/dir/", "path\\to\\dir"},
		{"path\\to\\file", "path\\to\\file"},
	}

	for _, tt := range tests {
		if got := ToSMBPath(tt.input); got != tt.expected {
			t.Errorf("ToSMBPath(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}
