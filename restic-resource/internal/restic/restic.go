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
	SSHKeyPassphrase string
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

	keyPath := cfg.SSHKeyPath
	if cfg.SSHKeyPassphrase != "" {
		// Use expect or a temporary decrypted key for sshfs.
		// Since we are in a container, we can decrypt the key to a temporary file.
		decryptedKeyPath := cfg.SSHKeyPath + ".decrypted"
		sdk.Log("Decrypting SSH key for mounting...")

		// Use ssh-keygen to remove the passphrase and save to a new file
		// ssh-keygen -p -P <passphrase> -N "" -f <key>
		// Note: This modifies the file in place or requires interaction usually.
		// Better: openssl or just use a tool that supports passphrases.
		// Actually, we can use a temporary file and 'ssh-keygen -p' but it's tricky.
		// Alternatives: use 'ssh-agent' or 'expect'.
		// A simpler way: decrypt using 'openssl' or 'ssh-keygen -p' equivalent.
		// Let's try: cp key to temp, then ssh-keygen -p -P <pass> -N "" -f <temp>

		if err := copyFile(cfg.SSHKeyPath, decryptedKeyPath); err != nil {
			return fmt.Errorf("preparing decrypted key: %w", err)
		}
		if err := os.Chmod(decryptedKeyPath, 0600); err != nil {
			return err
		}

		// Try to decrypt. ssh-keygen -p -P <old> -N <new> -f <file>
		decryptCmd := exec.CommandContext(ctx, "ssh-keygen", "-p", "-P", cfg.SSHKeyPassphrase, "-N", "", "-f", decryptedKeyPath)
		if out, err := decryptCmd.CombinedOutput(); err != nil {
			_ = os.Remove(decryptedKeyPath)
			return fmt.Errorf("decrypting ssh key failed: %w (output: %s)", err, string(out))
		}
		keyPath = decryptedKeyPath
		defer os.Remove(decryptedKeyPath)
	}

	sdk.Logf("Mounting SSHFS %s to %s", sshAddr, cfg.MountPathSource)
	sshfsArgs := []string{
		"-p", fmt.Sprintf("%d", sshPort),
		"-o", fmt.Sprintf("IdentityFile=%s", keyPath),
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

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0600)
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
