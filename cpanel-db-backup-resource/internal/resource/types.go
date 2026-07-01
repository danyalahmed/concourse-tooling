package resource

type DBCredentials struct {
	Username string `json:"user"`
	Password string `json:"pass"`
}

type Source struct {
	Host             string `json:"host"`
	Username         string `json:"username"` // Default SSH/DB user
	Port             int    `json:"port,omitempty"`
	SSHKey           string `json:"ssh_key"`
	SSHKeyPassphrase string `json:"ssh_key_passphrase,omitempty"`
	AdminMySQLPassword string `json:"admin_mysql_password"`
	SMBHost          string `json:"smb_host"`
	SMBPort          int    `json:"smb_port,omitempty"`
	SMBUsername      string `json:"smb_username"`
	SMBPassword      string `json:"smb_password"`
	SMBShare         string `json:"smb_share"`

	// Retention policy
	KeepDaily   int `json:"keep_daily,omitempty"`
	KeepWeekly  int `json:"keep_weekly,omitempty"`
	KeepMonthly int `json:"keep_monthly,omitempty"`
	KeepYearly  int `json:"keep_yearly,omitempty"`
}

type InParams struct {
	ParentDir string                    `json:"parent_dir"`
	Databases map[string]DBCredentials `json:"databases,omitempty"` // Map of db_name -> credentials
	AllDBs    bool                      `json:"all_dbs,omitempty"`   // If true, backup all accessible DBs
}

type OutParams = InParams // Reuse InParams for Out

type Driver struct{}
