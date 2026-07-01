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
	if err := os.MkdirAll(cfg.MountPathTarget, 0755); err != nil {
		return fmt.Errorf("mkdir target: %w", err)
	}

	addr := fmt.Sprintf("//%s/%s", cfg.SMBHost, cfg.SMBShare)
	opts := fmt.Sprintf("username=%s,password=%s,vers=3.0", cfg.SMBUser, cfg.SMBPass)

	sdk.Logf("Mounting SMB %s...", addr)
	cmd := exec.CommandContext(ctx, "mount", "-t", "cifs", addr, cfg.MountPathTarget, "-o", opts)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount smb: %w (output: %s)", err, string(out))
	}
	return nil
}

func MountAll(ctx context.Context, cfg Config) error {
	if err := MountSMB(ctx, cfg); err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.MountPathSource, 0755); err != nil {
		return fmt.Errorf("mkdir source: %w", err)
	}

	sshAddr := fmt.Sprintf("%s@%s:/home/%s", cfg.SSHUser, cfg.SSHHost, cfg.SSHUser)
	sshPort := cfg.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	keyPath := cfg.SSHKeyPath
	if cfg.SSHKeyPassphrase != "" {
		decrypted := cfg.SSHKeyPath + ".decrypted"
		sdk.Log("Decrypting SSH key...")

		if err := copyFile(cfg.SSHKeyPath, decrypted); err != nil {
			return err
		}
		if err := os.Chmod(decrypted, 0600); err != nil {
			return err
		}

		cmd := exec.CommandContext(ctx, "ssh-keygen", "-p", "-P", cfg.SSHKeyPassphrase, "-N", "", "-f", decrypted)
		if out, err := cmd.CombinedOutput(); err != nil {
			_ = os.Remove(decrypted)
			return fmt.Errorf("decrypt: %w (output: %s)", err, string(out))
		}
		keyPath = decrypted
		defer os.Remove(decrypted)
	}

	sdk.Logf("Mounting SSHFS %s...", sshAddr)
	args := []string{
		"-p", fmt.Sprintf("%d", sshPort),
		"-o", fmt.Sprintf("IdentityFile=%s", keyPath),
		"-o", "StrictHostKeyChecking=no",
		"-o", "allow_other",
		sshAddr, cfg.MountPathSource,
	}
	cmd := exec.CommandContext(ctx, "sshfs", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount sshfs: %w (output: %s)", err, string(out))
	}

	return nil
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0600)
}

func UnmountAll(cfg Config) {
	_ = exec.Command("umount", "-l", cfg.MountPathSource).Run()
	_ = exec.Command("umount", "-l", cfg.MountPathTarget).Run()
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
