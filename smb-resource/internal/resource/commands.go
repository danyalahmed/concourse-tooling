package resource

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

func (d *Driver) Check(ctx context.Context, source Source, version *sdk.Version) ([]sdk.Version, error) {
	conn, session, share, err := sdk.SMBConnect(ctx, source.Host, source.Port, source.Username, source.Password, source.Share)
	if err != nil {
		return nil, err
	}
	defer sdk.SMBCleanup(conn, session, share)

	watchPath := "."
	if source.Watch != "" {
		watchPath = sdk.ToSMBPath(source.Watch)
	}

	latestModTime, err := latestMtime(ctx, share, watchPath)
	if err != nil {
		return nil, fmt.Errorf("scanning watch path %s: %w", watchPath, err)
	}
	if latestModTime.IsZero() {
		return []sdk.Version{}, nil
	}

	vStr := fmt.Sprintf("%d", latestModTime.UnixNano())
	latest := sdk.Version{Ref: vStr}

	if version == nil || version.Ref == "" || version.Ref != vStr {
		return []sdk.Version{latest}, nil
	}
	return []sdk.Version{}, nil
}

func latestMtime(ctx context.Context, share interface{ ReadDir(string) ([]os.FileInfo, error) }, path string) (time.Time, error) {
	entries, err := share.ReadDir(path)
	if err != nil {
		return time.Time{}, fmt.Errorf("reading directory %s: %w", path, err)
	}

	var latest time.Time
	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}

		childPath := path + "\\" + name

		if entry.IsDir() {
			childMtime, err := latestMtime(ctx, share, childPath)
			if err != nil {
				return time.Time{}, err
			}
			if childMtime.After(latest) {
				latest = childMtime
			}
		} else {
			if entry.ModTime().After(latest) {
				latest = entry.ModTime()
			}
		}
	}
	return latest, nil
}

func (d *Driver) In(ctx context.Context, source Source, version sdk.Version, params InParams, targetDir string) (sdk.Version, sdk.Metadata, error) {
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

func (d *Driver) Out(ctx context.Context, source Source, params OutParams, sourceDir string) (sdk.Version, sdk.Metadata, error) {
	if params.File == "" {
		mountPath := "/mnt/smb_exploration"
		if err := sdk.MountSMB(ctx, source.Host, source.Username, source.Password, source.Share, mountPath); err != nil {
			return sdk.Version{}, nil, err
		}
		defer sdk.Unmount(mountPath)

		sdk.Logf("Exploring SMB mount at %s...", mountPath)
		out, _ := exec.Command("ls", "-R", mountPath).CombinedOutput()
		sdk.Log(string(out))

		return sdk.Version{Ref: fmt.Sprintf("%d", time.Now().Unix())}, sdk.Metadata{{Name: "exploration", Value: "completed"}}, nil
	}

	localPath := filepath.Join(sourceDir, params.File)
	localStat, err := os.Stat(localPath)
	if err != nil {
		return sdk.Version{}, nil, fmt.Errorf("failed to stat local path %s: %w", localPath, err)
	}

	conn, session, share, err := sdk.SMBConnect(ctx, source.Host, source.Port, source.Username, source.Password, source.Share)
	if err != nil {
		return sdk.Version{}, nil, err
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
			return sdk.Version{}, nil, fmt.Errorf("failed to upload directory: %w", err)
		}
		v := fmt.Sprintf("%d", time.Now().UnixNano())
		return sdk.Version{Ref: v}, sdk.Metadata{
			{Name: "type", Value: "directory"},
			{Name: "path", Value: remoteBase},
		}, nil
	}

	parent := remoteParent(remoteBase)
	if parent != "" {
		if err := share.MkdirAll(parent, 0755); err != nil {
			return sdk.Version{}, nil, fmt.Errorf("failed to create remote directories: %w", err)
		}
	}

	sdk.Logf("Uploading file: %s", localPath)
	sha, size, err := sdk.UploadFile(share, localPath, remoteBase)
	if err != nil {
		return sdk.Version{}, nil, err
	}

	remoteStat, err := share.Stat(remoteBase)
	if err != nil {
		return sdk.Version{}, nil, fmt.Errorf("failed to stat uploaded file: %w", err)
	}
	v := fmt.Sprintf("%d", remoteStat.ModTime().UnixNano())

	return sdk.Version{Ref: v}, sdk.Metadata{
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
