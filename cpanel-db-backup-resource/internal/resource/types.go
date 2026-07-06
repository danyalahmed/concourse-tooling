package resource

import sdk "github.com/danyalahmed/concourse-resource-sdk"

type Source struct {
	sdk.SSHSource
}

type InParams struct {
	Engine    string   `json:"engine"`    // "mysql" or "postgres"
	DBUser    string   `json:"db_user"`   // Database admin user
	DBPass    string   `json:"db_pass"`   // Database admin password
	Databases []string `json:"databases"` // Explicit list of databases to dump
}

type OutParams = InParams

type Driver struct{}
