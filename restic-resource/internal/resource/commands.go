package resource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
	"restic-resource/internal/restic"
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
	if err := os.WriteFile(keyFile, []byte(source.SSHKey), 0600); err != nil {
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
		SMBHost:         source.SMBHost,
		SMBShare:        source.SMBShare,
		SMBUser:         source.SMBUsername,
		SMBPass:         source.SMBPassword,
		MountPathSource: "/mnt/source",
		MountPathTarget: "/mnt/target",
	}, nil
}

func (d *Driver) In(ctx context.Context, source Source, version sdk.Version, params InParams, targetDir string) (sdk.Version, sdk.Metadata, error) {
	if d.Action != "restore" {
		// Default In behavior for backup/prune resources is just to report version
		return version, nil, nil
	}

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

func (d *Driver) Out(ctx context.Context, source Source, params OutParams, sourceDir string) (sdk.Version, sdk.Metadata, error) {
	cfg, err := d.setupConfig(source)
	if err != nil {
		return sdk.Version{}, nil, err
	}
	if err := restic.MountAll(ctx, cfg); err != nil {
		return sdk.Version{}, nil, err
	}
	defer restic.UnmountAll(cfg)

	if err := restic.InitIfNeeded(ctx, cfg.Repository, cfg.Password); err != nil {
		return sdk.Version{}, nil, err
	}

	action := d.Action
	if params.Action != "" {
		action = params.Action
	}

	var metadata sdk.Metadata

	switch action {
	case "backup":
		paths := []string{}
		for _, dir := range params.Directories {
			paths = append(paths, filepath.Join(cfg.MountPathSource, dir))
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
		if source.KeepDaily > 0 {
			args = append(args, "--keep-daily", fmt.Sprintf("%d", source.KeepDaily))
		} else {
			args = append(args, "--keep-daily", "7")
		}
		if source.KeepWeekly > 0 {
			args = append(args, "--keep-weekly", fmt.Sprintf("%d", source.KeepWeekly))
		} else {
			args = append(args, "--keep-weekly", "4")
		}
		if source.KeepMonthly > 0 {
			args = append(args, "--keep-monthly", fmt.Sprintf("%d", source.KeepMonthly))
		} else {
			args = append(args, "--keep-monthly", "12")
		}
		if source.KeepYearly > 0 {
			args = append(args, "--keep-yearly", fmt.Sprintf("%d", source.KeepYearly))
		} else {
			args = append(args, "--keep-yearly", "3")
		}
		args = append(args, "--prune")

		sdk.Log("Starting Restic prune (forget)...")
		out, err := restic.RunRestic(ctx, cfg.Password, args...)
		if err != nil {
			return sdk.Version{}, nil, err
		}
		metadata = append(metadata, sdk.MetadataItem{Name: "output", Value: string(out)})
	}

	v := fmt.Sprintf("%d", time.Now().Unix())
	return sdk.Version{Ref: v}, metadata, nil
}
