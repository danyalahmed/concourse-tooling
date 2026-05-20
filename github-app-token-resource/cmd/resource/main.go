package main

import (
	sdk "github.com/danyalahmed/concourse-resource-sdk"
	"github.com/danyalahmed/github-app-token-resource/internal/resource"
)

func main() {
	driver := &resource.Driver{}
	sdk.RunCommand(driver)
}
