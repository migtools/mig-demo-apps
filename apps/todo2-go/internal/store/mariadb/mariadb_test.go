package mariadb

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMariaDBStore_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN not set, skipping MariaDB integration test")
	}
	// We don't have a way to pass DSN into NewStore (it reads from env). So set env for this test.
	prevUser := os.Getenv("MYSQL_USER")
	prevPass := os.Getenv("MYSQL_PASSWORD")
	prevDB := os.Getenv("MYSQL_DATABASE")
	prevHost := os.Getenv("MYSQL_HOST")
	prevPort := os.Getenv("MYSQL_PORT")
	defer func() {
		os.Setenv("MYSQL_USER", prevUser)
		os.Setenv("MYSQL_PASSWORD", prevPass)
		os.Setenv("MYSQL_DATABASE", prevDB)
		os.Setenv("MYSQL_HOST", prevHost)
		os.Setenv("MYSQL_PORT", prevPort)
	}()
	// TEST_MYSQL_DSN format: user:password@tcp(host:port)/dbname - parse and set env
	// For simplicity, require MYSQL_* to be set when TEST_MYSQL_DSN is set, or skip.
	if os.Getenv("MYSQL_HOST") == "" {
		t.Skip("MYSQL_HOST (and other MYSQL_*) must be set when TEST_MYSQL_DSN is set for integration test")
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
	got, _ = s.GetByID(ctx, item.ID)
	assert.True(t, got.Completed)

	err = s.Delete(ctx, item.ID)
	require.NoError(t, err)
	_, err = s.GetByID(ctx, item.ID)
	assert.Error(t, err)
}
