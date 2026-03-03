package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/weshayutin/todo2-go/internal/model"
	"github.com/weshayutin/todo2-go/internal/store"
)

// TodoResponse is the JSON shape for the existing UI (uses Id, Description, Completed).
type TodoResponse struct {
	Id          string `json:"Id"`
	Description string `json:"Description"`
	Completed   bool   `json:"Completed"`
}

func modelToResponse(t *model.TodoItem) TodoResponse {
	return TodoResponse{Id: t.ID, Description: t.Description, Completed: t.Completed}
}

// Handler holds the store and optional dbReady for health.
type Handler struct {
	Store   store.TodoStore
	DBReady func() bool
}

// Healthz returns liveness; body indicates db state.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	dbState := "connecting"
	if h.DBReady != nil && h.DBReady() {
		dbState = "ready"
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"alive": true, "db": dbState})
}

// Readyz returns 200 only when DB ping succeeds.
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if err := h.Store.Ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "database not available"})
		return
	}
	w.WriteHeader(http.StatusOK)
}

// CreateItem handles POST /todo. Form field: description.
// Returns array of one item for UI compatibility (result[0].Id).
func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "Bad Request", "invalid form", err)
		return
	}
	description := r.FormValue("description")
	if description == "" {
		writeError(w, http.StatusBadRequest, "Bad Request", "description cannot be empty", nil)
		return
	}
	item, err := h.Store.Create(r.Context(), description)
	if err != nil {
		if errors.Is(err, store.ErrNotReady) {
			writeError(w, http.StatusServiceUnavailable, "Service Unavailable", "database not available", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "failed to create item", err)
		return
	}
	logrus.WithFields(logrus.Fields{"description": description, "id": item.ID}).Info("Add new TodoItem")
	w.Header().Set("Content-Type", "application/json")
	// UI expects array with one element, result[0].Id
	_ = json.NewEncoder(w).Encode([]TodoResponse{modelToResponse(item)})
}

// UpdateItem handles POST /todo/{id}. Form field: completed (true|false).
func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		writeError(w, http.StatusBadRequest, "Bad Request", "id required", nil)
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "Bad Request", "invalid form", err)
		return
	}
	completedStr := r.FormValue("completed")
	completed := completedStr == "true" || completedStr == "1"
	err := h.Store.Update(r.Context(), id, completed)
	if err != nil {
		if errors.Is(err, store.ErrNotReady) {
			writeError(w, http.StatusServiceUnavailable, "Service Unavailable", "database not available", err)
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Not Found", "record not found", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "failed to update", err)
		return
	}
	logrus.WithFields(logrus.Fields{"id": id, "completed": completed}).Info("Updating TodoItem")
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"updated": true}`)
}

// DeleteItem handles DELETE /todo/{id}.
func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		writeError(w, http.StatusBadRequest, "Bad Request", "id required", nil)
		return
	}
	err := h.Store.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotReady) {
			writeError(w, http.StatusServiceUnavailable, "Service Unavailable", "database not available", err)
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Not Found", "record not found", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "failed to delete", err)
		return
	}
	logrus.WithField("id", id).Info("Deleting TodoItem")
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"deleted": true}`)
}

// GetCompletedItems handles GET /todo-completed.
func (h *Handler) GetCompletedItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.GetByCompleted(r.Context(), true)
	if err != nil {
		if errors.Is(err, store.ErrNotReady) {
			writeError(w, http.StatusServiceUnavailable, "Service Unavailable", "database not available", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "failed to list", err)
		return
	}
	resp := make([]TodoResponse, len(items))
	for i := range items {
		resp[i] = modelToResponse(items[i])
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetIncompleteItems handles GET /todo-incomplete.
func (h *Handler) GetIncompleteItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.GetByCompleted(r.Context(), false)
	if err != nil {
		if errors.Is(err, store.ErrNotReady) {
			writeError(w, http.StatusServiceUnavailable, "Service Unavailable", "database not available", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "failed to list", err)
		return
	}
	resp := make([]TodoResponse, len(items))
	for i := range items {
		resp[i] = modelToResponse(items[i])
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetItem handles GET /todo/{id}.
func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		writeError(w, http.StatusBadRequest, "Bad Request", "id required", nil)
		return
	}
	item, err := h.Store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotReady) {
			writeError(w, http.StatusServiceUnavailable, "Service Unavailable", "database not available", err)
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Not Found", "record not found", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "failed to get item", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(modelToResponse(item))
}

// Home serves index.html from the given dir (e.g. "web").
func (h *Handler) Home(w http.ResponseWriter, r *http.Request, staticDir string) {
	w.Header().Set("Content-Type", "text/html")
	http.ServeFile(w, r, path.Join(staticDir, "index.html"))
}

// GetLogFile serves the log file.
func (h *Handler) GetLogFile(w http.ResponseWriter, r *http.Request, logPath string) {
	http.ServeFile(w, r, logPath)
}

// Favicon serves favicon.ico from staticDir.
func (h *Handler) Favicon(w http.ResponseWriter, r *http.Request, staticDir string) {
	http.ServeFile(w, r, path.Join(staticDir, "favicon.ico"))
}

func writeError(w http.ResponseWriter, code int, errLabel, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	body := map[string]interface{}{"error": errLabel, "message": message, "code": code}
	_ = json.NewEncoder(w).Encode(body)
	if err != nil {
		logrus.WithError(err).WithField("code", code).Warn(message)
	}
}
