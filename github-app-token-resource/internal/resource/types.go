package resource

import (
	"fmt"
)

type Source struct {
	AppID          string `json:"app_id"`
	InstallationID string `json:"installation_id"`
	PrivateKey     string `json:"private_key"`
}

func (s Source) GoString() string {
	return fmt.Sprintf("resource.Source{AppID: %q, InstallationID: %q, PrivateKey: <redacted>}", s.AppID, s.InstallationID)
}

type Driver struct{}

