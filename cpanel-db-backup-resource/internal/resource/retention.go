package resource

import (
	"fmt"
	"sort"
	"time"

	"github.com/cloudsoda/go-smb2"
	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

type dbBackupEntry struct {
	name      string
	path      string
	timestamp time.Time
}

func applyDBRetention(share *smb2.Share, parentDir string, source Source) error {
	daily := source.KeepDaily
	if daily == 0 {
		daily = 7
	}
	weekly := source.KeepWeekly
	if weekly == 0 {
		weekly = 4
	}
	monthly := source.KeepMonthly
	if monthly == 0 {
		monthly = 12
	}
	yearly := source.KeepYearly
	if yearly == 0 {
		yearly = 3
	}

	entries, err := share.ReadDir(parentDir)
	if err != nil {
		return fmt.Errorf("reading parent directory %s: %w", parentDir, err)
	}

	var backups []dbBackupEntry
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || name == "." || name == ".." {
			continue
		}
		// Expecting format YYYY-MM-DD
		ts, err := time.Parse("2006-01-02", name)
		if err != nil {
			continue
		}
		backups = append(backups, dbBackupEntry{
			name:      name,
			path:      sdk.ToSMBPath(parentDir + "/" + name),
			timestamp: ts,
		})
	}

	if len(backups) == 0 {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].timestamp.After(backups[j].timestamp)
	})

	keep := make(map[string]bool)

	// GFS Retention Logic
	for _, b := range backups {
		year, month, day := b.timestamp.Date()
		_, week := b.timestamp.ISOWeek()

		yearlyKey := fmt.Sprintf("y-%d", year)
		monthlyKey := fmt.Sprintf("m-%d-%d", year, month)
		weeklyKey := fmt.Sprintf("w-%d-%d", year, week)
		dailyKey := fmt.Sprintf("d-%d-%d-%d", year, month, day)

		if yearly > 0 {
			if !keep[yearlyKey] {
				keep[yearlyKey] = true
				yearly--
				keep[b.name] = true
			}
		}
		if monthly > 0 {
			if !keep[monthlyKey] {
				keep[monthlyKey] = true
				monthly--
				keep[b.name] = true
			}
		}
		if weekly > 0 {
			if !keep[weeklyKey] {
				keep[weeklyKey] = true
				weekly--
				keep[b.name] = true
			}
		}
		if daily > 0 {
			if !keep[dailyKey] {
				keep[dailyKey] = true
				daily--
				keep[b.name] = true
			}
		}
	}

	for _, b := range backups {
		if !keep[b.name] {
			sdk.Logf("Removing old database backup: %s", b.name)
			if err := share.RemoveAll(b.path); err != nil {
				return fmt.Errorf("removing backup %s: %w", b.name, err)
			}
		}
	}

	return nil
}
