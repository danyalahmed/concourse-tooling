package resource

import (
	"context"
	"testing"
)

func TestCheck(t *testing.T) {
	d := &Driver{}
	source := Source{}

	versions, err := d.Check(context.Background(), source, nil)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("Expected 1 version, got %d", len(versions))
	}
}
