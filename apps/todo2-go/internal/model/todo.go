package model

// TodoItem is the shared todo item representation used across API and store implementations.
// ID is a string to bridge MySQL integer primary keys and MongoDB ObjectID (hex) without leaking backend types.
type TodoItem struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}
