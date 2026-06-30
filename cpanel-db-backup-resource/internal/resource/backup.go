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

func runBackup(ctx context.Context, sshClient *ssh.Client, share *smb2.Share, source Source, params InParams) (sdk.Version, sdk.Metadata, error) {
	now := time.Now()
	dateStamp := now.Format("2006-01-02")
	backupDir := sdk.ToSMBPath(fmt.Sprintf("%s/%s", params.ParentDir, dateStamp))

	sdk.Logf("Creating backup directory on SMB: %s", backupDir)
	if err := share.MkdirAll(backupDir, 0755); err != nil {
		return sdk.Version{}, nil, fmt.Errorf("creating backup directory on SMB: %w", err)
	}

	var dbsToBackup []string
	if params.AllDBs {
		// List all databases accessible by admin
		sdk.Log("Fetching list of all databases...")
		cmd := fmt.Sprintf("MYSQL_PWD=%s mysql -u %s -e 'SHOW DATABASES;' -N", sdk.ShellQuote(source.AdminMySQLPassword), sdk.ShellQuote(source.Username))
		stdout, stderr, err := sdk.ExecuteCommand(ctx, sshClient, cmd)
		if err != nil {
			return sdk.Version{}, nil, fmt.Errorf("listing databases failed: %w (stderr: %s)", err, string(stderr))
		}
		lines := strings.Split(string(stdout), "\n")
		for _, line := range lines {
			db := strings.TrimSpace(line)
			if db != "" && db != "information_schema" && db != "performance_schema" && db != "mysql" && db != "sys" {
				dbsToBackup = append(dbsToBackup, db)
			}
		}
	} else {
		for db := range params.Databases {
			dbsToBackup = append(dbsToBackup, db)
		}
	}

	if len(dbsToBackup) == 0 {
		sdk.Log("No databases found to backup.")
	}

	var backedUpDBs []string
	for _, db := range dbsToBackup {
		user := source.Username
		pass := source.AdminMySQLPassword

		if creds, ok := params.Databases[db]; ok {
			if creds.Username != "" {
				user = creds.Username
			}
			if creds.Password != "" {
				pass = creds.Password
			}
		}

		smbFilePath := sdk.ToSMBPath(fmt.Sprintf("%s/%s.sql.gz", backupDir, db))
		sdk.Logf("Streaming database %s to %s...", db, smbFilePath)

		if err := streamDatabase(ctx, sshClient, share, smbFilePath, user, pass, db); err != nil {
			sdk.Logf("Warning: Failed to backup database %s: %v", db, err)
			continue
		}
		backedUpDBs = append(backedUpDBs, db)
	}

	// Apply retention
	sdk.Log("Applying GFS retention policy for databases...")
	if err := applyDBRetention(share, sdk.ToSMBPath(params.ParentDir), source); err != nil {
		sdk.Logf("Warning: Database retention failed: %v", err)
	}

	v := fmt.Sprintf("%d", now.Unix())
	return sdk.Version{Ref: v}, sdk.Metadata{
		{Name: "backup_dir", Value: backupDir},
		{Name: "databases", Value: strings.Join(backedUpDBs, ", ")},
		{Name: "timestamp", Value: now.Format(time.RFC3339)},
	}, nil
}

func streamDatabase(ctx context.Context, sshClient *ssh.Client, share *smb2.Share, dbFilePath, user, password, db string) error {
	// Use gzip for database dumps as they are highly compressible
	cmd := fmt.Sprintf("MYSQL_PWD=%s mysqldump -u %s %s | gzip", sdk.ShellQuote(password), sdk.ShellQuote(user), sdk.ShellQuote(db))

	smbFile, err := share.OpenFile(dbFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating SMB file %s: %w", dbFilePath, err)
	}
	defer smbFile.Close()

	if err := sdk.ExecuteStream(ctx, sshClient, cmd, smbFile, os.Stderr); err != nil {
		return fmt.Errorf("streaming mysqldump command failed: %w", err)
	}

	// Verification
	stat, err := share.Stat(dbFilePath)
	if err != nil {
		return fmt.Errorf("verification failed for %s: %w", dbFilePath, err)
	}
	if stat.Size() == 0 {
		return fmt.Errorf("verification failed: %s is empty", dbFilePath)
	}
	sdk.Logf("Successfully verified database backup: %s (%d bytes)", dbFilePath, stat.Size())

	return nil
}
