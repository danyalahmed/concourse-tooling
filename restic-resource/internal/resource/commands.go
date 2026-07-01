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
	Action string
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
		Repository:       repo,
		Password:         source.RepositoryPass,
		SSHHost:          source.Host,
		SSHUser:          source.Username,
		SSHPort:          source.Port,
		SSHKeyPath:       keyFile,
		SSHKeyPassphrase: source.SSHKeyPassphrase,
		SMBHost:          source.SMBHost,
		SMBShare:         source.SMBShare,
		SMBUser:          source.SMBUsername,
		SMBPass:          source.SMBPassword,
		MountPathSource:  "/mnt/source",
		MountPathTarget:  "/mnt/target",
	}, nil
}

func (d *Driver) In(ctx context.Context, source Source, version sdk.Version, params InParams, targetDir string) (sdk.Version, sdk.Metadata, error) {
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
		args = append(args, buildKeepArgs(source)...)
		args = append(args, "--prune")

		sdk.Log("Starting Restic prune (forget)...")
		out, err := restic.RunRestic(ctx, cfg.Password, args...)
		if err != nil {
			return sdk.Version{}, nil, err
		}
		metadata = append(metadata, sdk.MetadataItem{Name: "output", Value: string(out)})

	case "stats":
		sdk.Logf("Exploring SMB mount at %s...", cfg.MountPathTarget)

		sdk.Log("Running du -sh on mount point...")
		if out, err := exec.CommandContext(ctx, "du", "-sh", cfg.MountPathTarget).CombinedOutput(); err == nil {
			sdk.Log(string(out))
			metadata = append(metadata, sdk.MetadataItem{Name: "disk_usage", Value: string(out)})
		}

		sdk.Log("Listing files on SMB mount (ls -al)...")
		if out, err := exec.CommandContext(ctx, "ls", "-al", cfg.MountPathTarget).CombinedOutput(); err == nil {
			sdk.Log(string(out))
		}

		if _, err := os.Stat(filepath.Join(cfg.Repository, "config")); err == nil {
			sdk.Log("Gathering Restic repository insights...")

			if out, err := restic.RunRestic(ctx, cfg.Password, "-r", cfg.Repository, "snapshots"); err == nil {
				sdk.Log("Snapshots:\n" + string(out))
			}
			if out, err := restic.RunRestic(ctx, cfg.Password, "-r", cfg.Repository, "stats"); err == nil {
				sdk.Log("Stats:\n" + string(out))
			}
		}
	}

	v := fmt.Sprintf("%d", time.Now().Unix())
	return sdk.Version{Ref: v}, metadata, nil
}

func buildKeepArgs(source Source) []string {
	args := []string{}
	policies := []struct {
		name string
		val  int
		def  int
	}{
		{"--keep-daily", source.KeepDaily, 7},
		{"--keep-weekly", source.KeepWeekly, 4},
		{"--keep-monthly", source.KeepMonthly, 12},
		{"--keep-yearly", source.KeepYearly, 3},
	}
	for _, p := range policies {
		v := p.val
		if v == 0 {
			v = p.def
		}
		args = append(args, p.name, fmt.Sprintf("%d", v))
	}
	return args
}
