package resource

type Source struct {
	Host          string `json:"host"`
	Username      string `json:"username"`
	Port          int    `json:"port,omitempty"`
	SSHKey        string `json:"ssh_key"`
	SSHKeyPassphrase string `json:"ssh_key_passphrase,omitempty"`
	MySQLPassword string `json:"mysql_password"`
	SMBHost       string `json:"smb_host"`
	SMBPort       int    `json:"smb_port,omitempty"`
	SMBUsername   string `json:"smb_username"`
	SMBPassword   string `json:"smb_password"`
	SMBShare      string `json:"smb_share"`
}

type InParams struct {
	ParentDir   string   `json:"parent_dir"`
	Directories []string `json:"directories,omitempty"`
	Excludes    []string `json:"excludes,omitempty"`
	DBOnly      bool     `json:"db_only,omitempty"`
	KeepCount   int      `json:"keep_count,omitempty"`
	KeepDays    int      `json:"keep_days,omitempty"`
}

type OutParams any

type Version struct {
	Version string `json:"version"`
}

type Driver struct{}

