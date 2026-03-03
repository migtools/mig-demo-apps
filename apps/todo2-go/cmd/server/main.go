package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/weshayutin/todo2-go/internal/api"
	"github.com/weshayutin/todo2-go/internal/store"
	"github.com/weshayutin/todo2-go/internal/store/mariadb"
	"github.com/weshayutin/todo2-go/internal/store/mongodb"
)

func main() {
	// Logging
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		if level, err := logrus.ParseLevel(lvl); err == nil {
			logrus.SetLevel(level)
		}
	}
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetReportCaller(true)

	// Optional file log
	logDir := "/tmp/log/todoapp"
	if err := os.MkdirAll(logDir, 0755); err == nil {
		if f, err := os.OpenFile(filepath.Join(logDir, "app.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err == nil {
			defer f.Close()
			logrus.SetOutput(io.MultiWriter(f, os.Stdout))
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var st store.TodoStore
	backend := os.Getenv("DB_BACKEND")
	if backend == "" {
		backend = "mariadb"
	}

	var dbReady atomic.Bool
	onReady := func() { dbReady.Store(true) }

	switch backend {
	case "mariadb":
		st = mariadb.NewStore(ctx, onReady)
	case "mongodb":
		st = mongodb.NewStore(ctx, onReady)
	default:
		logrus.Fatalf("unknown DB_BACKEND: %q", backend)
	}
	defer func() {
		if st != nil {
			_ = st.Close()
		}
	}()

	staticDir := "web"
	if d := os.Getenv("STATIC_DIR"); d != "" {
		staticDir = d
	}
	logPath := filepath.Join(logDir, "app.log")
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8000"
	}

	h := &api.Handler{
		Store:   st,
		DBReady: func() bool { return dbReady.Load() },
	}

	router := mux.NewRouter()
	router.PathPrefix("/resources/").Handler(http.StripPrefix("/resources/", http.FileServer(http.Dir(filepath.Join(staticDir, "resources")))))
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { h.Home(w, r, staticDir) }).Methods(http.MethodGet)
	router.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) { h.Favicon(w, r, staticDir) }).Methods(http.MethodGet)
	router.HandleFunc("/healthz", h.Healthz).Methods(http.MethodGet)
	router.HandleFunc("/readyz", h.Readyz).Methods(http.MethodGet)
	router.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) { h.GetLogFile(w, r, logPath) }).Methods(http.MethodGet)
	router.HandleFunc("/todo-incomplete", h.GetIncompleteItems).Methods(http.MethodGet)
	router.HandleFunc("/todo-completed", h.GetCompletedItems).Methods(http.MethodGet)
	router.HandleFunc("/todo/{id}", h.GetItem).Methods(http.MethodGet)
	router.HandleFunc("/todo", h.CreateItem).Methods(http.MethodPost)
	router.HandleFunc("/todo/{id}", h.UpdateItem).Methods(http.MethodPost)
	router.HandleFunc("/todo/{id}", h.DeleteItem).Methods(http.MethodDelete)

	handler := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "DELETE", "PATCH", "OPTIONS"},
	}).Handler(router)

	srv := &http.Server{Addr: ":" + port, Handler: handler}
	go func() {
		logrus.Infof("Starting Todolist API server on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logrus.Info("Shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}
