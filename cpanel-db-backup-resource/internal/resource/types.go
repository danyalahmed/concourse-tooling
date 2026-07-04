package resource

type Source struct {
	Host             string `json:"host"`
	Username         string `json:"username"` // SSH username
	Port             int    `json:"port,omitempty"`
	SSHKey           string `json:"ssh_key"`
	SSHKeyPassphrase string `json:"ssh_key_passphrase,omitempty"`
}

type InParams struct {
	Engine    string   `json:"engine"`    // "mysql" or "postgres"
	DBUser    string   `json:"db_user"`   // Database admin user
	DBPass    string   `json:"db_pass"`   // Database admin password
	Databases []string `json:"databases"` // Explicit list of databases to dump
	AllDBs    bool     `json:"all_dbs"`   // If true, dump all accessible databases
}

type OutParams = InParams

type Driver struct{}
