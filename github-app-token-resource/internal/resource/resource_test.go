package resource

import (
	"context"
	"testing"
)

func TestCheck(t *testing.T) {
	d := &Driver{}
	source := Source{
		AppID: "123",
	}

	versions, err := d.Check(context.Background(), source, nil)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("Expected 1 version, got %d", len(versions))
	}
}

func TestOut(t *testing.T) {
	d := &Driver{}
	version, _, err := d.Out(context.Background(), Source{}, nil, "")
	if err != nil {
		t.Fatalf("Out failed: %v", err)
	}
	if version.Ref != "noop" {
		t.Errorf("Expected noop ref, got %s", version.Ref)
	}
}
