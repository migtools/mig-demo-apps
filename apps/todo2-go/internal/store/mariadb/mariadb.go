package mariadb

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/weshayutin/todo2-go/internal/model"
	"github.com/weshayutin/todo2-go/internal/store"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	defaultMaxBackoff = 30 * time.Second
	defaultInitialBackoff = time.Second
	jitterPercent = 0.1
)

// todoRow is the GORM model for the database table.
type todoRow struct {
	ID          int    `gorm:"primaryKey"`
	Description string
	Completed   bool
}

func (todoRow) TableName() string {
	return "todo_items"
}

// Store implements store.TodoStore for MariaDB/MySQL using GORM.
type Store struct {
	db      *gorm.DB
	dbReady atomic.Bool
	mu      sync.Mutex
}

// getDSNFromEnv returns user, password, database, host, port from env with fallbacks.
func getDSNFromEnv() (user, password, database, host, port string) {
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
		host = "127.0.0.1"
	}
	port = os.Getenv("MYSQL_PORT")
	if port == "" {
		port = "3306"
	}
	return user, password, database, host, port
}

func buildDSN(host, port, user, password, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s&readTimeout=10s&writeTimeout=10s",
		user, password, host, port, database)
}

// jitter returns backoff with ±10% jitter.
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

// NewStore creates a MariaDB store and starts a goroutine to connect with exponential backoff.
// The store is usable immediately; DB-dependent operations return errors until connected.
func NewStore(ctx context.Context, onReady func()) *Store {
	s := &Store{}
	go s.connectWithRetry(ctx, onReady)
	return s
}

func (s *Store) connectWithRetry(ctx context.Context, onReady func()) {
	user, password, database, host, port := getDSNFromEnv()
	remoteDSN := buildDSN(host, port, user, password, database)
	localDSN := buildDSN("127.0.0.1", port, user, password, database)

	backoff := defaultInitialBackoff
	attempt := 0

	for {
		attempt++
		select {
		case <-ctx.Done():
			return
		default:
		}

		var conn *gorm.DB
		var err error
		conn, err = gorm.Open(mysql.Open(remoteDSN), &gorm.Config{})
		if err != nil {
			conn, err = gorm.Open(mysql.Open(localDSN), &gorm.Config{})
		}
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

		sqlDB, err := conn.DB()
		if err != nil {
			time.Sleep(jitter(backoff))
			continue
		}
		if err := sqlDB.Ping(); err != nil {
			sqlDB.Close()
			time.Sleep(jitter(backoff))
			continue
		}

		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)

		s.mu.Lock()
		s.db = conn
		s.mu.Unlock()

		if err := conn.AutoMigrate(&todoRow{}); err != nil {
			// non-fatal
		}
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

func (s *Store) getDB() *gorm.DB {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db
}

// Create implements store.TodoStore.
func (s *Store) Create(ctx context.Context, description string) (*model.TodoItem, error) {
	if !s.requireDB() {
		return nil, store.ErrNotReady
	}
	row := &todoRow{Description: description, Completed: false}
	db := s.getDB()
	if db == nil {
		return nil, store.ErrNotReady
	}
	if err := db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return &model.TodoItem{
		ID:          strconv.Itoa(row.ID),
		Description: row.Description,
		Completed:   row.Completed,
	}, nil
}

// GetByCompleted implements store.TodoStore.
func (s *Store) GetByCompleted(ctx context.Context, completed bool) ([]*model.TodoItem, error) {
	if !s.requireDB() {
		return nil, store.ErrNotReady
	}
	db := s.getDB()
	if db == nil {
		return nil, store.ErrNotReady
	}
	var rows []todoRow
	if err := db.WithContext(ctx).Where("completed = ?", completed).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*model.TodoItem, len(rows))
	for i := range rows {
		out[i] = &model.TodoItem{
			ID:          strconv.Itoa(rows[i].ID),
			Description: rows[i].Description,
			Completed:   rows[i].Completed,
		}
	}
	return out, nil
}

// GetByID implements store.TodoStore.
func (s *Store) GetByID(ctx context.Context, id string) (*model.TodoItem, error) {
	if !s.requireDB() {
		return nil, store.ErrNotReady
	}
	pid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	db := s.getDB()
	if db == nil {
		return nil, store.ErrNotReady
	}
	var row todoRow
	if err := db.WithContext(ctx).First(&row, pid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, store.ErrNotFound
		}
		return nil, err
	}
	return &model.TodoItem{
		ID:          strconv.Itoa(row.ID),
		Description: row.Description,
		Completed:   row.Completed,
	}, nil
}

// Update implements store.TodoStore.
func (s *Store) Update(ctx context.Context, id string, completed bool) error {
	if !s.requireDB() {
		return store.ErrNotReady
	}
	pid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	db := s.getDB()
	if db == nil {
		return store.ErrNotReady
	}
	res := db.WithContext(ctx).Model(&todoRow{}).Where("id = ?", pid).Update("completed", completed)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

// Delete implements store.TodoStore.
func (s *Store) Delete(ctx context.Context, id string) error {
	if !s.requireDB() {
		return store.ErrNotReady
	}
	pid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	db := s.getDB()
	if db == nil {
		return store.ErrNotReady
	}
	res := db.WithContext(ctx).Delete(&todoRow{}, pid)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

// Ping implements store.TodoStore.
func (s *Store) Ping(ctx context.Context) error {
	if !s.requireDB() {
		return store.ErrNotReady
	}
	db := s.getDB()
	if db == nil {
		return store.ErrNotReady
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close implements store.TodoStore.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	s.dbReady.Store(false)
	s.db = nil
	return sqlDB.Close()
}

// Ensure Store implements store.TodoStore at compile time.
var _ store.TodoStore = (*Store)(nil)
