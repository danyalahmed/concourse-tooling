package sdk

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHSource represents common SSH configuration for Concourse resources.
type SSHSource struct {
	Host             string `json:"host"`
	Username         string `json:"username"`
	Port             int    `json:"port,omitempty"`
	SSHKey           string `json:"ssh_key"`
	SSHKeyPassphrase string `json:"ssh_key_passphrase,omitempty"`
}

// SSHConnect establishes an SSH connection. Pass an empty passphrase for unencrypted keys.
func SSHConnect(ctx context.Context, host string, port int, username, sshKey, passphrase string) (*ssh.Client, error) {
	var signer ssh.Signer
	var err error

	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(sshKey), []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey([]byte(sshKey))
	}
	if err != nil {
		return nil, fmt.Errorf("parsing SSH key: %w", err)
	}

	if port == 0 {
		port = 22
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	config := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH dial failed: %w", err)
	}
	return conn, nil
}

// ShellQuote escapes a string for use in a shell command.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// ExecuteStream runs a remote command and streams stdout/stderr to the provided writers.
// It respects context cancellation by signaling the remote process.
func ExecuteStream(ctx context.Context, client *ssh.Client, cmd string, stdout, stderr io.Writer) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdout = stdout
	session.Stderr = stderr

	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	select {
	case <-ctx.Done():
		// Attempt to kill the remote process
		_ = session.Signal(ssh.SIGKILL)
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// ExecuteCommand runs a remote command and returns its stdout and stderr.
func ExecuteCommand(ctx context.Context, client *ssh.Client, cmd string) ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer
	err := ExecuteStream(ctx, client, cmd, &stdout, &stderr)
	return stdout.Bytes(), stderr.Bytes(), err
}

// PrepareSSHKeyFile writes the SSH key to a temporary file and decrypts it if a passphrase is provided.
// This is useful for external commands like sshfs that require a key file.
// It returns the path to the key file and a cleanup function.
func PrepareSSHKeyFile(ctx context.Context, sshKey, passphrase string) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", "ssh-key")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp file: %w", err)
	}
	keyPath := tmpFile.Name()
	tmpFile.Close()
	cleanup := func() { os.Remove(keyPath) }

	cleanedKey := strings.ReplaceAll(sshKey, "\r", "")
	cleanedKey = strings.TrimSpace(cleanedKey) + "\n"

	if err := os.WriteFile(keyPath, []byte(cleanedKey), 0600); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("writing ssh key: %w", err)
	}

	if passphrase != "" {
		Log("Decrypting SSH key...")
		cmd := exec.CommandContext(ctx, "ssh-keygen", "-p", "-P", passphrase, "-N", "", "-f", keyPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("decrypting ssh key: %w (output: %s)", err, string(out))
		}
	}

	return keyPath, cleanup, nil
}
