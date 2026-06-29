package resource

type Source struct {
	Expression string `json:"expression"`
	Location   string `json:"location"`
}


type InParams any
type OutParams any

type Driver struct{}
