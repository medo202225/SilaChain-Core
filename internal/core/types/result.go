package types

type Result struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
