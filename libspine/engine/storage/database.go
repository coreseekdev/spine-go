package storage

import (
	"sync"
	"time"
	
	"spine-go/libspine/engine/storage/stream"
)

// ValueType represents the type of a Redis value
type ValueType int

const (
	TypeString ValueType = iota
	TypeList
	TypeSet
	TypeZSet
	TypeHash
	TypeStream
)

// Value represents a Redis value with its type and expiration
type Value struct {
	Type      ValueType
	Data      interface{}
	ExpiresAt *time.Time // nil means no expiration
}

// IsExpired checks if the value has expired
func (v *Value) IsExpired() bool {
	if v.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*v.ExpiresAt)
}

// Database represents a Redis database instance
type Database struct {
	mu     sync.RWMutex
	dbNum  int
	data   map[string]*Value
	expiry map[string]*time.Time // separate expiry tracking for efficiency
	
	// Category-specific storage interfaces
	StringStorage StringStorage
	HashStorage   HashStorage
	ListStorage   ListStorage
	SetStorage    SetStorage
	ZSetStorage   ZSetStorage
	BitmapStorage BitmapStorage
	StreamStorage StreamStorage
	CommonStorage CommonStorage
}

// NewDatabase creates a new database instance
func NewDatabase(dbNum int) *Database {
	db := &Database{
		dbNum:  dbNum,
		data:   make(map[string]*Value),
		expiry: make(map[string]*time.Time),
	}
	
	// Initialize category-specific storages
	db.StringStorage = NewStringStorage(db)
	db.HashStorage = NewHashStorage(db)
	db.ListStorage = NewListStorage(db)
	db.SetStorage = NewSetStorage(db)
	db.ZSetStorage = NewZSetStorage(db)
	db.BitmapStorage = NewBitmapStorage(db)
	db.StreamStorage = stream.NewStreamStorage(db)
	db.CommonStorage = NewCommonStorage(db)
	
	return db
}

// GetDBNum returns the database number
func (db *Database) GetDBNum() int {
	return db.dbNum
}

// Set sets a string value
func (db *Database) Set(key, value string, expiration *time.Time) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.data[key] = &Value{
		Type:      TypeString,
		Data:      value,
		ExpiresAt: expiration,
	}

	if expiration != nil {
		db.expiry[key] = expiration
	} else {
		delete(db.expiry, key)
	}

	return nil
}

// Get gets a string value
func (db *Database) Get(key string) (string, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	value, exists := db.data[key]
	if !exists {
		return "", false
	}

	// Check expiration
	if value.IsExpired() {
		// Clean up expired key
		db.mu.RUnlock()
		db.mu.Lock()
		delete(db.data, key)
		delete(db.expiry, key)
		db.mu.Unlock()
		db.mu.RLock()
		return "", false
	}

	if value.Type != TypeString {
		return "", false
	}

	str, ok := value.Data.(string)
	return str, ok
}

// Exists checks if a key exists
func (db *Database) Exists(keys ...string) int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	count := 0
	for _, key := range keys {
		if value, exists := db.data[key]; exists && !value.IsExpired() {
			count++
		}
	}
	return count
}

// Del deletes keys
func (db *Database) Del(keys ...string) int {
	db.mu.Lock()
	defer db.mu.Unlock()

	count := 0
	for _, key := range keys {
		if _, exists := db.data[key]; exists {
			delete(db.data, key)
			delete(db.expiry, key)
			count++
		}
	}
	return count
}

// Type returns the type of a key
func (db *Database) Type(key string) (ValueType, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	value, exists := db.data[key]
	if !exists || value.IsExpired() {
		return TypeString, false
	}

	return value.Type, true
}

// Expire sets expiration for a key
func (db *Database) Expire(key string, expiration time.Time) bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	value, exists := db.data[key]
	if !exists || value.IsExpired() {
		return false
	}

	value.ExpiresAt = &expiration
	db.expiry[key] = &expiration
	return true
}

// TTL returns the time to live for a key
func (db *Database) TTL(key string) (time.Duration, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	value, exists := db.data[key]
	if !exists {
		return 0, false
	}

	if value.IsExpired() {
		return 0, false
	}

	if value.ExpiresAt == nil {
		return -1, true // No expiration
	}

	ttl := time.Until(*value.ExpiresAt)
	if ttl <= 0 {
		return 0, false // Expired
	}

	return ttl, true
}

// Keys returns all keys matching a pattern (simplified implementation)
func (db *Database) Keys(pattern string) []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var keys []string
	for key, value := range db.data {
		if !value.IsExpired() {
			// Simple pattern matching - in production, use proper glob matching
			if pattern == "*" || key == pattern {
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// FlushDB removes all keys from the database
func (db *Database) FlushDB() {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.data = make(map[string]*Value)
	db.expiry = make(map[string]*time.Time)
}

// DBSize returns the number of keys in the database
func (db *Database) DBSize() int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	count := 0
	for _, value := range db.data {
		if !value.IsExpired() {
			count++
		}
	}
	return count
}

// CleanupExpired removes all expired keys
func (db *Database) CleanupExpired() int {
	db.mu.Lock()
	defer db.mu.Unlock()

	count := 0
	for key, value := range db.data {
		if value.IsExpired() {
			delete(db.data, key)
			delete(db.expiry, key)
			count++
		}
	}
	return count
}

// GetValue returns the raw value for a key (for internal use)
func (db *Database) GetValue(key string) (*Value, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	value, exists := db.data[key]
	if !exists || value.IsExpired() {
		return nil, false
	}

	return value, true
}

// SetValue sets a raw value (for internal use)
func (db *Database) SetValue(key string, value *Value) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.data[key] = value
	if value.ExpiresAt != nil {
		db.expiry[key] = value.ExpiresAt
	} else {
		delete(db.expiry, key)
	}
}