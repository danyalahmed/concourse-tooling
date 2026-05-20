package resource

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cloudsoda/go-smb2"
)

func (d *Driver) Check(ctx context.Context, source Source, version *Version) ([]Version, error) {
	conn, session, share, err := d.connect(ctx, source)
	if err != nil {
		return nil, err
	}
	defer cleanup(share, session, conn)

	files, err := share.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("reading share directory failed: %w", err)
	}

	type versionItem struct {
		id      string
		modTime time.Time
	}
	var foundVersions []versionItem
	for _, file := range files {
		if file.IsDir() {
			vStr := fmt.Sprintf("%s-%d", file.Name(), file.ModTime().Unix())
			foundVersions = append(foundVersions, versionItem{
				id:      vStr,
				modTime: file.ModTime(),
			})
		}
	}

	sort.Slice(foundVersions, func(i, j int) bool {
		return foundVersions[i].modTime.Before(foundVersions[j].modTime)
	})

	if len(foundVersions) == 0 {
		return []Version{}, nil
	}

	latest := Version{Version: foundVersions[len(foundVersions)-1].id}
	if version == nil {
		// First run: return only the single latest version
		return []Version{latest}, nil
	}

	// Subsequent runs: Find where the current 'version' sits in history
	var result []Version
	foundCurrentIndex := -1
	for i, fv := range foundVersions {
		if fv.id == version.Version {
			foundCurrentIndex = i
			break
		}
	}

	if foundCurrentIndex == -1 {
		return []Version{latest}, nil
	}

	for _, fv := range foundVersions[foundCurrentIndex:] {
		result = append(result, Version{Version: fv.id})
	}

	return result, nil
}

func (d *Driver) In(ctx context.Context, source Source, version Version, params InParams, targetDir string) (Version, []Metadata, error) {
	if params.File == "" {
		return version, []Metadata{}, fmt.Errorf("params.file must be specified in the get step")
	}

	// Parse the version string to find the exact historical directory name
	var remoteDir string
	if idx := strings.LastIndex(version.Version, "-"); idx != -1 {
		remoteDir = version.Version[:idx]
	} else {
		// Fallback or error if the version string is malformed
		return version, []Metadata{}, fmt.Errorf("invalid version format received: %s", version.Version)
	}

	conn, session, share, err := d.connect(ctx, source)
	if err != nil {
		return version, []Metadata{}, err
	}
	defer cleanup(share, session, conn)

	cleanFile := strings.Trim(params.File, "/\\")
	remotePath := remoteDir + "\\" + strings.ReplaceAll(cleanFile, "/", "\\")
	remoteFile, err := share.Open(remotePath)
	if err != nil {
		return version, []Metadata{}, fmt.Errorf("failed to open remote file %s: %w", remotePath, err)
	}
	defer remoteFile.Close()

	stat, err := remoteFile.Stat()
	if err != nil {
		return version, []Metadata{}, fmt.Errorf("failed to stat remote file: %w", err)
	}

	// Create the destination folder context locally if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return version, []Metadata{}, fmt.Errorf("failed to create target directory: %w", err)
	}

	localPath := filepath.Join(targetDir, filepath.Base(params.File))
	localFile, err := os.Create(localPath)
	if err != nil {
		return version, []Metadata{}, fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(localFile, hasher)

	if _, err := io.Copy(multiWriter, remoteFile); err != nil {
		return version, []Metadata{}, fmt.Errorf("failed to stream copy data: %w", err)
	}

	meta := []Metadata{
		{Name: "filename", Value: stat.Name()},
		{Name: "size_bytes", Value: fmt.Sprintf("%d", stat.Size())},
		{Name: "sha256", Value: fmt.Sprintf("%x", hasher.Sum(nil))},
	}

	return version, meta, nil
}

func (d *Driver) Out(ctx context.Context, source Source, params OutParams, targetDir string) (Version, []Metadata, error) {
	if params.File == "" {
		return Version{}, []Metadata{}, fmt.Errorf("params.file must be specified in the put step")
	}

	localPath := filepath.Join(targetDir, params.File)
	localFile, err := os.Open(localPath)
	if err != nil {
		return Version{}, []Metadata{}, fmt.Errorf("failed to open local file %s: %w", localPath, err)
	}
	defer localFile.Close()

	stat, err := localFile.Stat()
	if err != nil {
		return Version{}, []Metadata{}, fmt.Errorf("failed to stat local file: %w", err)
	}

	if stat.IsDir() {
		return Version{}, []Metadata{}, fmt.Errorf("params.file points to a directory; uploading directories is not supported directly")
	}

	conn, session, share, err := d.connect(ctx, source)
	if err != nil {
		return Version{}, []Metadata{}, err
	}
	defer cleanup(share, session, conn)

	cleanFileName := filepath.Base(params.File)
	folderBaseName := strings.TrimSuffix(cleanFileName, filepath.Ext(cleanFileName))
	if folderBaseName == "" {
		folderBaseName = "build" // Absolute fallback safety mechanism
	}

	remoteDir := fmt.Sprintf("%s-%d", folderBaseName, time.Now().Unix())

	err = share.Mkdir(remoteDir, 0755)
	if err != nil {
		return Version{}, []Metadata{}, fmt.Errorf("failed to create remote directory %s: %w", remoteDir, err)
	}

	remotePath := remoteDir + "\\" + cleanFileName
	remoteFile, err := share.Create(remotePath)
	if err != nil {
		return Version{}, []Metadata{}, fmt.Errorf("failed to create remote file %s: %w", remotePath, err)
	}
	defer remoteFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(remoteFile, hasher)

	if _, err := io.Copy(multiWriter, localFile); err != nil {
		return Version{}, []Metadata{}, fmt.Errorf("failed to upload data to SMB share: %w", err)
	}

	newVersionStr := remoteDir

	meta := []Metadata{
		{Name: "filename", Value: cleanFileName},
		{Name: "size_bytes", Value: fmt.Sprintf("%d", stat.Size())},
		{Name: "sha256", Value: fmt.Sprintf("%x", hasher.Sum(nil))},
		{Name: "remote_path", Value: remotePath},
	}

	return Version{Version: newVersionStr}, meta, nil
}

func cleanup(share *smb2.Share, session *smb2.Session, conn net.Conn) {
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
