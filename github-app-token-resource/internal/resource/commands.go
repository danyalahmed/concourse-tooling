package resource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sdk "github.com/danyalahmed/concourse-resource-sdk"
)

func (d *Driver) Check(ctx context.Context, source Source, version *Version) ([]Version, error) {
	v := fmt.Sprintf("%d", time.Now().Unix())
	return []Version{{Version: v}}, nil
}

func (d *Driver) In(ctx context.Context, source Source, version Version, _ any, targetDir string) (Version, sdk.Metadata, error) {
	client := sdk.NewGitHubClient()
	token, err := client.GenerateInstallationToken(source.AppID, source.InstallationID, source.PrivateKey)
	if err != nil {
		return Version{}, nil, fmt.Errorf("generating token: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return Version{}, nil, fmt.Errorf("creating directory: %w", err)
	}

	tokenPath := filepath.Join(targetDir, "token")
	if err := os.WriteFile(tokenPath, []byte(token), 0o600); err != nil {
		return Version{}, nil, fmt.Errorf("writing token file: %w", err)
	}

	sdk.Logf("GitHub App installation token written to %s", tokenPath)

	v := fmt.Sprintf("%d", time.Now().Unix())
	return Version{Version: v}, sdk.Metadata{
		{Name: "app_id", Value: source.AppID},
	}, nil
}

func (d *Driver) Out(ctx context.Context, source Source, _ any, _ string) (Version, sdk.Metadata, error) {
	return Version{Version: "noop"}, nil, nil
}

