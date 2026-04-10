package block

type ExecutionResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
