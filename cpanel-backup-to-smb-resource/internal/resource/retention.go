package resource

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cloudsoda/go-smb2"
	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

type backupEntry struct {
	name      string
	path      string
	timestamp time.Time
}

func retainBackups(share *smb2.Share, parentDir string, keepCount, keepDays int) error {
	if keepCount <= 0 && keepDays <= 0 {
		return nil
	}

	entries, err := share.ReadDir(parentDir)
	if err != nil {
		return fmt.Errorf("reading parent directory %s: %w", parentDir, err)
	}

	var backups []backupEntry
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || name == "." || name == ".." {
			continue
		}
		if !strings.HasPrefix(name, "backup_") {
			continue
		}
		ts, err := time.Parse("2006-01-02_15-04-05", strings.TrimPrefix(name, "backup_"))
		if err != nil {
			continue
		}
		backups = append(backups, backupEntry{
			name:      name,
			path:      parentDir + "\\" + name,
			timestamp: ts,
		})
	}

	if len(backups) <= 1 {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].timestamp.After(backups[j].timestamp)
	})

	now := time.Now()
	var toRemove []backupEntry

	for i, b := range backups {
		if keepCount > 0 && i >= keepCount {
			toRemove = append(toRemove, b)
			continue
		}
		if keepDays > 0 && now.Sub(b.timestamp) > time.Duration(keepDays)*24*time.Hour {
			toRemove = append(toRemove, b)
		}
	}

	for _, b := range toRemove {
		sdk.Logf("Removing old backup: %s", b.name)
		if err := share.RemoveAll(b.path); err != nil {
			return fmt.Errorf("removing backup %s: %w", b.name, err)
		}
	}

	return nil
}
