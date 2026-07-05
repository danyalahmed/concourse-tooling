package restic

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

type Config struct {
	Repository       string
	Password         string
	SSHHost          string
	SSHUser          string
	SSHPort          int
	SSHKeyPath       string
	SSHKeyPassphrase string
	SMBHost          string
	SMBShare         string
	SMBUser          string
	SMBPass          string
	MountPathSource  string
	MountPathTarget  string
}

func MountSMB(ctx context.Context, cfg Config) error {
	return sdk.MountSMB(ctx, cfg.SMBHost, cfg.SMBUser, cfg.SMBPass, cfg.SMBShare, cfg.MountPathTarget)
}

func MountAll(ctx context.Context, cfg Config) error {
	if err := MountSMB(ctx, cfg); err != nil {
		return err
	}

	remotePath := fmt.Sprintf("/home/%s", cfg.SSHUser)
	return sdk.MountSSHFS(ctx, cfg.SSHHost, cfg.SSHPort, cfg.SSHUser, cfg.SSHKeyPath, remotePath, cfg.MountPathSource)
}

func UnmountAll(cfg Config) {
	_ = sdk.Unmount(cfg.MountPathSource)
	_ = sdk.Unmount(cfg.MountPathTarget)
}

func RunRestic(ctx context.Context, password string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "restic", args...)
	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD="+password)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("restic: %w (output: %s)", err, string(out))
	}
	return out, nil
}

func InitIfNeeded(ctx context.Context, repo, password string) error {
	_, err := RunRestic(ctx, password, "-r", repo, "snapshots")
	if err == nil {
		return nil
	}

	sdk.Log("Initializing Restic repository...")
	_, err = RunRestic(ctx, password, "-r", repo, "init")
	return err
}
