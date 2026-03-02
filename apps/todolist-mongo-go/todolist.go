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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var db *mongo.Client
var tododb *mongo.Collection

type TodoItemModel struct {
	Id          primitive.ObjectID `bson:"_id,omitempty"`
	Description string
	Completed   bool
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// writeErrorResponse writes a standardized error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   errorMsg,
		Message: message,
		Code:    statusCode,
	}

	json.NewEncoder(w).Encode(response)
}

// writeSuccessResponse writes a standardized success response
func writeSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")

	response := SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
}

// panicRecoveryMiddleware recovers from panics and returns proper HTTP responses
func panicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("Panic recovered: %v", err)
				writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// connectToDB attempts to connect to the local MongoDB instance with retries.
// Both MongoDB and the Go app run in the same container, so we always connect
// to 127.0.0.1:27017. Credentials match the MONGO_INITDB_ROOT_USERNAME /
// MONGO_INITDB_ROOT_PASSWORD env vars used by the entrypoint script.
func connectToDB() {
	for i := 0; i < 30; i++ {
		client, err := connectToMongoLocal()
		if err == nil {
			// Verify the connection is actually usable
			pingErr := client.Ping(context.TODO(), nil)
			if pingErr == nil {
				log.Info("Successfully connected to MongoDB")
				db = client
				return
			}
			log.Warnf("Ping failed on attempt %d/30: %v", i+1, pingErr)
			client.Disconnect(context.TODO())
		} else {
			log.Warnf("Connection attempt %d/30 failed, retrying in 2s...", i+1)
		}
		time.Sleep(2 * time.Second)
	}
	log.Fatal("Failed to connect to MongoDB after 30 attempts")
}

// connectToMongoLocal connects to MongoDB at 127.0.0.1 (local, same container)
func connectToMongoLocal() (*mongo.Client, error) {
	log.Info("Attempting to connect to: mongodb://changeme:changeme@localhost:27017")
	clientOptions := options.Client().
		ApplyURI("mongodb://changeme:changeme@localhost:27017").
		SetWriteConcern(writeconcern.New(writeconcern.W(1), writeconcern.J(true)))
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Errorf("Connection failed: %v", err)
		return nil, err
	}
	return client, nil
}

func CreateItem(w http.ResponseWriter, r *http.Request) {
	description := r.FormValue("description")

	// Validate input
	if description == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Bad Request", "Description cannot be empty")
		return
	}

	log.WithFields(log.Fields{"description": description}).Info("Add new TodoItem. Saving to database.")
	todo := &TodoItemModel{Description: description, Completed: false}

	result, err := tododb.InsertOne(context.TODO(), todo)
	if err != nil {
		log.Errorf("Failed to insert todo item: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error", "Failed to create todo item")
		return
	}

	id := result.InsertedID.(primitive.ObjectID)
	todo.Id = id
	log.Infof("Inserted document with ID %v", id.Hex())

	// Return the original format for backward compatibility
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
	// Get URL parameter from mux
	vars := mux.Vars(r)
	id := vars["id"]

	// Validate ObjectID format
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Bad Request", "Invalid ID format")
		return
	}

	// Test if the TodoItem exists in DB
	exists := GetItemByID(id)
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, "Not Found", "Todo item not found")
		return
	}

	// Parse completed status with proper error handling
	completedStr := r.FormValue("completed")
	completed, err := strconv.ParseBool(completedStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Bad Request", "Invalid completed value. Must be true or false")
		return
	}

	log.WithFields(log.Fields{"_id": id, "Completed": completed}).Info("Updating TodoItem")

	filter := bson.M{"_id": objID}
	updateResult, err := tododb.UpdateOne(
		context.TODO(),
		filter,
		bson.D{
			{"$set", bson.D{{"completed", completed}}},
		},
	)

	if err != nil {
		log.Errorf("Failed to update todo item: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error", "Failed to update todo item")
		return
	}

	if updateResult.ModifiedCount == 0 {
		writeErrorResponse(w, http.StatusNotFound, "Not Found", "Todo item not found or no changes made")
		return
	}

	// Return the original format for backward compatibility
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"updated": true}`)
}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
	// Get URL parameter from mux
	vars := mux.Vars(r)
	id := vars["id"]

	// Validate ObjectID format
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Bad Request", "Invalid ID format")
		return
	}

	// Test if the TodoItem exists in DB
	exists := GetItemByID(id)
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, "Not Found", "Todo item not found")
		return
	}

	log.WithFields(log.Fields{"_id": id}).Info("Deleting TodoItem")

	filter := bson.M{"_id": objID}
	opts := options.Delete().SetCollation(&options.Collation{
		Locale:    "en_US",
		Strength:  1,
		CaseLevel: false,
	})

	res, err := tododb.DeleteOne(context.TODO(), filter, opts)
	if err != nil {
		log.Errorf("Failed to delete todo item: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error", "Failed to delete todo item")
		return
	}

	if res.DeletedCount == 0 {
		writeErrorResponse(w, http.StatusNotFound, "Not Found", "Todo item not found")
		return
	}

	log.Infof("Deleted %v documents", res.DeletedCount)
	// Return the original format for backward compatibility
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"deleted": true}`)
}

func GetItemByID(Id string) bool {
	objID, err := primitive.ObjectIDFromHex(Id)
	if err != nil {
		log.Errorf("Invalid ObjectID format: %v", err)
		return false
	}

	filter := bson.M{"_id": objID}
	var result TodoItemModel
	err = tododb.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Debugf("Todo item with ID %s not found", Id)
		} else {
			log.Errorf("Database error while finding todo item: %v", err)
		}
		return false
	}

	log.Debugf("Found todo item: %+v", result)
	return true
}

func GetCompletedItems(w http.ResponseWriter, r *http.Request) {
	log.Info("Get completed TodoItems")
	completedTodoItems, err := GetTodoItems(true)
	if err != nil {
		log.Errorf("Failed to get completed todo items: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error", "Failed to retrieve completed todo items")
		return
	}
	// Return the original format for backward compatibility
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(completedTodoItems)
}

func GetIncompleteItems(w http.ResponseWriter, r *http.Request) {
	log.Info("Get Incomplete TodoItems")
	incompleteTodoItems, err := GetTodoItems(false)
	if err != nil {
		log.Errorf("Failed to get incomplete todo items: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error", "Failed to retrieve incomplete todo items")
		return
	}
	// Return the original format for backward compatibility
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(incompleteTodoItems)
}

func GetTodoItems(completed bool) ([]*TodoItemModel, error) {
	findOptions := options.Find()
	findOptions.SetLimit(50)

	var results []*TodoItemModel
	filter := bson.M{"completed": completed}

	cur, err := tododb.Find(context.TODO(), filter, findOptions)
	if err != nil {
		log.Errorf("Failed to query todo items: %v", err)
		return nil, err
	}
	defer cur.Close(context.TODO())

	// Iterate through the cursor
	for cur.Next(context.TODO()) {
		var elem TodoItemModel
		err := cur.Decode(&elem)
		if err != nil {
			log.Errorf("Failed to decode todo item: %v", err)
			return nil, err
		}

		results = append(results, &elem)
	}

	// Check for cursor errors
	if err := cur.Err(); err != nil {
		log.Errorf("Cursor error: %v", err)
		return nil, err
	}

	return results, nil
}

func Healthz(w http.ResponseWriter, r *http.Request) {
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

func prepopulate(collection *mongo.Collection) error {
	log.Info("Prepopulate the db")
	prepop := TodoItemModel{Description: "prepopulate the db", Completed: true}
	donuts := TodoItemModel{Description: "time", Completed: false}
	both_prepop := []interface{}{prepop, donuts}

	insertManyResult, err := collection.InsertMany(context.TODO(), both_prepop)
	if err != nil {
		log.Errorf("Failed to prepopulate database: %v", err)
		return err
	}
	log.Infof("Inserted multiple prepopulate documents: %v", insertManyResult.InsertedIDs)
	return nil
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

	// Connect to MongoDB (retries until ready, since mongod starts in background)
	connectToDB()

	// collection
	tododb = db.Database("todolist").Collection("TodoItemModel")
	log.Info("Connected to MongoDB!")

	fs := http.FileServer(http.Dir("./resources/"))

	log.Info("Starting Todolist API server")
	router := mux.NewRouter()
	router.PathPrefix("/resources/").Handler(http.StripPrefix("/resources/", fs))
	router.HandleFunc("/", Home).Methods("GET")
	router.HandleFunc("/favicon.ico", faviconHandler)
	router.HandleFunc("/healthz", Healthz).Methods("GET")
	router.HandleFunc("/log", GetLogFile).Methods("GET")
	router.HandleFunc("/todo-completed", GetCompletedItems).Methods("GET")
	router.HandleFunc("/todo-incomplete", GetIncompleteItems).Methods("GET")
	router.HandleFunc("/todo", CreateItem).Methods("POST")
	router.HandleFunc("/todo/{id}", UpdateItem).Methods("POST")
	router.HandleFunc("/todo/{id}", DeleteItem).Methods("DELETE")

	// Apply panic recovery middleware
	handler := panicRecoveryMiddleware(router)

	// Apply CORS
	corsHandler := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "DELETE", "PATCH", "OPTIONS"},
	}).Handler(handler)

	log.Info("Server starting on port 8000")
	if err := http.ListenAndServe(":8000", corsHandler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
