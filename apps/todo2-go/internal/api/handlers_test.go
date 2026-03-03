package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weshayutin/todo2-go/internal/model"
	"github.com/weshayutin/todo2-go/internal/store"
)

func TestHealthz(t *testing.T) {
	h := &Handler{Store: NewMockStore(nil)}
	h.DBReady = func() bool { return true }

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.Healthz(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.True(t, body["alive"].(bool))
	assert.Equal(t, "ready", body["db"])
}

func TestHealthz_Connecting(t *testing.T) {
	h := &Handler{Store: NewMockStore(nil)}
	h.DBReady = func() bool { return false }

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.Healthz(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "connecting", body["db"])
}

func TestReadyz_Ready(t *testing.T) {
	mock := NewMockStore(nil)
	h := &Handler{Store: mock}

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	h.Readyz(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestReadyz_NotReady(t *testing.T) {
	mock := NewMockStore(nil)
	mock.PingErr = store.ErrNotReady
	h := &Handler{Store: mock}

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	h.Readyz(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestCreateItem_Success(t *testing.T) {
	mock := NewMockStore(nil)
	h := &Handler{Store: mock}
	h.DBReady = func() bool { return true }

	req := httptest.NewRequest(http.MethodPost, "/todo", bytes.NewBufferString("description=hello"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.CreateItem(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var arr []TodoResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&arr))
	require.Len(t, arr, 1)
	assert.Equal(t, "hello", arr[0].Description)
	assert.False(t, arr[0].Completed)
	assert.NotEmpty(t, arr[0].Id)
}

func TestCreateItem_EmptyDescription(t *testing.T) {
	h := &Handler{Store: NewMockStore(nil)}

	req := httptest.NewRequest(http.MethodPost, "/todo", bytes.NewBufferString("description="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.CreateItem(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateItem_DBNotReady(t *testing.T) {
	mock := NewMockStore(nil)
	mock.NotReady = true
	h := &Handler{Store: mock}

	req := httptest.NewRequest(http.MethodPost, "/todo", bytes.NewBufferString("description=hi"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.CreateItem(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestGetIncompleteItems_Success(t *testing.T) {
	mock := NewMockStore([]*model.TodoItem{
		{ID: "1", Description: "a", Completed: false},
		{ID: "2", Description: "b", Completed: false},
	})
	h := &Handler{Store: mock}

	req := httptest.NewRequest(http.MethodGet, "/todo-incomplete", nil)
	rec := httptest.NewRecorder()
	h.GetIncompleteItems(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var arr []TodoResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&arr))
	assert.Len(t, arr, 2)
}

func TestGetCompletedItems_Success(t *testing.T) {
	mock := NewMockStore([]*model.TodoItem{
		{ID: "1", Description: "done", Completed: true},
	})
	h := &Handler{Store: mock}

	req := httptest.NewRequest(http.MethodGet, "/todo-completed", nil)
	rec := httptest.NewRecorder()
	h.GetCompletedItems(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var arr []TodoResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&arr))
	require.Len(t, arr, 1)
	assert.True(t, arr[0].Completed)
}

func TestGetItem_Success(t *testing.T) {
	mock := NewMockStore([]*model.TodoItem{
		{ID: "1", Description: "item", Completed: false},
	})
	h := &Handler{Store: mock}

	req := httptest.NewRequest(http.MethodGet, "/todo/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	rec := httptest.NewRecorder()
	h.GetItem(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var item TodoResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&item))
	assert.Equal(t, "1", item.Id)
	assert.Equal(t, "item", item.Description)
}

func TestGetItem_NotFound(t *testing.T) {
	mock := NewMockStore(nil)
	h := &Handler{Store: mock}

	req := httptest.NewRequest(http.MethodGet, "/todo/999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	rec := httptest.NewRecorder()
	h.GetItem(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateItem_Success(t *testing.T) {
	mock := NewMockStore([]*model.TodoItem{
		{ID: "1", Description: "x", Completed: false},
	})
	h := &Handler{Store: mock}

	req := httptest.NewRequest(http.MethodPost, "/todo/1", bytes.NewBufferString("completed=true"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	rec := httptest.NewRecorder()
	h.UpdateItem(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"updated": true`)
}

func TestUpdateItem_NotFound(t *testing.T) {
	mock := NewMockStore(nil)
	h := &Handler{Store: mock}

	req := httptest.NewRequest(http.MethodPost, "/todo/999", bytes.NewBufferString("completed=true"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	rec := httptest.NewRecorder()
	h.UpdateItem(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteItem_Success(t *testing.T) {
	mock := NewMockStore([]*model.TodoItem{
		{ID: "1", Description: "x", Completed: false},
	})
	h := &Handler{Store: mock}

	req := httptest.NewRequest(http.MethodDelete, "/todo/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	rec := httptest.NewRecorder()
	h.DeleteItem(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"deleted": true`)

	_, err := mock.GetByID(context.Background(), "1")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestDeleteItem_NotFound(t *testing.T) {
	mock := NewMockStore(nil)
	h := &Handler{Store: mock}

	req := httptest.NewRequest(http.MethodDelete, "/todo/999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	rec := httptest.NewRecorder()
	h.DeleteItem(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
