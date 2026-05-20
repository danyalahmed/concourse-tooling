package resource

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudsoda/go-smb2"
	"golang.org/x/crypto/ssh"
	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

func runBackup(ctx context.Context, sshClient *ssh.Client, share *smb2.Share, source Source, params InParams) (Version, sdk.Metadata, error) {
	now := time.Now()
	stamp := now.Format("2006-01-02_15-04-05")
	backupDir := sdk.ToSMBPath(fmt.Sprintf("%s/backup_%s", params.ParentDir, stamp))

	filesDir := backupDir + "\\files"
	dbDir := backupDir + "\\database"

	sdk.Logf("Creating backup directories on SMB: %s", backupDir)
	if err := share.MkdirAll(filesDir, 0755); err != nil {
		return Version{}, nil, fmt.Errorf("creating files directory on SMB: %w", err)
	}
	if err := share.MkdirAll(dbDir, 0755); err != nil {
		return Version{}, nil, fmt.Errorf("creating database directory on SMB: %w", err)
	}

	for _, dir := range params.Directories {
		sdk.Logf("Streaming directory %s to SMB...", dir)
		if err := streamDirectory(ctx, sshClient, share, filesDir, source.Username, dir, params.Excludes); err != nil {
			return Version{}, nil, fmt.Errorf("streaming directory %s: %w", dir, err)
		}
	}

	dbFile := dbDir + "\\all_dbs_" + now.Format("2006-01-02") + ".sql"
	sdk.Logf("Streaming all databases to SMB: %s", dbFile)
	if err := streamDatabase(ctx, sshClient, share, dbFile, source.Username, source.MySQLPassword); err != nil {
		return Version{}, nil, fmt.Errorf("streaming database: %w", err)
	}

	v := fmt.Sprintf("%d", now.Unix())
	return Version{Version: v}, sdk.Metadata{
		{Name: "backup_dir", Value: backupDir},
		{Name: "directories", Value: strings.Join(params.Directories, ", ")},
		{Name: "timestamp", Value: now.Format(time.RFC3339)},
	}, nil
}

func dirToFilename(dir string) string {
	return strings.ReplaceAll(dir, "/", "_")
}

func buildExcludesForDir(dir string, excludes []string) ([]string, bool) {
	var result []string
	for _, exc := range excludes {
		if exc == "" {
			continue
		}
		if exc == dir {
			return nil, true
		}
		if strings.HasPrefix(exc, dir+"/") {
			result = append(result, "--exclude="+strings.TrimPrefix(exc, dir+"/"))
			continue
		}
		if strings.Contains(exc, "/") {
			continue
		}
		result = append(result, "--exclude="+exc)
	}
	return result, false
}

func streamDirectory(ctx context.Context, sshClient *ssh.Client, share *smb2.Share, filesDir, remoteUser, dir string, allExcludes []string) error {
	excludes, skip := buildExcludesForDir(dir, allExcludes)
	if skip {
		sdk.Logf("Skipping directory %s (excluded)", dir)
		return nil
	}

	var cmd strings.Builder
	cmd.WriteString(fmt.Sprintf("cd /home/%s && tar -czf -", sdk.ShellQuote(remoteUser)))
	for _, exc := range excludes {
		cmd.WriteString(" ")
		cmd.WriteString(exc)
	}
	cmd.WriteString(" ")
	cmd.WriteString(sdk.ShellQuote(dir))

	smbFilePath := filesDir + "\\" + dirToFilename(dir) + ".tar.gz"
	smbFile, err := share.OpenFile(smbFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating SMB file %s: %w", smbFilePath, err)
	}
	defer smbFile.Close()

	if err := sdk.ExecuteStream(ctx, sshClient, cmd.String(), smbFile, os.Stderr); err != nil {
		return fmt.Errorf("streaming tar command failed: %w", err)
	}

	return nil
}

func streamDatabase(ctx context.Context, sshClient *ssh.Client, share *smb2.Share, dbFilePath, remoteUser, mysqlPassword string) error {
	cmd := fmt.Sprintf("MYSQL_PWD=%s mysqldump -u %s --all-databases", sdk.ShellQuote(mysqlPassword), sdk.ShellQuote(remoteUser))

	smbFile, err := share.OpenFile(dbFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating SMB file %s: %w", dbFilePath, err)
	}
	defer smbFile.Close()

	if err := sdk.ExecuteStream(ctx, sshClient, cmd, smbFile, os.Stderr); err != nil {
		return fmt.Errorf("streaming mysqldump command failed: %w", err)
	}

	return nil
}
