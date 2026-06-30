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
	// In is now a no-op for consistency with restic-resource backup
	return version, nil, nil
}

func (d *Driver) Out(ctx context.Context, source Source, params InParams, sourceDir string) (sdk.Version, sdk.Metadata, error) {
	if params.ParentDir == "" {
		return sdk.Version{}, nil, fmt.Errorf("params.parent_dir is required")
	}

	sdk.Logf("Connecting to SSH host %s...", source.Host)
	sshClient, err := sdk.SSHConnect(ctx, source.Host, source.Port, source.Username, source.SSHKey, source.SSHKeyPassphrase)
	if err != nil {
		return sdk.Version{}, nil, fmt.Errorf("SSH connection failed: %w", err)
	}
	defer sshClient.Close()

	sdk.Logf("Connecting to SMB share %s on %s...", source.SMBShare, source.SMBHost)
	conn, session, share, err := sdk.SMBConnect(ctx, source.SMBHost, source.SMBPort, source.SMBUsername, source.SMBPassword, source.SMBShare)
	if err != nil {
		return sdk.Version{}, nil, fmt.Errorf("SMB connection failed: %w", err)
	}
	defer sdk.SMBCleanup(conn, session, share)

	return runBackup(ctx, sshClient, share, source, params)
}
