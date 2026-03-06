package api

import (
	"context"
	"strconv"
	"sync"

	"github.com/weshayutin/todo2-go/internal/model"
	"github.com/weshayutin/todo2-go/internal/store"
)

// MockStore is an in-memory TodoStore for tests.
type MockStore struct {
	mu     sync.Mutex
	items  map[string]*model.TodoItem
	nextID int
	// If non-nil, Ping returns this error (e.g. store.ErrNotReady).
	PingErr error
	// If true, all methods return store.ErrNotReady.
	NotReady bool
}

// NewMockStore returns a MockStore with optional initial items.
func NewMockStore(initial []*model.TodoItem) *MockStore {
	m := &MockStore{items: make(map[string]*model.TodoItem), nextID: 1}
	if initial != nil {
		maxID := 0
		for _, t := range initial {
			m.items[t.ID] = t
			if id, err := strconv.Atoi(t.ID); err == nil && id >= maxID {
				maxID = id + 1
			}
		}
		if maxID > 0 {
			m.nextID = maxID
		} else {
			m.nextID = len(initial) + 1
		}
	}
	return m
}

func (m *MockStore) requireReady() error {
	if m.NotReady {
		return store.ErrNotReady
	}
	return nil
}

func (m *MockStore) Create(ctx context.Context, description string) (*model.TodoItem, error) {
	if err := m.requireReady(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id := strconv.Itoa(m.nextID)
	m.nextID++
	item := &model.TodoItem{ID: id, Description: description, Completed: false}
	m.items[id] = item
	return item, nil
}

func (m *MockStore) GetByCompleted(ctx context.Context, completed bool) ([]*model.TodoItem, error) {
	if err := m.requireReady(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*model.TodoItem
	for _, t := range m.items {
		if t.Completed == completed {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *MockStore) GetByID(ctx context.Context, id string) (*model.TodoItem, error) {
	if err := m.requireReady(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.items[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return t, nil
}

func (m *MockStore) Update(ctx context.Context, id string, completed bool) error {
	if err := m.requireReady(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.items[id]
	if !ok {
		return store.ErrNotFound
	}
	t.Completed = completed
	return nil
}

func (m *MockStore) Delete(ctx context.Context, id string) error {
	if err := m.requireReady(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return store.ErrNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *MockStore) Ping(ctx context.Context) error {
	if m.PingErr != nil {
		return m.PingErr
	}
	return m.requireReady()
}

func (m *MockStore) Close() error {
	return nil
}

var _ store.TodoStore = (*MockStore)(nil)
