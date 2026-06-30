package resource

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

func (d *Driver) Check(ctx context.Context, source Source, version *sdk.Version) ([]sdk.Version, error) {
	v := fmt.Sprintf("%d", time.Now().Unix())
	return []sdk.Version{{Ref: v}}, nil
}

func (d *Driver) In(ctx context.Context, source Source, version sdk.Version, params InParams, targetDir string) (sdk.Version, sdk.Metadata, error) {
	if params.ParentDir == "" {
		return version, nil, fmt.Errorf("params.parent_dir is required")
	}
	if len(params.Directories) == 0 && !params.DBOnly {
		return version, nil, fmt.Errorf("params.directories must include at least one entry when db_only is false")
	}
	if params.DBOnly && params.SkipDB {
		return version, nil, fmt.Errorf("db_only and skip_db cannot both be true")
	}

	sdk.Logf("Connecting to SSH host %s...", source.Host)
	sshClient, err := sdk.SSHConnect(ctx, source.Host, source.Port, source.Username, source.SSHKey, source.SSHKeyPassphrase)
	if err != nil {
		return version, nil, fmt.Errorf("SSH connection failed: %w", err)
	}
	defer sshClient.Close()

	sdk.Logf("Connecting to SMB share %s on %s...", source.SMBShare, source.SMBHost)
	conn, session, share, err := sdk.SMBConnect(ctx, source.SMBHost, source.SMBPort, source.SMBUsername, source.SMBPassword, source.SMBShare)
	if err != nil {
		return version, nil, fmt.Errorf("SMB connection failed: %w", err)
	}
	defer sdk.SMBCleanup(conn, session, share)

	return runBackup(ctx, sshClient, share, source, params)
}

func (d *Driver) Out(ctx context.Context, source Source, params OutParams, sourceDir string) (sdk.Version, sdk.Metadata, error) {
	return sdk.Version{Ref: "noop"}, nil, nil
}

