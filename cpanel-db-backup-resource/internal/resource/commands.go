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
	return version, nil, nil
}

func (d *Driver) Out(ctx context.Context, source Source, params InParams, sourceDir string) (sdk.Version, sdk.Metadata, error) {
	sdk.Logf("Connecting to SSH host %s...", source.Host)
	sshClient, err := sdk.SSHConnect(ctx, source.Host, source.Port, source.Username, source.SSHKey, source.SSHKeyPassphrase)
	if err != nil {
		return sdk.Version{}, nil, fmt.Errorf("SSH connection failed: %w", err)
	}
	defer sshClient.Close()

	return runBackup(ctx, sshClient, source, params)
}
