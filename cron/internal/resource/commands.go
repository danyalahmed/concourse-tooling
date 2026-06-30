package resource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
	"github.com/robfig/cron/v3"
)

func (d *Driver) Check(ctx context.Context, source Source, version *sdk.Version) ([]sdk.Version, error) {
	loc := time.UTC
	if source.Location != "" {
		var err error
		if loc, err = time.LoadLocation(source.Location); err != nil {
			return nil, err
		}
	}

	now := time.Now().In(loc)

	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	sched, err := parser.Parse(source.Expression)
	if err != nil {
		return nil, err
	}

	// First execution logic (no historical version provided)
	if version == nil || version.Ref == "" {
		lastTrigger := getLatestPassedTrigger(ctx, sched, now)
		if lastTrigger.IsZero() {
			// If no trigger occurred within the past year, use 'now' as the baseline pinning point
			lastTrigger = now
		}
		return []sdk.Version{{Ref: lastTrigger.Format(time.RFC3339)}}, nil
	}

	// Subsequent runs delta tracking
	lastVersion, err := time.Parse(time.RFC3339, version.Ref)
	if err != nil {
		return nil, err
	}
	lastVersion = lastVersion.In(loc)

	// Gather all cron triggers between last version and now.
	// Cap at 100 versions to avoid CPU/memory spike if the resource was inactive for long.
	var versions []sdk.Version
	for cursor := sched.Next(lastVersion); !cursor.After(now); cursor = sched.Next(cursor) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		versions = append(versions, sdk.Version{Ref: cursor.Format(time.RFC3339)})
		if len(versions) >= 100 {
			break
		}
	}

	// If no new cron marks were hit, maintain the status quo
	if len(versions) == 0 {
		return []sdk.Version{*version}, nil
	}

	return versions, nil
}

func (d *Driver) In(ctx context.Context, source Source, version sdk.Version, params InParams, targetDir string) (sdk.Version, sdk.Metadata, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return version, nil, err
	}

	if err := os.WriteFile(filepath.Join(targetDir, "timestamp"), []byte(version.Ref), 0644); err != nil {
		return version, nil, fmt.Errorf("failed to write timestamp: %w", err)
	}

	return version, sdk.Metadata{
		{Name: "triggered_time", Value: version.Ref},
	}, nil
}

func (d *Driver) Out(ctx context.Context, source Source, params OutParams, sourceDir string) (sdk.Version, sdk.Metadata, error) {
	return sdk.Version{}, nil, fmt.Errorf("the put/out step is not supported for cron resource")
}

// getLatestPassedTrigger looks backward using progressive evaluation windows
// to avoid heavy CPU loops on high-frequency (e.g., secondly/minutely) crons.
func getLatestPassedTrigger(ctx context.Context, sched cron.Schedule, now time.Time) time.Time {
	windows := []time.Duration{
		1 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,       // 1 day
		7 * 24 * time.Hour,   // 1 week
		30 * 24 * time.Hour,  // 1 month
		365 * 24 * time.Hour, // 1 year
	}

	for _, window := range windows {
		if ctx.Err() != nil {
			return time.Time{}
		}

		backDated := now.Add(-window)
		cursor := sched.Next(backDated)

		// If the first trigger after backdating is already in the future,
		// nothing triggered in this window. Try a larger window.
		if cursor.After(now) {
			continue
		}

		var lastTrigger time.Time
		for !cursor.After(now) {
			if ctx.Err() != nil {
				return time.Time{}
			}
			lastTrigger = cursor
			cursor = sched.Next(cursor)
		}
		return lastTrigger
	}

	return time.Time{}
}
