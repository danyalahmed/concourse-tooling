package sdk

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudsoda/go-smb2"
)

// SMBSource represents common SMB configuration for Concourse resources.
type SMBSource struct {
	Host     string `json:"smb_host"`
	Port     int    `json:"smb_port,omitempty"`
	Username string `json:"smb_username"`
	Password string `json:"smb_password"`
	Share    string `json:"smb_share"`
}

// SMBConnect establishes a connection to an SMB share.
func SMBConnect(ctx context.Context, host string, port int, username, password, share string) (net.Conn, *smb2.Session, *smb2.Share, error) {
	if port == 0 {
		port = 445
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))

	var netDialer net.Dialer
	conn, err := netDialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("dial network failed: %w", err)
	}

	dialer := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     username,
			Password: password,
		},
	}

	session, err := dialer.DialConn(ctx, conn, addr)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("smb authentication failed: %w", err)
	}

	session = session.WithContext(ctx)

	mounted, err := session.Mount(share)
	if err != nil {
		session.Logoff()
		conn.Close()
		return nil, nil, nil, fmt.Errorf("mounting share failed: %w", err)
	}
	return conn, session, mounted, nil
}

// SMBCleanup safely closes SMB connections.
func SMBCleanup(conn net.Conn, session *smb2.Session, share *smb2.Share) {
	if share != nil {
		_ = share.Umount()
	}
	if session != nil {
		_ = session.Logoff()
	}
	if conn != nil {
		_ = conn.Close()
	}
}

// ToSMBPath converts a generic path to an SMB-compatible path (backslashes).
func ToSMBPath(p string) string {
	return strings.ReplaceAll(strings.Trim(p, "/\\"), "/", "\\")
}

// DownloadFile downloads a single file from an SMB share.
func DownloadFile(share *smb2.Share, remotePath, localPath string) (sha256Hex string, size int64, err error) {
	remoteFile, err := share.Open(remotePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open remote file %s: %w", remotePath, err)
	}
	defer remoteFile.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create local file %s: %w", localPath, err)
	}
	defer localFile.Close()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(localFile, hasher), remoteFile)
	if err != nil {
		return "", 0, fmt.Errorf("failed to copy data: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), written, nil
}

// UploadFile uploads a single file to an SMB share.
func UploadFile(share *smb2.Share, localPath, remotePath string) (sha256Hex string, size int64, err error) {
	localFile, err := os.Open(localPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open local file %s: %w", localPath, err)
	}
	defer localFile.Close()

	remoteFile, err := share.OpenFile(remotePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create remote file %s: %w", remotePath, err)
	}
	defer remoteFile.Close()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(remoteFile, hasher), localFile)
	if err != nil {
		return "", 0, fmt.Errorf("failed to upload data: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), written, nil
}

// DownloadDir recursively downloads a directory from an SMB share.
func DownloadDir(share *smb2.Share, remoteDir, localDir string) error {
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory %s: %w", localDir, err)
	}

	entries, err := share.ReadDir(remoteDir)
	if err != nil {
		return fmt.Errorf("failed to read remote directory %s: %w", remoteDir, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		childRemote := remoteDir + "\\" + name
		childLocal := filepath.Join(localDir, name)

		if entry.IsDir() {
			if err := DownloadDir(share, childRemote, childLocal); err != nil {
				return err
			}
		} else {
			if _, _, err := DownloadFile(share, childRemote, childLocal); err != nil {
				return err
			}
		}
	}
	return nil
}

// UploadDir recursively uploads a directory to an SMB share.
func UploadDir(share *smb2.Share, localDir, remoteDir string) error {
	if err := share.MkdirAll(remoteDir, 0755); err != nil {
		return fmt.Errorf("failed to create remote directory %s: %w", remoteDir, err)
	}

	entries, err := os.ReadDir(localDir)
	if err != nil {
		return fmt.Errorf("failed to read local directory %s: %w", localDir, err)
	}

	for _, entry := range entries {
		childLocal := filepath.Join(localDir, entry.Name())
		childRemote := remoteDir + "\\" + entry.Name()

		if entry.IsDir() {
			if err := UploadDir(share, childLocal, childRemote); err != nil {
				return err
			}
		} else {
			if _, _, err := UploadFile(share, childLocal, childRemote); err != nil {
				return err
			}
		}
	}
	return nil
}

