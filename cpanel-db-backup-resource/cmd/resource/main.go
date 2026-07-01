package main

import (
	"cpanel-db-backup-resource/internal/resource"
	"github.com/danyalahmed/concourse-resource-sdk"
)

func main() {
	sdk.RunCommand(&resource.Driver{})
}
