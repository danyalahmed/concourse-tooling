package resource

import (
	"context"
	"testing"
	"time"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

func TestCheck(t *testing.T) {
	d := &Driver{}
	source := Source{
		Expression: "*/5 * * * *", // every 5 minutes
	}

	// Test first run
	versions, err := d.Check(context.Background(), source, nil)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("Expected 1 version on first run, got %d", len(versions))
	}

	// Test subsequent run with no new triggers
	lastVersion := versions[0]
	// Mock time would be better, but we can just use the last version
	versions2, err := d.Check(context.Background(), source, &lastVersion)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(versions2) != 1 || versions2[0].Ref != lastVersion.Ref {
		t.Errorf("Expected same version when no triggers, got %v", versions2)
	}

	// Test subsequent run with triggers
	pastTime := time.Now().Add(-12 * time.Minute).Truncate(time.Minute).Format(time.RFC3339)
	pastVersion := sdk.Version{Ref: pastTime}
	versions3, err := d.Check(context.Background(), source, &pastVersion)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(versions3) < 2 {
		t.Errorf("Expected at least 2 triggers, got %d", len(versions3))
	}
}
