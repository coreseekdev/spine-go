package storage

import (
	"testing"
	"time"
)

func TestNewCommonStorage(t *testing.T) {
	db := NewDatabase(0)
	commonStorage := NewCommonStorage(db)
	
	if commonStorage == nil {
		t.Error("NewCommonStorage should not return nil")
	}
	
	// Test that it implements CommonStorage interface
	_, ok := commonStorage.(CommonStorage)
	if !ok {
		t.Error("NewCommonStorage should return an object that implements CommonStorage interface")
	}
}

func TestExists(t *testing.T) {
	db := NewDatabase(0)
	commonStorage := NewCommonStorage(db)
	
	// Test non-existent key
	if commonStorage.Exists("nonexistent") {
		t.Error("Exists should return false for non-existent key")
	}
	
	// Add a key
	db.Set("testkey", "testvalue", nil)
	
	// Test existing key
	if !commonStorage.Exists("testkey") {
		t.Error("Exists should return true for existing key")
	}
	
	// Test expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expiredkey", "expiredvalue", &expiredTime)
	
	if commonStorage.Exists("expiredkey") {
		t.Error("Exists should return false for expired key")
	}
}

func TestDel(t *testing.T) {
	db := NewDatabase(0)
	commonStorage := NewCommonStorage(db)
	
	// Add some test keys
	db.Set("key1", "value1", nil)
	db.Set("key2", "value2", nil)
	db.Set("key3", "value3", nil)
	
	tests := []struct {
		name          string
		keys          []string
		expectedCount int64
	}{
		{"Delete single existing key", []string{"key1"}, 1},
		{"Delete multiple existing keys", []string{"key2", "key3"}, 2},
		{"Delete non-existent key", []string{"nonexistent"}, 0},
		{"Delete mix of existing and non-existent", []string{"key1", "nonexistent"}, 0}, // key1 already deleted
		{"Delete empty list", []string{}, 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleted := commonStorage.Del(tt.keys)
			if deleted != tt.expectedCount {
				t.Errorf("Expected %d deleted keys, got %d", tt.expectedCount, deleted)
			}
		})
	}
}

func TestType(t *testing.T) {
	db := NewDatabase(0)
	commonStorage := NewCommonStorage(db)
	
	// Add keys of different types
	db.Set("stringkey", "stringvalue", nil)
	
	// Create a hash value manually
	db.data["hashkey"] = &Value{
		Type: TypeHash,
		Data: make(map[string]string),
	}
	
	// Create a list value manually
	db.data["listkey"] = &Value{
		Type: TypeList,
		Data: []string{},
	}
	
	tests := []struct {
		name         string
		key          string
		expectedType ValueType
	}{
		{"String type", "stringkey", TypeString},
		{"Hash type", "hashkey", TypeHash},
		{"List type", "listkey", TypeList},
		{"Non-existent key", "nonexistent", ValueType(-1)},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyType := commonStorage.Type(tt.key)
			if keyType != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, keyType)
			}
		})
	}
	
	// Test expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expiredkey", "expiredvalue", &expiredTime)
	
	keyType := commonStorage.Type("expiredkey")
	if keyType != ValueType(-1) {
		t.Errorf("Expected type -1 for expired key, got %v", keyType)
	}
}

func TestTTL(t *testing.T) {
	db := NewDatabase(0)
	commonStorage := NewCommonStorage(db)
	
	// Test non-existent key
	ttl := commonStorage.TTL("nonexistent")
	if ttl != -2*time.Second {
		t.Errorf("Expected TTL -2s for non-existent key, got %v", ttl)
	}
	
	// Test key without expiration
	db.Set("persistent", "value", nil)
	ttl = commonStorage.TTL("persistent")
	if ttl != -1*time.Second {
		t.Errorf("Expected TTL -1s for persistent key, got %v", ttl)
	}
	
	// Test key with expiration
	futureTime := time.Now().Add(10 * time.Second)
	db.Set("expiring", "value", &futureTime)
	ttl = commonStorage.TTL("expiring")
	
	// TTL should be positive and roughly 10 seconds (allow some variance)
	if ttl <= 9*time.Second || ttl > 10*time.Second {
		t.Errorf("Expected TTL around 10s, got %v", ttl)
	}
	
	// Test expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired", "value", &expiredTime)
	ttl = commonStorage.TTL("expired")
	if ttl != -2*time.Second {
		t.Errorf("Expected TTL -2s for expired key, got %v", ttl)
	}
}

func TestExpire(t *testing.T) {
	db := NewDatabase(0)
	commonStorage := NewCommonStorage(db)
	
	// Test setting expiration on existing key
	db.Set("testkey", "testvalue", nil)
	
	success := commonStorage.Expire("testkey", 5*time.Second)
	if !success {
		t.Error("Expire should return true for existing key")
	}
	
	// Verify expiration was set
	ttl := commonStorage.TTL("testkey")
	if ttl <= 4*time.Second || ttl > 5*time.Second {
		t.Errorf("Expected TTL around 5s after Expire, got %v", ttl)
	}
	
	// Test setting expiration on non-existent key
	success = commonStorage.Expire("nonexistent", 5*time.Second)
	if success {
		t.Error("Expire should return false for non-existent key")
	}
	
	// Test setting expiration on expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired", "value", &expiredTime)
	
	success = commonStorage.Expire("expired", 5*time.Second)
	if success {
		t.Error("Expire should return false for expired key")
	}
}

func TestExpireAt(t *testing.T) {
	db := NewDatabase(0)
	commonStorage := NewCommonStorage(db)
	
	// Test setting expiration timestamp on existing key
	db.Set("testkey", "testvalue", nil)
	
	futureTime := time.Now().Add(10 * time.Second)
	success := commonStorage.ExpireAt("testkey", futureTime)
	if !success {
		t.Error("ExpireAt should return true for existing key")
	}
	
	// Verify expiration was set
	ttl := commonStorage.TTL("testkey")
	if ttl <= 9*time.Second || ttl > 10*time.Second {
		t.Errorf("Expected TTL around 10s after ExpireAt, got %v", ttl)
	}
	
	// Test setting expiration on non-existent key
	success = commonStorage.ExpireAt("nonexistent", futureTime)
	if success {
		t.Error("ExpireAt should return false for non-existent key")
	}
	
	// Test setting expiration on expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired", "value", &expiredTime)
	
	success = commonStorage.ExpireAt("expired", futureTime)
	if success {
		t.Error("ExpireAt should return false for expired key")
	}
}

func TestPersist(t *testing.T) {
	db := NewDatabase(0)
	commonStorage := NewCommonStorage(db)
	
	// Test removing expiration from key with expiration
	futureTime := time.Now().Add(10 * time.Second)
	db.Set("expiring", "value", &futureTime)
	
	success := commonStorage.Persist("expiring")
	if !success {
		t.Error("Persist should return true for key with expiration")
	}
	
	// Verify expiration was removed
	ttl := commonStorage.TTL("expiring")
	if ttl != -1*time.Second {
		t.Errorf("Expected TTL -1s after Persist, got %v", ttl)
	}
	
	// Test removing expiration from key without expiration
	db.Set("persistent", "value", nil)
	
	success = commonStorage.Persist("persistent")
	if success {
		t.Error("Persist should return false for key without expiration")
	}
	
	// Test removing expiration from non-existent key
	success = commonStorage.Persist("nonexistent")
	if success {
		t.Error("Persist should return false for non-existent key")
	}
	
	// Test removing expiration from expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired", "value", &expiredTime)
	
	success = commonStorage.Persist("expired")
	if success {
		t.Error("Persist should return false for expired key")
	}
}

func TestSwapDB(t *testing.T) {
	db := NewDatabase(0)
	commonStorage := NewCommonStorage(db)
	
	// Test SwapDB (currently a no-op implementation)
	err := commonStorage.SwapDB(0, 1)
	if err != nil {
		t.Errorf("SwapDB should not return error, got %v", err)
	}
}

func TestExistsWithExpiredKeyCleanup(t *testing.T) {
	db := NewDatabase(0)
	commonStorage := NewCommonStorage(db)
	
	// Add an expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired", "value", &expiredTime)
	
	// Verify key exists in database before cleanup
	if _, exists := db.data["expired"]; !exists {
		t.Error("Expired key should exist in database before cleanup")
	}
	
	// Call Exists which should trigger cleanup
	exists := commonStorage.Exists("expired")
	if exists {
		t.Error("Exists should return false for expired key")
	}
	
	// Verify key was cleaned up from database
	if _, exists := db.data["expired"]; exists {
		t.Error("Expired key should be cleaned up from database")
	}
	
	// Verify key was cleaned up from expiry map
	if _, exists := db.expiry["expired"]; exists {
		t.Error("Expired key should be cleaned up from expiry map")
	}
}
