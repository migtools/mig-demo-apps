// MIT License

// Copyright (c) 2020 Mohamad Fadhil

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB
var dbReady atomic.Bool

// getDSNFromEnv returns user, password, database, host from env with fallbacks.
func getDSNFromEnv() (user, password, database, host string) {
	user = os.Getenv("MYSQL_USER")
	if user == "" {
		user = "changeme"
	}
	password = os.Getenv("MYSQL_PASSWORD")
	if password == "" {
		password = "changeme"
	}
	database = os.Getenv("MYSQL_DATABASE")
	if database == "" {
		database = "todolist"
	}
	host = os.Getenv("MYSQL_HOST")
	if host == "" {
		host = "mysql"
	}
	return user, password, database, host
}

// buildDSN builds a MySQL DSN with timeouts (5s connect, 10s read/write).
func buildDSN(host, user, password, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s&readTimeout=10s&writeTimeout=10s",
		user, password, host, database)
}

// connectWithRetry tries to connect to MariaDB with exponential backoff (1s to 30s cap).
// On success it runs migrations, sets dbReady, and configures the connection pool.
func connectWithRetry() {
	user, password, database, host := getDSNFromEnv()
	remoteDSN := buildDSN(host, user, password, database)
	localDSN := buildDSN("127.0.0.1", user, password, database)

	backoff := time.Second
	const maxBackoff = 30 * time.Second
	attempt := 0

	for {
		attempt++
		log.Infof("Database connection attempt %d (remote: %s@tcp(%s:3306)/%s)", attempt, user, host, database)

		var conn *gorm.DB
		var err error
		conn, err = gorm.Open(mysql.Open(remoteDSN), &gorm.Config{})
		if err != nil {
			log.Warnf("Remote connection failed: %v", err)
			conn, err = gorm.Open(mysql.Open(localDSN), &gorm.Config{})
		}
		if err != nil {
			log.Warnf("Local connection also failed: %v; retrying in %v", err, backoff)
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}

		sqlDB, err := conn.DB()
		if err != nil {
			log.Warnf("Failed to get underlying *sql.DB: %v; retrying in %v", err, backoff)
			time.Sleep(backoff)
			continue
		}
		if err := sqlDB.Ping(); err != nil {
			log.Warnf("Ping failed: %v; retrying in %v", err, backoff)
			sqlDB.Close()
			time.Sleep(backoff)
			continue
		}

		// Configure connection pool
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)

		db = conn
		if err := db.Migrator().CreateTable(&TodoItemModel{}); err != nil {
			log.Warnf("CreateTable failed (table may already exist): %v", err)
		}
		dbReady.Store(true)
		log.Info("Database connected successfully; dbReady set")
		return
	}
}

// requireDB returns false if the database is not ready and writes 503 to w.
func requireDB(w http.ResponseWriter) bool {
	if !dbReady.Load() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, `{"error": "database not available"}`)
		return false
	}
	return true
}

type TodoItemModel struct {
	Id          int `gorm:"primary_key"`
	Description string
	Completed   bool
}

func CreateItem(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	description := r.FormValue("description")
	log.WithFields(log.Fields{"description": description}).Info("Add new TodoItem. Saving to database.")
	todo := &TodoItemModel{Description: description, Completed: false}
	db.Create(&todo)
	//result := db.Last(&todo).Value
	//ModelTodo := &TodoItemModel{}
	var ModelTodo []TodoItemModel
	db.Debug().Last(&ModelTodo).Scan(&ModelTodo)
	//log.Info(result.Statement.Dest)
	log.Info("New Id of row: ", ModelTodo[0].Id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&ModelTodo)
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	// Get URL parameter from mux
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	log.Info("ID of item update from HTTP Request, ID:", id)

	// Test if the TodoItem exist in DB
	err := GetItemByID(id)
	if err == false || id == 0 {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"updated": false, "error": "Record Not Found"}`)
	} else {
		completed, _ := strconv.ParseBool(r.FormValue("completed"))
		log.WithFields(log.Fields{"Id": id, "Completed": completed}).Info("Updating TodoItem")
		todo := &TodoItemModel{}
		db.First(&todo, id)
		todo.Completed = completed
		db.Save(&todo)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"updated": true}`)
	}
}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	// Get URL parameter from mux
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	// Test if the TodoItem exist in DB
	err := GetItemByID(id)
	if err == false {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"deleted": false, "error": "Record Not Found"}`)
	} else {
		log.WithFields(log.Fields{"Id": id}).Info("Deleting TodoItem")
		todo := &TodoItemModel{}
		db.First(&todo, id)
		db.Delete(&todo)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"deleted": true}`)
	}
}

func GetItemByID(Id int) bool {
	//todo := &TodoItemModel{}
	var todo []TodoItemModel
	result := db.Debug().First(&todo, Id)
	if result.Error != nil {
		log.Error("TodoItem not found in database")
		return false
	}
	return true
}

func GetCompletedItems(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	log.Info("Get completed TodoItems")
	completedTodoItems := GetTodoItems(true)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(completedTodoItems)
}

func GetIncompleteItems(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	log.Info("Get Incomplete TodoItems")
	IncompleteTodoItems := GetTodoItems(false)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(IncompleteTodoItems)
}

func GetTodoItems(completed bool) interface{} {
	var todos []TodoItemModel
	//var print_results []map[string]interface{}
	db.Debug().Where("completed = ?", completed).Find(&todos).Scan(&todos)
	//DebugTodoItems := db.Raw("SELECT * FROM `todo_item_models` WHERE completed = false").Scan(&todos)
	log.Info(&todos)
	return &todos
}

func Healthz(w http.ResponseWriter, r *http.Request) {
	if !dbReady.Load() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, `{"alive": true, "database": "connecting"}`)
		return
	}
	log.Info("API Health is OK")
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"alive": true}`)
}

func Home(w http.ResponseWriter, r *http.Request) {
	log.Info("Get index.html")
	p := path.Dir("index.html")
	// set header
	w.Header().Set("Content-type", "text/html")
	http.ServeFile(w, r, p)
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetReportCaller(true)
}

func GetLogFile(w http.ResponseWriter, r *http.Request) {
	// if file not found we simply get a 404
	filename := "/tmp/log/todoapp/app.log"
	http.ServeFile(w, r, filename)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "favicon.ico")
}

func main() {
	// logging to volume
	if _, err := os.Stat("/tmp/log/todoapp"); os.IsNotExist(err) {
		os.MkdirAll("/tmp/log/todoapp", 0700)
	}
	f, err := os.OpenFile("/tmp/log/todoapp/app.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	// if directory or volume is not mounted, do not exit
	if err != nil {
		fmt.Println("Failed to create logfile" + "logrus.txt")
		logrus.Info("Failed: log file /tmp/log/todoapp/app.log create failed")
		f.Close()
	} else {
		defer f.Close()
		multi := io.MultiWriter(f, os.Stdout)
		logrus.SetOutput(multi)
		logrus.Info("Success: Attached volume and redirected logs to /tmp/log/todoapp/app.log")
	}

	go connectWithRetry()

	fs := http.FileServer(http.Dir("./resources/"))

	log.Info("Starting Todolist API server")
	router := mux.NewRouter()
	router.PathPrefix("/resources/").Handler(http.StripPrefix("/resources/", fs))
	router.HandleFunc("/favicon.ico", faviconHandler)
	router.HandleFunc("/", Home).Methods("GET")
	router.HandleFunc("/healthz", Healthz).Methods("GET")
	router.HandleFunc("/log", GetLogFile).Methods("GET")
	router.HandleFunc("/todo-completed", GetCompletedItems).Methods("GET")
	router.HandleFunc("/todo-incomplete", GetIncompleteItems).Methods("GET")
	router.HandleFunc("/todo", CreateItem).Methods("POST")
	router.HandleFunc("/todo/{id}", UpdateItem).Methods("POST")
	router.HandleFunc("/todo/{id}", DeleteItem).Methods("DELETE")

	handler := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "DELETE", "PATCH", "OPTIONS"},
	}).Handler(router)

	http.ListenAndServe(":8000", handler)
}
