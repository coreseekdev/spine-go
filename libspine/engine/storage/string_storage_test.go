package storage

import (
	"testing"
	"time"
)

func TestStringStorage_Set_Get(t *testing.T) {
	db := NewDatabase(0)
	stringStorage := db.StringStorage

	// Test basic set and get
	err := stringStorage.Set("key1", "value1", nil)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	value, exists := stringStorage.Get("key1")
	if !exists {
		t.Error("Key should exist")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}
}

func TestStringStorage_Set_WithExpiration(t *testing.T) {
	db := NewDatabase(0)
	stringStorage := db.StringStorage

	// Test set with expiration
	expiration := time.Now().Add(100 * time.Millisecond)
	err := stringStorage.Set("key2", "value2", &expiration)
	if err != nil {
		t.Errorf("Set with expiration failed: %v", err)
	}

	// Should exist immediately
	value, exists := stringStorage.Get("key2")
	if !exists {
		t.Error("Key should exist before expiration")
	}
	if value != "value2" {
		t.Errorf("Expected 'value2', got '%s'", value)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not exist after expiration
	_, exists = stringStorage.Get("key2")
	if exists {
		t.Error("Key should not exist after expiration")
	}
}

func TestStringStorage_MSet_MGet(t *testing.T) {
	db := NewDatabase(0)
	stringStorage := db.StringStorage

	// Test MSet
	pairs := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := stringStorage.MSet(pairs)
	if err != nil {
		t.Errorf("MSet failed: %v", err)
	}

	// Test MGet
	keys := []string{"key1", "key2", "key3", "nonexistent"}
	result := stringStorage.MGet(keys)

	if len(result) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result))
	}
	if result["key1"] != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result["key1"])
	}
	if result["key2"] != "value2" {
		t.Errorf("Expected 'value2', got '%s'", result["key2"])
	}
	if result["key3"] != "value3" {
		t.Errorf("Expected 'value3', got '%s'", result["key3"])
	}
	if _, exists := result["nonexistent"]; exists {
		t.Error("Nonexistent key should not be in result")
	}
}

func TestStringStorage_Exists(t *testing.T) {
	db := NewDatabase(0)
	stringStorage := db.StringStorage

	// Key should not exist initially
	if stringStorage.Exists("key1") {
		t.Error("Key should not exist initially")
	}

	// Set key
	stringStorage.Set("key1", "value1", nil)

	// Key should exist now
	if !stringStorage.Exists("key1") {
		t.Error("Key should exist after set")
	}
}

func TestStringStorage_Del(t *testing.T) {
	db := NewDatabase(0)
	stringStorage := db.StringStorage

	// Set key
	stringStorage.Set("key1", "value1", nil)

	// Delete key
	deleted := stringStorage.Del("key1")
	if !deleted {
		t.Error("Delete should return true for existing key")
	}

	// Key should not exist now
	if stringStorage.Exists("key1") {
		t.Error("Key should not exist after delete")
	}

	// Delete non-existent key
	deleted = stringStorage.Del("nonexistent")
	if deleted {
		t.Error("Delete should return false for non-existent key")
	}
}

func TestStringStorage_Incr_Decr(t *testing.T) {
	db := NewDatabase(0)
	stringStorage := db.StringStorage

	// Test Incr on non-existent key (should start at 0)
	result, err := stringStorage.Incr("counter")
	if err != nil {
		t.Errorf("Incr failed: %v", err)
	}
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}

	// Test Incr on existing key
	result, err = stringStorage.Incr("counter")
	if err != nil {
		t.Errorf("Incr failed: %v", err)
	}
	if result != 2 {
		t.Errorf("Expected 2, got %d", result)
	}

	// Test Decr
	result, err = stringStorage.Decr("counter")
	if err != nil {
		t.Errorf("Decr failed: %v", err)
	}
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
}

func TestStringStorage_IncrBy_DecrBy(t *testing.T) {
	db := NewDatabase(0)
	stringStorage := db.StringStorage

	// Test IncrBy
	result, err := stringStorage.IncrBy("counter", 5)
	if err != nil {
		t.Errorf("IncrBy failed: %v", err)
	}
	if result != 5 {
		t.Errorf("Expected 5, got %d", result)
	}

	// Test DecrBy
	result, err = stringStorage.DecrBy("counter", 3)
	if err != nil {
		t.Errorf("DecrBy failed: %v", err)
	}
	if result != 2 {
		t.Errorf("Expected 2, got %d", result)
	}
}

func TestStringStorage_Append(t *testing.T) {
	db := NewDatabase(0)
	stringStorage := db.StringStorage

	// Test append to non-existent key
	length, err := stringStorage.Append("key1", "hello")
	if err != nil {
		t.Errorf("Append failed: %v", err)
	}
	if length != 5 {
		t.Errorf("Expected length 5, got %d", length)
	}

	// Test append to existing key
	length, err = stringStorage.Append("key1", " world")
	if err != nil {
		t.Errorf("Append failed: %v", err)
	}
	if length != 11 {
		t.Errorf("Expected length 11, got %d", length)
	}

	// Verify final value
	value, exists := stringStorage.Get("key1")
	if !exists {
		t.Error("Key should exist")
	}
	if value != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", value)
	}
}

func TestStringStorage_StrLen(t *testing.T) {
	db := NewDatabase(0)
	stringStorage := db.StringStorage

	// Test StrLen on non-existent key
	length := stringStorage.StrLen("nonexistent")
	if length != 0 {
		t.Errorf("Expected length 0, got %d", length)
	}

	// Set key and test StrLen
	stringStorage.Set("key1", "hello", nil)
	length = stringStorage.StrLen("key1")
	if length != 5 {
		t.Errorf("Expected length 5, got %d", length)
	}
}

func TestStringStorage_IncrError(t *testing.T) {
	db := NewDatabase(0)
	stringStorage := db.StringStorage

	// Set non-numeric value
	stringStorage.Set("key1", "not_a_number", nil)

	// Try to increment
	_, err := stringStorage.Incr("key1")
	if err == nil {
		t.Error("Expected error when incrementing non-numeric value")
	}
	if err != ErrNotInteger {
		t.Errorf("Expected ErrNotInteger, got %v", err)
	}
}
