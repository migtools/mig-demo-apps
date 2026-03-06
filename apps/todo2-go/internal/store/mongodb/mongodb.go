package mongodb

import (
	"context"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/weshayutin/todo2-go/internal/model"
	"github.com/weshayutin/todo2-go/internal/store"
)

const (
	defaultMaxBackoff    = 30 * time.Second
	defaultInitialBackoff = time.Second
	jitterPercent        = 0.1
)

// doc is the BSON document shape for the collection.
type doc struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Description string             `bson:"description"`
	Completed   bool               `bson:"completed"`
}

// Store implements store.TodoStore for MongoDB.
type Store struct {
	client   *mongo.Client
	coll     *mongo.Collection
	dbReady  atomic.Bool
	mu       sync.Mutex
	database string
}

func getMongoConfig() (uri, database string) {
	uri = os.Getenv("MONGO_URI")
	if uri == "" {
		// No credentials: local all-in-one mode starts mongod with --noauth
		uri = "mongodb://127.0.0.1:27017"
	}
	database = os.Getenv("MONGO_DATABASE")
	if database == "" {
		database = "todolist"
	}
	return uri, database
}

func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	delta := int64(float64(d) * jitterPercent)
	if delta == 0 {
		return d
	}
	j := time.Duration(rand.Int63n(2*delta+1) - delta)
	return d + j
}

// NewStore creates a MongoDB store and starts a goroutine to connect with exponential backoff.
func NewStore(ctx context.Context, onReady func()) *Store {
	uri, database := getMongoConfig()
	s := &Store{database: database}
	go s.connectWithRetry(ctx, uri, onReady)
	return s
}

func (s *Store) connectWithRetry(ctx context.Context, uri string, onReady func()) {
	backoff := defaultInitialBackoff

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		opts := options.Client().
			ApplyURI(uri).
			SetServerSelectionTimeout(5 * time.Second).
			SetConnectTimeout(5 * time.Second).
			SetSocketTimeout(10 * time.Second).
			SetWriteConcern(writeconcern.New(writeconcern.W(1), writeconcern.J(true)))

		client, err := mongo.Connect(ctx, opts)
		if err != nil {
			time.Sleep(jitter(backoff))
			if backoff < defaultMaxBackoff {
				backoff *= 2
				if backoff > defaultMaxBackoff {
					backoff = defaultMaxBackoff
				}
			}
			continue
		}

		if err := client.Ping(ctx, nil); err != nil {
			_ = client.Disconnect(ctx)
			time.Sleep(jitter(backoff))
			if backoff < defaultMaxBackoff {
				backoff *= 2
				if backoff > defaultMaxBackoff {
					backoff = defaultMaxBackoff
				}
			}
			continue
		}

		s.mu.Lock()
		s.client = client
		s.coll = client.Database(s.database).Collection("todo_items")
		s.mu.Unlock()
		s.dbReady.Store(true)
		if onReady != nil {
			onReady()
		}
		return
	}
}

func (s *Store) requireDB() bool {
	return s.dbReady.Load()
}

func (s *Store) getColl() *mongo.Collection {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.coll
}

// Create implements store.TodoStore.
func (s *Store) Create(ctx context.Context, description string) (*model.TodoItem, error) {
	if !s.requireDB() {
		return nil, store.ErrNotReady
	}
	coll := s.getColl()
	if coll == nil {
		return nil, store.ErrNotReady
	}
	d := doc{Description: description, Completed: false}
	res, err := coll.InsertOne(ctx, d)
	if err != nil {
		return nil, err
	}
	oid := res.InsertedID.(primitive.ObjectID)
	return &model.TodoItem{
		ID:          oid.Hex(),
		Description: description,
		Completed:   false,
	}, nil
}

// GetByCompleted implements store.TodoStore.
func (s *Store) GetByCompleted(ctx context.Context, completed bool) ([]*model.TodoItem, error) {
	if !s.requireDB() {
		return nil, store.ErrNotReady
	}
	coll := s.getColl()
	if coll == nil {
		return nil, store.ErrNotReady
	}
	cur, err := coll.Find(ctx, bson.M{"completed": completed}, options.Find().SetLimit(500))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*model.TodoItem
	for cur.Next(ctx) {
		var d doc
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		out = append(out, &model.TodoItem{
			ID:          d.ID.Hex(),
			Description: d.Description,
			Completed:   d.Completed,
		})
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetByID implements store.TodoStore.
func (s *Store) GetByID(ctx context.Context, id string) (*model.TodoItem, error) {
	if !s.requireDB() {
		return nil, store.ErrNotReady
	}
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	coll := s.getColl()
	if coll == nil {
		return nil, store.ErrNotReady
	}
	var d doc
	err = coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&d)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, store.ErrNotFound
		}
		return nil, err
	}
	return &model.TodoItem{
		ID:          d.ID.Hex(),
		Description: d.Description,
		Completed:   d.Completed,
	}, nil
}

// Update implements store.TodoStore.
func (s *Store) Update(ctx context.Context, id string, completed bool) error {
	if !s.requireDB() {
		return store.ErrNotReady
	}
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	coll := s.getColl()
	if coll == nil {
		return store.ErrNotReady
	}
	res, err := coll.UpdateOne(ctx, bson.M{"_id": oid}, bson.D{{Key: "$set", Value: bson.D{{Key: "completed", Value: completed}}}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return store.ErrNotFound
	}
	return nil
}

// Delete implements store.TodoStore.
func (s *Store) Delete(ctx context.Context, id string) error {
	if !s.requireDB() {
		return store.ErrNotReady
	}
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	coll := s.getColl()
	if coll == nil {
		return store.ErrNotReady
	}
	res, err := coll.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return store.ErrNotFound
	}
	return nil
}

// Ping implements store.TodoStore.
func (s *Store) Ping(ctx context.Context) error {
	if !s.requireDB() {
		return store.ErrNotReady
	}
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return store.ErrNotReady
	}
	return client.Ping(ctx, nil)
}

// Close implements store.TodoStore.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil {
		return nil
	}
	s.dbReady.Store(false)
	err := s.client.Disconnect(context.Background())
	s.client = nil
	s.coll = nil
	return err
}

var _ store.TodoStore = (*Store)(nil)
