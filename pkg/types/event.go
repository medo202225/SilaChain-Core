package types

type Event struct {
	Address string            `json:"address"`
	Name    string            `json:"name"`
	Topics  []string          `json:"topics,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
}
