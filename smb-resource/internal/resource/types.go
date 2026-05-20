package resource

type Source struct {
	Host     string `json:"host"`
	Share    string `json:"share,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	Port     int    `json:"port,omitempty"`
}

type Version struct {
	Version string `json:"version"`
}

type InParams struct {
	File string `json:"file"`
}

type OutParams struct {
	File string `json:"file"`
	Dest string `json:"dest"`
}

type Metadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Driver struct{}
