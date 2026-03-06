package store

import (
	"context"
	"errors"

	"github.com/weshayutin/todo2-go/internal/model"
)

// Sentinel errors for store implementations.
var (
	ErrNotReady = errors.New("database not ready")
	ErrNotFound = errors.New("not found")
)

// TodoStore abstracts persistence for todo items. Both MariaDB and MongoDB implementations satisfy this interface.
type TodoStore interface {
	Create(ctx context.Context, description string) (*model.TodoItem, error)
	GetByCompleted(ctx context.Context, completed bool) ([]*model.TodoItem, error)
	GetByID(ctx context.Context, id string) (*model.TodoItem, error)
	Update(ctx context.Context, id string, completed bool) error
	Delete(ctx context.Context, id string) error
	Ping(ctx context.Context) error
	Close() error
}
