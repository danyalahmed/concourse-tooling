package restic

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

type Config struct {
	Repository       string
	Password         string
	SSHHost          string
	SSHUser          string
	SSHPort          int
	SSHKeyPath       string
	SMBHost          string
	SMBShare         string
	SMBUser          string
	SMBPass          string
	MountPathSource  string
	MountPathTarget  string
}

func MountAll(ctx context.Context, cfg Config) error {
	// 1. Mount SMB
	if err := os.MkdirAll(cfg.MountPathTarget, 0755); err != nil {
		return fmt.Errorf("mkdir target: %w", err)
	}

	smbAddr := fmt.Sprintf("//%s/%s", cfg.SMBHost, cfg.SMBShare)
	opts := fmt.Sprintf("username=%s,password=%s,vers=3.0", cfg.SMBUser, cfg.SMBPass)

	sdk.Logf("Mounting SMB %s to %s", smbAddr, cfg.MountPathTarget)
	cmd := exec.CommandContext(ctx, "mount", "-t", "cifs", smbAddr, cfg.MountPathTarget, "-o", opts)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount smb failed: %w (output: %s)", err, string(out))
	}

	// 2. Mount SSHFS
	if err := os.MkdirAll(cfg.MountPathSource, 0755); err != nil {
		return fmt.Errorf("mkdir source: %w", err)
	}

	sshAddr := fmt.Sprintf("%s@%s:", cfg.SSHUser, cfg.SSHHost)
	sshPort := cfg.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	sdk.Logf("Mounting SSHFS %s to %s", sshAddr, cfg.MountPathSource)
	// sshfs -p 22 -o IdentityFile=/key -o StrictHostKeyChecking=no user@host:/ /mnt/source
	sshfsArgs := []string{
		"-p", fmt.Sprintf("%d", sshPort),
		"-o", fmt.Sprintf("IdentityFile=%s", cfg.SSHKeyPath),
		"-o", "StrictHostKeyChecking=no",
		"-o", "allow_other",
		sshAddr, cfg.MountPathSource,
	}
	cmd = exec.CommandContext(ctx, "sshfs", sshfsArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount sshfs failed: %w (output: %s)", err, string(out))
	}

	return nil
}

func UnmountAll(cfg Config) {
	sdk.Logf("Unmounting %s", cfg.MountPathSource)
	_ = exec.Command("umount", "-l", cfg.MountPathSource).Run()
	sdk.Logf("Unmounting %s", cfg.MountPathTarget)
	_ = exec.Command("umount", "-l", cfg.MountPathTarget).Run()
}

func RunRestic(ctx context.Context, password string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "restic", args...)
	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD="+password)
	sdk.Logf("Running restic %s", strings.Join(args, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("restic command failed: %w (output: %s)", err, string(out))
	}
	return out, nil
}

func InitIfNeeded(ctx context.Context, repo, password string) error {
	_, err := RunRestic(ctx, password, "-r", repo, "snapshots")
	if err == nil {
		return nil
	}

	sdk.Log("Restic repository not initialized or inaccessible. Initializing...")
	_, err = RunRestic(ctx, password, "-r", repo, "init")
	return err
}
