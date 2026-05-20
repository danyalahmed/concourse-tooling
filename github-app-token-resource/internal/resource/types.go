package resource

import "fmt"

type Source struct {
	AppID          string `json:"app_id"`
	InstallationID string `json:"installation_id"`
	PrivateKey     string `json:"private_key"`
}

func (s Source) GoString() string {
	return fmt.Sprintf("resource.Source{AppID: %q, InstallationID: %q, PrivateKey: <redacted>}", s.AppID, s.InstallationID)
}

type Version struct {
	Version string `json:"version"`
}

type CheckRequest struct {
	Source  Source   `json:"source"`
	Version *Version `json:"version"`
}

type CheckResponse []Version

type InRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
	Params  any     `json:"params"`
}

type InResponse struct {
	Version  Version    `json:"version"`
	Metadata []Metadata `json:"metadata,omitempty"`
}

type OutRequest struct {
	Source Source `json:"source"`
	Params any    `json:"params"`
}

type OutResponse struct {
	Version  Version    `json:"version"`
	Metadata []Metadata `json:"metadata,omitempty"`
}

type Metadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Driver struct{}
