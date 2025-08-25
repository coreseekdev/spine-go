package storage

import (
	"testing"
)

func TestHashStorage_HSet_HGet(t *testing.T) {
	db := NewDatabase(0)
	hashStorage := db.HashStorage

	// Test HSet on new hash
	isNewField, err := hashStorage.HSet("hash1", "field1", "value1")
	if err != nil {
		t.Errorf("HSet failed: %v", err)
	}
	if !isNewField {
		t.Error("Expected new field to return true")
	}

	// Test HGet
	value, exists := hashStorage.HGet("hash1", "field1")
	if !exists {
		t.Error("Field should exist")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	// Test HSet on existing field
	isNewField, err = hashStorage.HSet("hash1", "field1", "newvalue1")
	if err != nil {
		t.Errorf("HSet failed: %v", err)
	}
	if isNewField {
		t.Error("Expected existing field to return false")
	}

	// Verify updated value
	value, exists = hashStorage.HGet("hash1", "field1")
	if !exists {
		t.Error("Field should exist")
	}
	if value != "newvalue1" {
		t.Errorf("Expected 'newvalue1', got '%s'", value)
	}
}

func TestHashStorage_HMSet_HMGet(t *testing.T) {
	db := NewDatabase(0)
	hashStorage := db.HashStorage

	// Test HMSet
	fields := map[string]string{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
	}
	err := hashStorage.HMSet("hash1", fields)
	if err != nil {
		t.Errorf("HMSet failed: %v", err)
	}

	// Test HMGet
	fieldNames := []string{"field1", "field2", "field3", "nonexistent"}
	result := hashStorage.HMGet("hash1", fieldNames)

	if len(result) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result))
	}
	if result["field1"] != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result["field1"])
	}
	if result["field2"] != "value2" {
		t.Errorf("Expected 'value2', got '%s'", result["field2"])
	}
	if result["field3"] != "value3" {
		t.Errorf("Expected 'value3', got '%s'", result["field3"])
	}
	if _, exists := result["nonexistent"]; exists {
		t.Error("Nonexistent field should not be in result")
	}
}

func TestHashStorage_HGetAll(t *testing.T) {
	db := NewDatabase(0)
	hashStorage := db.HashStorage

	// Test HGetAll on non-existent hash
	result, err := hashStorage.HGetAll("nonexistent")
	if err != nil {
		t.Errorf("HGetAll failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}

	// Set some fields
	fields := map[string]string{
		"field1": "value1",
		"field2": "value2",
	}
	hashStorage.HMSet("hash1", fields)

	// Test HGetAll
	result, err = hashStorage.HGetAll("hash1")
	if err != nil {
		t.Errorf("HGetAll failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}
	if result["field1"] != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result["field1"])
	}
	if result["field2"] != "value2" {
		t.Errorf("Expected 'value2', got '%s'", result["field2"])
	}
}

func TestHashStorage_HExists(t *testing.T) {
	db := NewDatabase(0)
	hashStorage := db.HashStorage

	// Field should not exist initially
	if hashStorage.HExists("hash1", "field1") {
		t.Error("Field should not exist initially")
	}

	// Set field
	hashStorage.HSet("hash1", "field1", "value1")

	// Field should exist now
	if !hashStorage.HExists("hash1", "field1") {
		t.Error("Field should exist after set")
	}

	// Non-existent field should not exist
	if hashStorage.HExists("hash1", "nonexistent") {
		t.Error("Non-existent field should not exist")
	}
}

func TestHashStorage_HDel(t *testing.T) {
	db := NewDatabase(0)
	hashStorage := db.HashStorage

	// Set some fields
	fields := map[string]string{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
	}
	hashStorage.HMSet("hash1", fields)

	// Delete some fields
	deletedCount := hashStorage.HDel("hash1", []string{"field1", "field3", "nonexistent"})
	if deletedCount != 2 {
		t.Errorf("Expected 2 deleted fields, got %d", deletedCount)
	}

	// Verify deletions
	if hashStorage.HExists("hash1", "field1") {
		t.Error("field1 should be deleted")
	}
	if !hashStorage.HExists("hash1", "field2") {
		t.Error("field2 should still exist")
	}
	if hashStorage.HExists("hash1", "field3") {
		t.Error("field3 should be deleted")
	}

	// Delete remaining field (should delete the hash)
	deletedCount = hashStorage.HDel("hash1", []string{"field2"})
	if deletedCount != 1 {
		t.Errorf("Expected 1 deleted field, got %d", deletedCount)
	}

	// Hash should be completely gone
	result, _ := hashStorage.HGetAll("hash1")
	if len(result) != 0 {
		t.Error("Hash should be completely deleted")
	}
}

func TestHashStorage_HLen(t *testing.T) {
	db := NewDatabase(0)
	hashStorage := db.HashStorage

	// Test HLen on non-existent hash
	length := hashStorage.HLen("nonexistent")
	if length != 0 {
		t.Errorf("Expected length 0, got %d", length)
	}

	// Set some fields
	fields := map[string]string{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
	}
	hashStorage.HMSet("hash1", fields)

	// Test HLen
	length = hashStorage.HLen("hash1")
	if length != 3 {
		t.Errorf("Expected length 3, got %d", length)
	}
}

func TestHashStorage_HKeys_HVals(t *testing.T) {
	db := NewDatabase(0)
	hashStorage := db.HashStorage

	// Test on non-existent hash
	keys := hashStorage.HKeys("nonexistent")
	if len(keys) != 0 {
		t.Errorf("Expected empty keys, got %d", len(keys))
	}

	vals := hashStorage.HVals("nonexistent")
	if len(vals) != 0 {
		t.Errorf("Expected empty values, got %d", len(vals))
	}

	// Set some fields
	fields := map[string]string{
		"field1": "value1",
		"field2": "value2",
	}
	hashStorage.HMSet("hash1", fields)

	// Test HKeys
	keys = hashStorage.HKeys("hash1")
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	// Check if keys contain expected fields (order may vary)
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}
	if !keyMap["field1"] || !keyMap["field2"] {
		t.Error("Keys should contain field1 and field2")
	}

	// Test HVals
	vals = hashStorage.HVals("hash1")
	if len(vals) != 2 {
		t.Errorf("Expected 2 values, got %d", len(vals))
	}

	// Check if values contain expected values (order may vary)
	valMap := make(map[string]bool)
	for _, val := range vals {
		valMap[val] = true
	}
	if !valMap["value1"] || !valMap["value2"] {
		t.Error("Values should contain value1 and value2")
	}
}
