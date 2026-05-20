package resource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

func (d *Driver) Check(ctx context.Context, source Source, version *Version) ([]Version, error) {
	conn, session, share, err := sdk.SMBConnect(ctx, source.Host, source.Port, source.Username, source.Password, source.Share)
	if err != nil {
		return nil, err
	}
	defer sdk.SMBCleanup(conn, session, share)

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

func (d *Driver) In(ctx context.Context, source Source, version Version, params InParams, targetDir string) (Version, sdk.Metadata, error) {
	if params.File == "" {
		return version, sdk.Metadata{}, nil
	}

	conn, session, share, err := sdk.SMBConnect(ctx, source.Host, source.Port, source.Username, source.Password, source.Share)
	if err != nil {
		return version, nil, err
	}
	defer sdk.SMBCleanup(conn, session, share)

	remotePath := sdk.ToSMBPath(params.File)

	stat, err := share.Stat(remotePath)
	if err != nil {
		return version, nil, fmt.Errorf("failed to stat remote path %s: %w", remotePath, err)
	}

	if stat.IsDir() {
		sdk.Logf("Downloading directory: %s", remotePath)
		dest := filepath.Join(targetDir, filepath.Base(params.File))
		if err := sdk.DownloadDir(share, remotePath, dest); err != nil {
			return version, nil, fmt.Errorf("failed to download directory: %w", err)
		}
		return version, sdk.Metadata{
			{Name: "type", Value: "directory"},
			{Name: "path", Value: remotePath},
		}, nil
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return version, nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	sdk.Logf("Downloading file: %s", remotePath)
	localPath := filepath.Join(targetDir, filepath.Base(params.File))
	sha, size, err := sdk.DownloadFile(share, remotePath, localPath)
	if err != nil {
		return version, nil, err
	}

	return version, sdk.Metadata{
		{Name: "filename", Value: filepath.Base(params.File)},
		{Name: "size_bytes", Value: fmt.Sprintf("%d", size)},
		{Name: "sha256", Value: sha},
	}, nil
}

func (d *Driver) Out(ctx context.Context, source Source, params OutParams, sourceDir string) (Version, sdk.Metadata, error) {
	if params.File == "" {
		return Version{}, nil, fmt.Errorf("params.file must be specified in the put step")
	}

	localPath := filepath.Join(sourceDir, params.File)
	localStat, err := os.Stat(localPath)
	if err != nil {
		return Version{}, nil, fmt.Errorf("failed to stat local path %s: %w", localPath, err)
	}

	conn, session, share, err := sdk.SMBConnect(ctx, source.Host, source.Port, source.Username, source.Password, source.Share)
	if err != nil {
		return Version{}, nil, err
	}
	defer sdk.SMBCleanup(conn, session, share)

	destRoot := params.Dest
	if destRoot == "" {
		destRoot = filepath.Base(params.File)
	}
	remoteBase := sdk.ToSMBPath(destRoot)

	if localStat.IsDir() {
		sdk.Logf("Uploading directory: %s", localPath)
		if err := sdk.UploadDir(share, localPath, remoteBase); err != nil {
			return Version{}, nil, fmt.Errorf("failed to upload directory: %w", err)
		}
		v := strconv.FormatInt(time.Now().UnixNano(), 10)
		return Version{Version: v}, sdk.Metadata{
			{Name: "type", Value: "directory"},
			{Name: "path", Value: remoteBase},
		}, nil
	}

	parent := remoteParent(remoteBase)
	if parent != "" {
		if err := share.MkdirAll(parent, 0755); err != nil {
			return Version{}, nil, fmt.Errorf("failed to create remote directories: %w", err)
		}
	}

	sdk.Logf("Uploading file: %s", localPath)
	sha, size, err := sdk.UploadFile(share, localPath, remoteBase)
	if err != nil {
		return Version{}, nil, err
	}

	remoteStat, err := share.Stat(remoteBase)
	if err != nil {
		return Version{}, nil, fmt.Errorf("failed to stat uploaded file: %w", err)
	}
	v := strconv.FormatInt(remoteStat.ModTime().UnixNano(), 10)

	return Version{Version: v}, sdk.Metadata{
		{Name: "filename", Value: filepath.Base(destRoot)},
		{Name: "size_bytes", Value: fmt.Sprintf("%d", size)},
		{Name: "sha256", Value: sha},
		{Name: "remote_path", Value: remoteBase},
	}, nil
}

func remoteParent(p string) string {
	idx := strings.LastIndex(p, "\\")
	if idx <= 0 {
		return ""
	}
	return p[:idx]
}
