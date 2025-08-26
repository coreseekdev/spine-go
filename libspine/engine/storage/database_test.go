package storage

import (
	"testing"
	"time"
)

func TestNewDatabase(t *testing.T) {
	db := NewDatabase(5)
	
	if db == nil {
		t.Error("NewDatabase should not return nil")
	}
	
	if db.GetDBNum() != 5 {
		t.Errorf("Expected database number 5, got %d", db.GetDBNum())
	}
	
	// Test that all storage interfaces are initialized
	if db.StringStorage == nil {
		t.Error("StringStorage should be initialized")
	}
	if db.HashStorage == nil {
		t.Error("HashStorage should be initialized")
	}
	if db.ListStorage == nil {
		t.Error("ListStorage should be initialized")
	}
	if db.SetStorage == nil {
		t.Error("SetStorage should be initialized")
	}
	if db.ZSetStorage == nil {
		t.Error("ZSetStorage should be initialized")
	}
	if db.BitmapStorage == nil {
		t.Error("BitmapStorage should be initialized")
	}
	if db.CommonStorage == nil {
		t.Error("CommonStorage should be initialized")
	}
}

func TestGetDBNum(t *testing.T) {
	tests := []int{0, 1, 5, 15}
	
	for _, dbNum := range tests {
		db := NewDatabase(dbNum)
		if db.GetDBNum() != dbNum {
			t.Errorf("Expected database number %d, got %d", dbNum, db.GetDBNum())
		}
	}
}

func TestSetAndGet(t *testing.T) {
	db := NewDatabase(0)
	
	// Test basic set and get
	err := db.Set("key1", "value1", nil)
	if err != nil {
		t.Errorf("Set should not return error, got %v", err)
	}
	
	value, exists := db.Get("key1")
	if !exists {
		t.Error("Get should return true for existing key")
	}
	if value != "value1" {
		t.Errorf("Expected value 'value1', got '%s'", value)
	}
	
	// Test get non-existent key
	value, exists = db.Get("nonexistent")
	if exists {
		t.Error("Get should return false for non-existent key")
	}
	if value != "" {
		t.Errorf("Expected empty string for non-existent key, got '%s'", value)
	}
	
	// Test set with expiration
	futureTime := time.Now().Add(1 * time.Hour)
	err = db.Set("expiring", "expiring_value", &futureTime)
	if err != nil {
		t.Errorf("Set with expiration should not return error, got %v", err)
	}
	
	value, exists = db.Get("expiring")
	if !exists {
		t.Error("Get should return true for key with future expiration")
	}
	if value != "expiring_value" {
		t.Errorf("Expected value 'expiring_value', got '%s'", value)
	}
	
	// Test get expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	err = db.Set("expired", "expired_value", &expiredTime)
	if err != nil {
		t.Errorf("Set with expired time should not return error, got %v", err)
	}
	
	value, exists = db.Get("expired")
	if exists {
		t.Error("Get should return false for expired key")
	}
	if value != "" {
		t.Errorf("Expected empty string for expired key, got '%s'", value)
	}
}

func TestDatabaseExists(t *testing.T) {
	db := NewDatabase(0)
	
	// Test with no keys
	count := db.Exists()
	if count != 0 {
		t.Errorf("Expected 0 existing keys, got %d", count)
	}
	
	// Add some keys
	db.Set("key1", "value1", nil)
	db.Set("key2", "value2", nil)
	
	// Test single key
	count = db.Exists("key1")
	if count != 1 {
		t.Errorf("Expected 1 existing key, got %d", count)
	}
	
	// Test multiple keys
	count = db.Exists("key1", "key2")
	if count != 2 {
		t.Errorf("Expected 2 existing keys, got %d", count)
	}
	
	// Test mix of existing and non-existent
	count = db.Exists("key1", "nonexistent", "key2")
	if count != 2 {
		t.Errorf("Expected 2 existing keys, got %d", count)
	}
	
	// Test expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired", "expired_value", &expiredTime)
	
	count = db.Exists("key1", "expired")
	if count != 1 {
		t.Errorf("Expected 1 existing key (expired should not count), got %d", count)
	}
}

func TestDatabaseDel(t *testing.T) {
	db := NewDatabase(0)
	
	// Add some keys
	db.Set("key1", "value1", nil)
	db.Set("key2", "value2", nil)
	db.Set("key3", "value3", nil)
	
	// Test delete single key
	count := db.Del("key1")
	if count != 1 {
		t.Errorf("Expected 1 deleted key, got %d", count)
	}
	
	// Verify key was deleted
	_, exists := db.Get("key1")
	if exists {
		t.Error("Deleted key should not exist")
	}
	
	// Test delete multiple keys
	count = db.Del("key2", "key3")
	if count != 2 {
		t.Errorf("Expected 2 deleted keys, got %d", count)
	}
	
	// Test delete non-existent key
	count = db.Del("nonexistent")
	if count != 0 {
		t.Errorf("Expected 0 deleted keys for non-existent key, got %d", count)
	}
	
	// Test delete already deleted key
	count = db.Del("key1")
	if count != 0 {
		t.Errorf("Expected 0 deleted keys for already deleted key, got %d", count)
	}
}

func TestDatabaseType(t *testing.T) {
	db := NewDatabase(0)
	
	// Test string type
	db.Set("stringkey", "stringvalue", nil)
	valueType, exists := db.Type("stringkey")
	if !exists {
		t.Error("Type should return true for existing key")
	}
	if valueType != TypeString {
		t.Errorf("Expected TypeString, got %v", valueType)
	}
	
	// Test non-existent key
	valueType, exists = db.Type("nonexistent")
	if exists {
		t.Error("Type should return false for non-existent key")
	}
	
	// Test expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired", "expired_value", &expiredTime)
	
	valueType, exists = db.Type("expired")
	if exists {
		t.Error("Type should return false for expired key")
	}
}

func TestDatabaseExpire(t *testing.T) {
	db := NewDatabase(0)
	
	// Test expire existing key
	db.Set("key1", "value1", nil)
	
	futureTime := time.Now().Add(1 * time.Hour)
	success := db.Expire("key1", futureTime)
	if !success {
		t.Error("Expire should return true for existing key")
	}
	
	// Test expire non-existent key
	success = db.Expire("nonexistent", futureTime)
	if success {
		t.Error("Expire should return false for non-existent key")
	}
	
	// Test expire expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired", "expired_value", &expiredTime)
	
	success = db.Expire("expired", futureTime)
	if success {
		t.Error("Expire should return false for expired key")
	}
}

func TestDatabaseTTL(t *testing.T) {
	db := NewDatabase(0)
	
	// Test TTL for non-existent key
	ttl, exists := db.TTL("nonexistent")
	if exists {
		t.Error("TTL should return false for non-existent key")
	}
	
	// Test TTL for key without expiration
	db.Set("persistent", "value", nil)
	ttl, exists = db.TTL("persistent")
	if !exists {
		t.Error("TTL should return true for existing key")
	}
	if ttl != -1 {
		t.Errorf("Expected TTL -1 for persistent key, got %v", ttl)
	}
	
	// Test TTL for key with expiration
	futureTime := time.Now().Add(10 * time.Second)
	db.Set("expiring", "value", &futureTime)
	ttl, exists = db.TTL("expiring")
	if !exists {
		t.Error("TTL should return true for key with expiration")
	}
	if ttl <= 9*time.Second || ttl > 10*time.Second {
		t.Errorf("Expected TTL around 10s, got %v", ttl)
	}
	
	// Test TTL for expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired", "value", &expiredTime)
	ttl, exists = db.TTL("expired")
	if exists {
		t.Error("TTL should return false for expired key")
	}
}

func TestKeys(t *testing.T) {
	db := NewDatabase(0)
	
	// Test empty database
	keys := db.Keys("*")
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys in empty database, got %d", len(keys))
	}
	
	// Add some keys
	db.Set("key1", "value1", nil)
	db.Set("key2", "value2", nil)
	db.Set("test", "testvalue", nil)
	
	// Test wildcard pattern
	keys = db.Keys("*")
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys with wildcard, got %d", len(keys))
	}
	
	// Test specific key pattern
	keys = db.Keys("key1")
	if len(keys) != 1 {
		t.Errorf("Expected 1 key with specific pattern, got %d", len(keys))
	}
	if keys[0] != "key1" {
		t.Errorf("Expected key 'key1', got '%s'", keys[0])
	}
	
	// Test pattern that doesn't match
	keys = db.Keys("nonexistent")
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys with non-matching pattern, got %d", len(keys))
	}
	
	// Add expired key and test it's not included
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired", "expired_value", &expiredTime)
	
	keys = db.Keys("*")
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys (expired should not be included), got %d", len(keys))
	}
}

func TestFlushDB(t *testing.T) {
	db := NewDatabase(0)
	
	// Add some keys
	db.Set("key1", "value1", nil)
	db.Set("key2", "value2", nil)
	futureTime := time.Now().Add(1 * time.Hour)
	db.Set("key3", "value3", &futureTime)
	
	// Verify keys exist
	if db.DBSize() != 3 {
		t.Errorf("Expected 3 keys before flush, got %d", db.DBSize())
	}
	
	// Flush database
	db.FlushDB()
	
	// Verify database is empty
	if db.DBSize() != 0 {
		t.Errorf("Expected 0 keys after flush, got %d", db.DBSize())
	}
	
	// Verify keys don't exist
	_, exists := db.Get("key1")
	if exists {
		t.Error("Key should not exist after flush")
	}
}

func TestDBSize(t *testing.T) {
	db := NewDatabase(0)
	
	// Test empty database
	size := db.DBSize()
	if size != 0 {
		t.Errorf("Expected size 0 for empty database, got %d", size)
	}
	
	// Add keys
	db.Set("key1", "value1", nil)
	db.Set("key2", "value2", nil)
	
	size = db.DBSize()
	if size != 2 {
		t.Errorf("Expected size 2, got %d", size)
	}
	
	// Add expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired", "expired_value", &expiredTime)
	
	size = db.DBSize()
	if size != 2 {
		t.Errorf("Expected size 2 (expired key should not count), got %d", size)
	}
	
	// Delete a key
	db.Del("key1")
	
	size = db.DBSize()
	if size != 1 {
		t.Errorf("Expected size 1 after deletion, got %d", size)
	}
}

func TestCleanupExpired(t *testing.T) {
	db := NewDatabase(0)
	
	// Add some keys
	db.Set("key1", "value1", nil)
	db.Set("key2", "value2", nil)
	
	// Add expired keys
	expiredTime := time.Now().Add(-1 * time.Hour)
	db.Set("expired1", "expired_value1", &expiredTime)
	db.Set("expired2", "expired_value2", &expiredTime)
	
	// Test cleanup
	count := db.CleanupExpired()
	if count != 2 {
		t.Errorf("Expected 2 expired keys cleaned up, got %d", count)
	}
	
	// Verify expired keys are gone
	_, exists := db.Get("expired1")
	if exists {
		t.Error("Expired key should be cleaned up")
	}
	
	// Verify non-expired keys remain
	_, exists = db.Get("key1")
	if !exists {
		t.Error("Non-expired key should remain")
	}
	
	// Test cleanup when no expired keys
	count = db.CleanupExpired()
	if count != 0 {
		t.Errorf("Expected 0 expired keys cleaned up, got %d", count)
	}
}

func TestGetValueAndSetValue(t *testing.T) {
	db := NewDatabase(0)
	
	// Test GetValue for non-existent key
	value, exists := db.GetValue("nonexistent")
	if exists {
		t.Error("GetValue should return false for non-existent key")
	}
	if value != nil {
		t.Error("GetValue should return nil for non-existent key")
	}
	
	// Test SetValue and GetValue
	testValue := &Value{
		Type: TypeString,
		Data: "test_data",
	}
	
	db.SetValue("testkey", testValue)
	
	retrievedValue, exists := db.GetValue("testkey")
	if !exists {
		t.Error("GetValue should return true for existing key")
	}
	if retrievedValue == nil {
		t.Error("GetValue should not return nil for existing key")
	}
	if retrievedValue.Type != TypeString {
		t.Errorf("Expected TypeString, got %v", retrievedValue.Type)
	}
	if retrievedValue.Data != "test_data" {
		t.Errorf("Expected 'test_data', got %v", retrievedValue.Data)
	}
	
	// Test SetValue with expiration
	futureTime := time.Now().Add(1 * time.Hour)
	testValueWithExpiry := &Value{
		Type:      TypeString,
		Data:      "expiring_data",
		ExpiresAt: &futureTime,
	}
	
	db.SetValue("expiring_key", testValueWithExpiry)
	
	retrievedValue, exists = db.GetValue("expiring_key")
	if !exists {
		t.Error("GetValue should return true for key with future expiration")
	}
	if retrievedValue.ExpiresAt == nil {
		t.Error("Retrieved value should have expiration time")
	}
	
	// Test GetValue for expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredValue := &Value{
		Type:      TypeString,
		Data:      "expired_data",
		ExpiresAt: &expiredTime,
	}
	
	db.SetValue("expired_key", expiredValue)
	
	retrievedValue, exists = db.GetValue("expired_key")
	if exists {
		t.Error("GetValue should return false for expired key")
	}
}

func TestValueIsExpired(t *testing.T) {
	// Test value without expiration
	value := &Value{
		Type: TypeString,
		Data: "test",
	}
	
	if value.IsExpired() {
		t.Error("Value without expiration should not be expired")
	}
	
	// Test value with future expiration
	futureTime := time.Now().Add(1 * time.Hour)
	value.ExpiresAt = &futureTime
	
	if value.IsExpired() {
		t.Error("Value with future expiration should not be expired")
	}
	
	// Test value with past expiration
	pastTime := time.Now().Add(-1 * time.Hour)
	value.ExpiresAt = &pastTime
	
	if !value.IsExpired() {
		t.Error("Value with past expiration should be expired")
	}
}

func TestGetWithWrongType(t *testing.T) {
	db := NewDatabase(0)
	
	// Set a non-string value manually
	db.data["hashkey"] = &Value{
		Type: TypeHash,
		Data: make(map[string]string),
	}
	
	// Try to get as string
	value, exists := db.Get("hashkey")
	if exists {
		t.Error("Get should return false for non-string type")
	}
	if value != "" {
		t.Errorf("Expected empty string for wrong type, got '%s'", value)
	}
}
