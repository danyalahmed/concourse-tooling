package sdk

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

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

