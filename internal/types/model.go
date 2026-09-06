package types

type Model struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	ContextSize int64  `json:"context_size"`
}
