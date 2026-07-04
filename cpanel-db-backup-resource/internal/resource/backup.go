package resource

import (
	"context"
	"fmt"
	"strings"
	"time"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
	"golang.org/x/crypto/ssh"
)

func runBackup(ctx context.Context, sshClient *ssh.Client, source Source, params InParams) (sdk.Version, sdk.Metadata, error) {
	engine := strings.ToLower(params.Engine)
	if engine == "" {
		engine = "mysql"
	}

	backupDir := fmt.Sprintf("/home/%s/database-dumps/%s", source.Username, engine)
	sdk.Logf("Ensuring backup directory exists: %s", backupDir)

	mkdirCmd := fmt.Sprintf("mkdir -p %s", sdk.ShellQuote(backupDir))
	if _, _, err := sdk.ExecuteCommand(ctx, sshClient, mkdirCmd); err != nil {
		return sdk.Version{}, nil, fmt.Errorf("creating backup directory: %w", err)
	}

	if len(params.Databases) == 0 {
		return sdk.Version{}, nil, fmt.Errorf("no databases specified")
	}

	var backedUp []string
	var errs []string
	for _, db := range params.Databases {
		sdk.Logf("Dumping database: %s", db)
		destFile := fmt.Sprintf("%s/%s.sql", backupDir, db)

		var dumpCmd string
		switch engine {
		case "postgres":
			dumpCmd = fmt.Sprintf("PGPASSWORD=%s pg_dump -U %s -h localhost -F p %s > %s",
				sdk.ShellQuote(params.DBPass),
				sdk.ShellQuote(params.DBUser),
				sdk.ShellQuote(db),
				sdk.ShellQuote(destFile),
			)
		case "mysql":
			fallthrough
		default:
			dumpCmd = fmt.Sprintf("MYSQL_PWD=%s mysqldump -u %s %s > %s",
				sdk.ShellQuote(params.DBPass),
				sdk.ShellQuote(params.DBUser),
				sdk.ShellQuote(db),
				sdk.ShellQuote(destFile),
			)
		}

		if _, _, err := sdk.ExecuteCommand(ctx, sshClient, dumpCmd); err != nil {
			sdk.Logf("Error: dump failed for %s: %v", db, err)
			errs = append(errs, fmt.Sprintf("%s: %v", db, err))
			continue
		}
		backedUp = append(backedUp, db)
	}

	if len(errs) > 0 {
		return sdk.Version{}, nil, fmt.Errorf("failed to dump some databases: %s", strings.Join(errs, "; "))
	}

	now := time.Now()
	return sdk.Version{Ref: fmt.Sprintf("%d", now.Unix())}, sdk.Metadata{
		{Name: "backup_dir", Value: backupDir},
		{Name: "engine", Value: engine},
		{Name: "databases", Value: strings.Join(backedUp, ", ")},
		{Name: "timestamp", Value: now.Format(time.RFC3339)},
	}, nil
}
