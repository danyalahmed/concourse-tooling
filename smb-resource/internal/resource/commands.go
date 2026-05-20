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
	"strconv"
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
		if !file.IsDir() {
			vStr := strconv.FormatInt(file.ModTime().UnixNano(), 10)
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
	if version == nil || version.Version == "" {
		return []Version{latest}, nil
	}

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
		return version, []Metadata{}, nil
	}

	conn, session, share, err := d.connect(ctx, source)
	if err != nil {
		return version, []Metadata{}, err
	}
	defer cleanup(share, session, conn)

	remotePath := toSMBPath(params.File)

	stat, err := share.Stat(remotePath)
	if err != nil {
		return version, []Metadata{}, fmt.Errorf("failed to stat remote path %s: %w", remotePath, err)
	}

	if stat.IsDir() {
		dest := filepath.Join(targetDir, filepath.Base(params.File))
		if err := downloadDir(share, remotePath, dest); err != nil {
			return version, []Metadata{}, fmt.Errorf("failed to download directory: %w", err)
		}
		return version, []Metadata{
			{Name: "type", Value: "directory"},
			{Name: "path", Value: remotePath},
		}, nil
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return version, []Metadata{}, fmt.Errorf("failed to create target directory: %w", err)
	}

	localPath := filepath.Join(targetDir, filepath.Base(params.File))
	sha, size, err := downloadFile(share, remotePath, localPath)
	if err != nil {
		return version, []Metadata{}, err
	}

	return version, []Metadata{
		{Name: "filename", Value: filepath.Base(params.File)},
		{Name: "size_bytes", Value: fmt.Sprintf("%d", size)},
		{Name: "sha256", Value: sha},
	}, nil
}

func (d *Driver) Out(ctx context.Context, source Source, params OutParams, sourceDir string) (Version, []Metadata, error) {
	if params.File == "" {
		return Version{}, []Metadata{}, fmt.Errorf("params.file must be specified in the put step")
	}

	localPath := filepath.Join(sourceDir, params.File)
	localStat, err := os.Stat(localPath)
	if err != nil {
		return Version{}, []Metadata{}, fmt.Errorf("failed to stat local path %s: %w", localPath, err)
	}

	conn, session, share, err := d.connect(ctx, source)
	if err != nil {
		return Version{}, []Metadata{}, err
	}
	defer cleanup(share, session, conn)

	destRoot := params.Dest
	if destRoot == "" {
		destRoot = filepath.Base(params.File)
	}
	remoteBase := toSMBPath(destRoot)

	if localStat.IsDir() {
		if err := uploadDir(share, localPath, remoteBase); err != nil {
			return Version{}, []Metadata{}, fmt.Errorf("failed to upload directory: %w", err)
		}
		v := strconv.FormatInt(time.Now().UnixNano(), 10)
		return Version{Version: v}, []Metadata{
			{Name: "type", Value: "directory"},
			{Name: "path", Value: remoteBase},
		}, nil
	}

	parent := remoteParent(remoteBase)
	if parent != "" {
		if err := share.MkdirAll(parent, 0755); err != nil {
			return Version{}, []Metadata{}, fmt.Errorf("failed to create remote directories: %w", err)
		}
	}

	sha, size, err := uploadFile(share, localPath, remoteBase)
	if err != nil {
		return Version{}, []Metadata{}, err
	}

	remoteStat, err := share.Stat(remoteBase)
	if err != nil {
		return Version{}, []Metadata{}, fmt.Errorf("failed to stat uploaded file: %w", err)
	}
	v := strconv.FormatInt(remoteStat.ModTime().UnixNano(), 10)

	return Version{Version: v}, []Metadata{
		{Name: "filename", Value: filepath.Base(destRoot)},
		{Name: "size_bytes", Value: fmt.Sprintf("%d", size)},
		{Name: "sha256", Value: sha},
		{Name: "remote_path", Value: remoteBase},
	}, nil
}

// --- helpers ---

func toSMBPath(p string) string {
	return strings.ReplaceAll(strings.Trim(p, "/\\"), "/", "\\")
}

func remoteParent(p string) string {
	idx := strings.LastIndex(p, "\\")
	if idx <= 0 {
		return ""
	}
	return p[:idx]
}

func downloadFile(share *smb2.Share, remotePath, localPath string) (sha256Hex string, size int64, err error) {
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

func uploadFile(share *smb2.Share, localPath, remotePath string) (sha256Hex string, size int64, err error) {
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

func downloadDir(share *smb2.Share, remoteDir, localDir string) error {
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
			if err := downloadDir(share, childRemote, childLocal); err != nil {
				return err
			}
		} else {
			if _, _, err := downloadFile(share, childRemote, childLocal); err != nil {
				return err
			}
		}
	}
	return nil
}

func uploadDir(share *smb2.Share, localDir, remoteDir string) error {
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
			if err := uploadDir(share, childLocal, childRemote); err != nil {
				return err
			}
		} else {
			if _, _, err := uploadFile(share, childLocal, childRemote); err != nil {
				return err
			}
		}
	}
	return nil
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
