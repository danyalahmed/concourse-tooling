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

	sdk.Logf("Preparing backup directory: %s", backupDir)
	if err := share.MkdirAll(backupDir, 0755); err != nil {
		return sdk.Version{}, nil, fmt.Errorf("mkdir on SMB: %w", err)
	}

	dbs, err := resolveDatabases(ctx, sshClient, source, params)
	if err != nil {
		return sdk.Version{}, nil, err
	}

	var backedUp []string
	for _, db := range dbs {
		user, pass := resolveCredentials(source, params, db)
		dest := sdk.ToSMBPath(fmt.Sprintf("%s/%s.sql.gz", backupDir, db))

		sdk.Logf("Backing up %s...", db)
		if err := streamDatabase(ctx, sshClient, share, dest, user, pass, db); err != nil {
			sdk.Logf("Warning: %s failed: %v", db, err)
			continue
		}
		backedUp = append(backedUp, db)
	}

	sdk.Log("Applying retention policy...")
	if err := applyDBRetention(share, sdk.ToSMBPath(params.ParentDir), source); err != nil {
		sdk.Logf("Warning: Retention failed: %v", err)
	}

	return sdk.Version{Ref: fmt.Sprintf("%d", now.Unix())}, sdk.Metadata{
		{Name: "backup_dir", Value: backupDir},
		{Name: "databases", Value: strings.Join(backedUp, ", ")},
		{Name: "timestamp", Value: now.Format(time.RFC3339)},
	}, nil
}

func resolveDatabases(ctx context.Context, sshClient *ssh.Client, source Source, params InParams) ([]string, error) {
	if !params.AllDBs {
		var dbs []string
		for db := range params.Databases {
			dbs = append(dbs, db)
		}
		return dbs, nil
	}

	sdk.Log("Fetching all databases...")
	cmd := fmt.Sprintf("MYSQL_PWD=%s mysql -u %s -e 'SHOW DATABASES;' -N", sdk.ShellQuote(source.AdminMySQLPassword), sdk.ShellQuote(source.Username))
	out, stderr, err := sdk.ExecuteCommand(ctx, sshClient, cmd)
	if err != nil {
		return nil, fmt.Errorf("list dbs: %w (stderr: %s)", err, string(stderr))
	}

	var filtered []string
	for _, line := range strings.Split(string(out), "\n") {
		db := strings.TrimSpace(line)
		switch db {
		case "", "information_schema", "performance_schema", "mysql", "sys":
			continue
		default:
			filtered = append(filtered, db)
		}
	}
	return filtered, nil
}

func resolveCredentials(source Source, params InParams, db string) (string, string) {
	user, pass := source.Username, source.AdminMySQLPassword
	if creds, ok := params.Databases[db]; ok {
		if creds.Username != "" {
			user = creds.Username
		}
		if creds.Password != "" {
			pass = creds.Password
		}
	}
	return user, pass
}

func streamDatabase(ctx context.Context, sshClient *ssh.Client, share *smb2.Share, dest, user, password, db string) error {
	cmd := fmt.Sprintf("MYSQL_PWD=%s mysqldump -u %s %s | gzip", sdk.ShellQuote(password), sdk.ShellQuote(user), sdk.ShellQuote(db))

	f, err := share.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create SMB file: %w", err)
	}
	defer f.Close()

	if err := sdk.ExecuteStream(ctx, sshClient, cmd, f, os.Stderr); err != nil {
		return fmt.Errorf("mysqldump stream: %w", err)
	}

	stat, err := share.Stat(dest)
	if err != nil || stat.Size() == 0 {
		return fmt.Errorf("verification failed: empty or missing file")
	}
	return nil
}
