package resource

import sdk "github.com/danyalahmed/concourse-resource-sdk"

type Source struct {
	sdk.SSHSource
	sdk.SMBSource

	RepositoryPath string `json:"repository_path"` // Path inside SMB share
	RepositoryPass string `json:"repository_pass"`

	// Retention defaults
	KeepDaily   int `json:"keep_daily,omitempty"`
	KeepWeekly  int `json:"keep_weekly,omitempty"`
	KeepMonthly int `json:"keep_monthly,omitempty"`
	KeepYearly  int `json:"keep_yearly,omitempty"`
}

type InParams struct {
	Restore      bool   `json:"restore,omitempty"`
	SnapshotID   string `json:"snapshot_id,omitempty"`
	TargetSubDir string `json:"target_subdir,omitempty"` // Path inside cPanel to restore to
}

type OutParams struct {
	Action      string   `json:"action,omitempty"` // "backup", "prune", or empty (default backup)
	Directories []string `json:"directories,omitempty"`
	Excludes    []string `json:"excludes,omitempty"`
}
