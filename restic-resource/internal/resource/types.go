package resource

type Source struct {
	Host             string `json:"host"`
	Username         string `json:"username"`
	Port             int    `json:"port,omitempty"`
	SSHKey           string `json:"ssh_key"`
	SSHKeyPassphrase string `json:"ssh_key_passphrase,omitempty"`
	SMBHost          string `json:"smb_host"`
	SMBPort          int    `json:"smb_port,omitempty"`
	SMBUsername      string `json:"smb_username"`
	SMBPassword      string `json:"smb_password"`
	SMBShare         string `json:"smb_share"`
	RepositoryPath   string `json:"repository_path"` // Path inside SMB share
	RepositoryPass   string `json:"repository_pass"`

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
