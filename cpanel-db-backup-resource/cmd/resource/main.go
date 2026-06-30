package main

import (
	"github.com/danyalahmed/concourse-resource-sdk"
	"cpanel-db-backup-resource/internal/resource"
)

func main() {
	sdk.RunCommand(&resource.Driver{})
}
