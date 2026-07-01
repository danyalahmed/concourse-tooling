package main

import (
	sdk "github.com/danyalahmed/concourse-resource-sdk"
	"restic-resource/internal/resource"
)

func main() {
	sdk.RunCommand(&resource.Driver{Action: "restore"})
}
