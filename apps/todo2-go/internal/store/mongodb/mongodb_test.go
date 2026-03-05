package mongodb

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMongoDBStore_Integration(t *testing.T) {
	uri := os.Getenv("TEST_MONGO_URI")
	if uri == "" {
		t.Skip("TEST_MONGO_URI not set, skipping MongoDB integration test")
	}
	prevURI := os.Getenv("MONGO_URI")
	prevDB := os.Getenv("MONGO_DATABASE")
	defer func() {
		os.Setenv("MONGO_URI", prevURI)
		os.Setenv("MONGO_DATABASE", prevDB)
	}()
	os.Setenv("MONGO_URI", uri)
	if os.Getenv("MONGO_DATABASE") == "" {
		os.Setenv("MONGO_DATABASE", "todolist")
	}

	ctx := context.Background()
	ready := make(chan struct{})
	s := NewStore(ctx, func() { close(ready) })
	<-ready
	defer s.Close()

	item, err := s.Create(ctx, "integration test item")
	require.NoError(t, err)
	require.NotEmpty(t, item.ID)

	got, err := s.GetByID(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, item.ID, got.ID)
	assert.Equal(t, "integration test item", got.Description)
	assert.False(t, got.Completed)

	list, err := s.GetByCompleted(ctx, false)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)

	err = s.Update(ctx, item.ID, true)
	require.NoError(t, err)
	got, err = s.GetByID(ctx, item.ID)
	require.NoError(t, err)
	assert.True(t, got.Completed)

	err = s.Delete(ctx, item.ID)
	require.NoError(t, err)
	_, err = s.GetByID(ctx, item.ID)
	assert.Error(t, err)
}
