package resource

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"restic-resource/internal/restic"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

type Driver struct {
	Action string // "backup", "prune", "restore"
}

func (d *Driver) Check(ctx context.Context, source Source, version *sdk.Version) ([]sdk.Version, error) {
	v := fmt.Sprintf("%d", time.Now().Unix())
	return []sdk.Version{{Ref: v}}, nil
}

func (d *Driver) setupConfig(source Source) (restic.Config, error) {
	keyFile := "/tmp/ssh_key"
	cleanedKey := strings.ReplaceAll(source.SSHKey, "\r", "")
	cleanedKey = strings.TrimSpace(cleanedKey) + "\n"
	if err := os.WriteFile(keyFile, []byte(cleanedKey), 0600); err != nil {
		return restic.Config{}, fmt.Errorf("writing ssh key: %w", err)
	}

	repo := "/mnt/target"
	if source.RepositoryPath != "" {
		repo = filepath.Join("/mnt/target", source.RepositoryPath)
	}

	return restic.Config{
		Repository:      repo,
		Password:        source.RepositoryPass,
		SSHHost:         source.Host,
		SSHUser:         source.Username,
		SSHPort:         source.Port,
		SSHKeyPath:      keyFile,
		SSHKeyPassphrase: source.SSHKeyPassphrase,
		SMBHost:         source.SMBHost,
		SMBShare:        source.SMBShare,
		SMBUser:         source.SMBUsername,
		SMBPass:         source.SMBPassword,
		MountPathSource: "/mnt/source",
		MountPathTarget: "/mnt/target",
	}, nil
}

func (d *Driver) In(ctx context.Context, source Source, version sdk.Version, params InParams, targetDir string) (sdk.Version, sdk.Metadata, error) {
	// If it's a restore resource, 'in' performs the restore
	if d.Action == "restore" {
		cfg, err := d.setupConfig(source)
		if err != nil {
			return version, nil, err
		}
		if err := restic.MountAll(ctx, cfg); err != nil {
			return version, nil, err
		}
		defer restic.UnmountAll(cfg)

		snapshotID := params.SnapshotID
		if snapshotID == "" {
			snapshotID = "latest"
		}

		target := cfg.MountPathSource
		if params.TargetSubDir != "" {
			target = filepath.Join(cfg.MountPathSource, params.TargetSubDir)
		}

		sdk.Logf("Restoring snapshot %s to %s...", snapshotID, target)
		_, err = restic.RunRestic(ctx, cfg.Password, "-r", cfg.Repository, "restore", snapshotID, "--target", target)
		if err != nil {
			return version, nil, fmt.Errorf("restore failed: %w", err)
		}

		return version, sdk.Metadata{{Name: "snapshot_id", Value: snapshotID}}, nil
	}

	// For backup/prune, 'in' is just a no-op that reports the version
	return version, nil, nil
}

func (d *Driver) Out(ctx context.Context, source Source, params OutParams, sourceDir string) (sdk.Version, sdk.Metadata, error) {
	cfg, err := d.setupConfig(source)
	if err != nil {
		return sdk.Version{}, nil, err
	}

	action := d.Action
	if params.Action != "" {
		action = params.Action
	}

	if action == "stats" {
		if err := restic.MountSMB(ctx, cfg); err != nil {
			return sdk.Version{}, nil, err
		}
		defer restic.UnmountAll(cfg)
	} else {
		if err := restic.MountAll(ctx, cfg); err != nil {
			return sdk.Version{}, nil, err
		}
		defer restic.UnmountAll(cfg)

		if err := restic.InitIfNeeded(ctx, cfg.Repository, cfg.Password); err != nil {
			return sdk.Version{}, nil, err
		}
	}

	var metadata sdk.Metadata

	switch action {
	case "backup":
		paths := []string{}
		for _, dir := range params.Directories {
			paths = append(paths, filepath.Join(cfg.MountPathSource, filepath.Clean("/"+dir)))
		}
		if len(paths) == 0 {
			paths = append(paths, cfg.MountPathSource)
		}

		args := []string{"-r", cfg.Repository, "backup"}
		for _, exc := range params.Excludes {
			args = append(args, "--exclude", exc)
		}
		args = append(args, paths...)

		sdk.Log("Starting Restic backup...")
		out, err := restic.RunRestic(ctx, cfg.Password, args...)
		if err != nil {
			return sdk.Version{}, nil, err
		}
		metadata = append(metadata, sdk.MetadataItem{Name: "output", Value: string(out)})

	case "prune":
		args := []string{"-r", cfg.Repository, "forget"}

		daily := source.KeepDaily
		if daily == 0 {
			daily = 7
		}
		args = append(args, "--keep-daily", fmt.Sprintf("%d", daily))

		weekly := source.KeepWeekly
		if weekly == 0 {
			weekly = 4
		}
		args = append(args, "--keep-weekly", fmt.Sprintf("%d", weekly))

		monthly := source.KeepMonthly
		if monthly == 0 {
			monthly = 12
		}
		args = append(args, "--keep-monthly", fmt.Sprintf("%d", monthly))

		yearly := source.KeepYearly
		if yearly == 0 {
			yearly = 3
		}
		args = append(args, "--keep-yearly", fmt.Sprintf("%d", yearly))

		args = append(args, "--prune")

		sdk.Log("Starting Restic prune (forget)...")
		out, err := restic.RunRestic(ctx, cfg.Password, args...)
		if err != nil {
			return sdk.Version{}, nil, err
		}
		metadata = append(metadata, sdk.MetadataItem{Name: "output", Value: string(out)})

	case "stats":
		// Action to explore the SMB mount
		sdk.Logf("Exploring SMB mount at %s...", cfg.MountPathTarget)

		// 1. Run du -sh
		sdk.Log("Running du -sh on mount point...")
		out, err := exec.CommandContext(ctx, "du", "-sh", cfg.MountPathTarget).CombinedOutput()
		if err != nil {
			sdk.Logf("Warning: du command failed: %v", err)
		} else {
			sdk.Log(string(out))
			metadata = append(metadata, sdk.MetadataItem{Name: "disk_usage", Value: string(out)})
		}

		// 2. Run ls -R (limited to a few levels if needed, but let's do recursive for now)
		sdk.Log("Listing files on SMB mount (ls -R)...")
		out, err = exec.CommandContext(ctx, "ls", "-R", cfg.MountPathTarget).CombinedOutput()
		if err != nil {
			sdk.Logf("Warning: ls command failed: %v", err)
		} else {
			sdk.Log(string(out))
		}
	}

	v := fmt.Sprintf("%d", time.Now().Unix())
	return sdk.Version{Ref: v}, metadata, nil
}
