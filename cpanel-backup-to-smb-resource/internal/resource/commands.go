package resource

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

func (d *Driver) Check(ctx context.Context, source Source, version *Version) ([]Version, error) {
	v := fmt.Sprintf("%d", time.Now().Unix())
	return []Version{{Version: v}}, nil
}

func (d *Driver) In(ctx context.Context, source Source, version Version, params InParams, targetDir string) (Version, sdk.Metadata, error) {
	if params.ParentDir == "" {
		return version, nil, fmt.Errorf("params.parent_dir is required")
	}
	if len(params.Directories) == 0 {
		return version, nil, fmt.Errorf("params.directories must include at least one entry")
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

func (d *Driver) Out(ctx context.Context, source Source, params OutParams, sourceDir string) (Version, sdk.Metadata, error) {
	return Version{Version: "noop"}, nil, nil
}

