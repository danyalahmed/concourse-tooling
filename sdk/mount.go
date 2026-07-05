package sdk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// MountSSHFS mounts a remote directory via SSHFS.
func MountSSHFS(ctx context.Context, host string, port int, username, keyPath, remotePath, localPath string) error {
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("mkdir local mount point: %w", err)
	}

	if port == 0 {
		port = 22
	}

	sshAddr := fmt.Sprintf("%s@%s:%s", username, host, remotePath)
	args := []string{
		"-p", fmt.Sprintf("%d", port),
		"-o", fmt.Sprintf("IdentityFile=%s", keyPath),
		"-o", "StrictHostKeyChecking=no",
		"-o", "allow_other",
		sshAddr, localPath,
	}

	Logf("Mounting SSHFS %s to %s...", sshAddr, localPath)
	cmd := exec.CommandContext(ctx, "sshfs", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount sshfs: %w (output: %s)", err, string(out))
	}

	return nil
}

// MountSMB mounts a remote SMB share using the cifs kernel module.
func MountSMB(ctx context.Context, host string, username, password, share, localPath string) error {
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("mkdir local mount point: %w", err)
	}

	addr := fmt.Sprintf("//%s/%s", host, share)
	opts := fmt.Sprintf("username=%s,password=%s,vers=3.0", username, password)

	Logf("Mounting SMB %s to %s...", addr, localPath)
	cmd := exec.CommandContext(ctx, "mount", "-t", "cifs", addr, localPath, "-o", opts)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount smb: %w (output: %s)", err, string(out))
	}
	return nil
}

// Unmount unmounts a filesystem, lazily if needed.
func Unmount(localPath string) error {
	return exec.Command("umount", "-l", localPath).Run()
}
