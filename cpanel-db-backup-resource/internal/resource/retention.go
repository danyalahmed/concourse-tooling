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
	entries, err := share.ReadDir(parentDir)
	if err != nil {
		return fmt.Errorf("reading parent directory %s: %w", parentDir, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "." && entry.Name() != ".." {
			names = append(names, entry.Name())
		}
	}

	toRemove := calculateBackupsToRemove(names, source)

	for _, name := range toRemove {
		path := sdk.ToSMBPath(parentDir + "/" + name)
		sdk.Logf("Removing old database backup: %s", name)
		if err := share.RemoveAll(path); err != nil {
			return fmt.Errorf("removing backup %s: %w", name, err)
		}
	}

	return nil
}

func calculateBackupsToRemove(names []string, source Source) []string {
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

	var backups []dbBackupEntry
	for _, name := range names {
		// Expecting format YYYY-MM-DD
		ts, err := time.Parse("2006-01-02", name)
		if err != nil {
			continue
		}
		backups = append(backups, dbBackupEntry{
			name:      name,
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
	d, w, m, y := daily, weekly, monthly, yearly
	for _, b := range backups {
		year, month, day := b.timestamp.Date()
		_, week := b.timestamp.ISOWeek()

		yearlyKey := fmt.Sprintf("y-%d", year)
		monthlyKey := fmt.Sprintf("m-%d-%d", year, month)
		weeklyKey := fmt.Sprintf("w-%d-%d", year, week)
		dailyKey := fmt.Sprintf("d-%d-%d-%d", year, month, day)

		keepThis := false
		if y > 0 && !keep[yearlyKey] {
			keep[yearlyKey] = true
			y--
			keepThis = true
		}
		if m > 0 && !keep[monthlyKey] {
			keep[monthlyKey] = true
			m--
			keepThis = true
		}
		if w > 0 && !keep[weeklyKey] {
			keep[weeklyKey] = true
			w--
			keepThis = true
		}
		if d > 0 && !keep[dailyKey] {
			keep[dailyKey] = true
			d--
			keepThis = true
		}

		if keepThis {
			keep[b.name] = true
		}
	}

	var toRemove []string
	for _, b := range backups {
		if !keep[b.name] {
			toRemove = append(toRemove, b.name)
		}
	}

	return toRemove
}
