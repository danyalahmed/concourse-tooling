package resource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (d *Driver) Check(ctx context.Context, source Source, version *Version) ([]Version, error) {
	v := fmt.Sprintf("%d", time.Now().Unix())
	return []Version{{Version: v}}, nil
}

func (d *Driver) In(ctx context.Context, source Source, version Version, req InRequest, targetDir string) (Version, Metadata, error) {
	client := NewGitHubClient()
	token, err := client.GenerateInstallationToken(req.Source)
	if err != nil {
		return Version{}, Metadata{}, fmt.Errorf("generating token: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return Version{}, Metadata{}, fmt.Errorf("creating directory: %w", err)
	}

	tokenPath := filepath.Join(targetDir, "token")
	if err := os.WriteFile(tokenPath, []byte(token), 0o600); err != nil {
		return Version{}, Metadata{}, fmt.Errorf("writing token file: %w", err)
	}

	v := fmt.Sprintf("%d", time.Now().Unix())
	return Version{Version: v}, Metadata{
		Name: "app_id", Value: req.Source.AppID,
	}, nil
}

func (d *Driver) Out(ctx context.Context, source Source, _ OutRequest, _ string) (Version, Metadata, error) {
	return Version{Version: "noop"}, Metadata{}, nil
}
