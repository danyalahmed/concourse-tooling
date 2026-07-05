package resource

import sdk "github.com/danyalahmed/concourse-resource-sdk"

type Source struct {
	sdk.SMBSourceLegacy
	Watch string `json:"watch,omitempty"`
}


type InParams struct {
	File string `json:"file"`
}

type OutParams struct {
	File string `json:"file"`
	Dest string `json:"dest"`
}

type Driver struct{}

