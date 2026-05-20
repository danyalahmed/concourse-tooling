package main

import (
	sdk "github.com/danyalahmed/concourse-resource-sdk"
	"github.com/danyalahmed/cpanel-backup-to-smb-resource/internal/resource"
)

func main() {
	sdk.RunCommand(&resource.Driver{})
}
