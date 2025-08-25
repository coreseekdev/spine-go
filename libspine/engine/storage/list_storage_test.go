package storage

import (
	"testing"
)

func TestListStorage_LPush_RPush(t *testing.T) {
	db := NewDatabase(0)
	listStorage := db.ListStorage

	// Test LPush
	length, err := listStorage.LPush("list1", []string{"value1", "value2"})
	if err != nil {
		t.Errorf("LPush failed: %v", err)
	}
	if length != 2 {
		t.Errorf("Expected length 2, got %d", length)
	}

	// Test RPush
	length, err = listStorage.RPush("list1", []string{"value3", "value4"})
	if err != nil {
		t.Errorf("RPush failed: %v", err)
	}
	if length != 4 {
		t.Errorf("Expected length 4, got %d", length)
	}

	// Verify order: [value2, value1, value3, value4]
	// LPush: ["value1", "value2"] results in ["value2", "value1"] (Redis behavior)
	// Then RPush appends: ["value2", "value1"] + ["value3", "value4"] = ["value2", "value1", "value3", "value4"]
	items := listStorage.LRange("list1", 0, -1)
	expected := []string{"value2", "value1", "value3", "value4"}
	if len(items) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(items))
	}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected '%s' at index %d, got '%s'", expected[i], i, item)
		}
	}
}

func TestListStorage_LPop_RPop(t *testing.T) {
	db := NewDatabase(0)
	listStorage := db.ListStorage

	// Test pop on non-existent list
	_, exists := listStorage.LPop("nonexistent")
	if exists {
		t.Error("LPop should return false for non-existent list")
	}

	// Setup list
	listStorage.RPush("list1", []string{"value1", "value2", "value3"})

	// Test LPop
	value, exists := listStorage.LPop("list1")
	if !exists {
		t.Error("LPop should return true for existing list")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	// Test RPop
	value, exists = listStorage.RPop("list1")
	if !exists {
		t.Error("RPop should return true for existing list")
	}
	if value != "value3" {
		t.Errorf("Expected 'value3', got '%s'", value)
	}

	// Verify remaining item
	length := listStorage.LLen("list1")
	if length != 1 {
		t.Errorf("Expected length 1, got %d", length)
	}

	// Pop last item
	value, exists = listStorage.LPop("list1")
	if !exists {
		t.Error("LPop should return true for last item")
	}
	if value != "value2" {
		t.Errorf("Expected 'value2', got '%s'", value)
	}

	// List should be empty now
	length = listStorage.LLen("list1")
	if length != 0 {
		t.Errorf("Expected length 0, got %d", length)
	}
}

func TestListStorage_LLen(t *testing.T) {
	db := NewDatabase(0)
	listStorage := db.ListStorage

	// Test LLen on non-existent list
	length := listStorage.LLen("nonexistent")
	if length != 0 {
		t.Errorf("Expected length 0, got %d", length)
	}

	// Setup list
	listStorage.RPush("list1", []string{"value1", "value2", "value3"})

	// Test LLen
	length = listStorage.LLen("list1")
	if length != 3 {
		t.Errorf("Expected length 3, got %d", length)
	}
}

func TestListStorage_LIndex(t *testing.T) {
	db := NewDatabase(0)
	listStorage := db.ListStorage

	// Test LIndex on non-existent list
	_, exists := listStorage.LIndex("nonexistent", 0)
	if exists {
		t.Error("LIndex should return false for non-existent list")
	}

	// Setup list
	listStorage.RPush("list1", []string{"value1", "value2", "value3"})

	// Test positive indices
	value, exists := listStorage.LIndex("list1", 0)
	if !exists {
		t.Error("LIndex should return true for valid index")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	value, exists = listStorage.LIndex("list1", 2)
	if !exists {
		t.Error("LIndex should return true for valid index")
	}
	if value != "value3" {
		t.Errorf("Expected 'value3', got '%s'", value)
	}

	// Test negative indices
	value, exists = listStorage.LIndex("list1", -1)
	if !exists {
		t.Error("LIndex should return true for valid negative index")
	}
	if value != "value3" {
		t.Errorf("Expected 'value3', got '%s'", value)
	}

	value, exists = listStorage.LIndex("list1", -3)
	if !exists {
		t.Error("LIndex should return true for valid negative index")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	// Test out of bounds
	_, exists = listStorage.LIndex("list1", 5)
	if exists {
		t.Error("LIndex should return false for out of bounds index")
	}

	_, exists = listStorage.LIndex("list1", -5)
	if exists {
		t.Error("LIndex should return false for out of bounds negative index")
	}
}

func TestListStorage_LSet(t *testing.T) {
	db := NewDatabase(0)
	listStorage := db.ListStorage

	// Test LSet on non-existent list
	err := listStorage.LSet("nonexistent", 0, "value")
	if err != ErrNoSuchKey {
		t.Errorf("Expected ErrNoSuchKey, got %v", err)
	}

	// Setup list
	listStorage.RPush("list1", []string{"value1", "value2", "value3"})

	// Test LSet with valid index
	err = listStorage.LSet("list1", 1, "newvalue2")
	if err != nil {
		t.Errorf("LSet failed: %v", err)
	}

	// Verify change
	value, exists := listStorage.LIndex("list1", 1)
	if !exists {
		t.Error("Index should exist")
	}
	if value != "newvalue2" {
		t.Errorf("Expected 'newvalue2', got '%s'", value)
	}

	// Test LSet with negative index
	err = listStorage.LSet("list1", -1, "newvalue3")
	if err != nil {
		t.Errorf("LSet failed: %v", err)
	}

	// Verify change
	value, exists = listStorage.LIndex("list1", -1)
	if !exists {
		t.Error("Index should exist")
	}
	if value != "newvalue3" {
		t.Errorf("Expected 'newvalue3', got '%s'", value)
	}

	// Test LSet with out of bounds index
	err = listStorage.LSet("list1", 5, "value")
	if err != ErrIndexOutOfRange {
		t.Errorf("Expected ErrIndexOutOfRange, got %v", err)
	}
}

func TestListStorage_LRange(t *testing.T) {
	db := NewDatabase(0)
	listStorage := db.ListStorage

	// Test LRange on non-existent list
	items := listStorage.LRange("nonexistent", 0, -1)
	if len(items) != 0 {
		t.Errorf("Expected empty result, got %d items", len(items))
	}

	// Setup list
	listStorage.RPush("list1", []string{"value1", "value2", "value3", "value4", "value5"})

	// Test full range
	items = listStorage.LRange("list1", 0, -1)
	expected := []string{"value1", "value2", "value3", "value4", "value5"}
	if len(items) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(items))
	}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected '%s' at index %d, got '%s'", expected[i], i, item)
		}
	}

	// Test partial range
	items = listStorage.LRange("list1", 1, 3)
	expected = []string{"value2", "value3", "value4"}
	if len(items) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(items))
	}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected '%s' at index %d, got '%s'", expected[i], i, item)
		}
	}

	// Test negative indices
	items = listStorage.LRange("list1", -3, -1)
	expected = []string{"value3", "value4", "value5"}
	if len(items) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(items))
	}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected '%s' at index %d, got '%s'", expected[i], i, item)
		}
	}

	// Test out of bounds
	items = listStorage.LRange("list1", 10, 20)
	if len(items) != 0 {
		t.Errorf("Expected empty result for out of bounds range, got %d items", len(items))
	}
}

func TestListStorage_LTrim(t *testing.T) {
	db := NewDatabase(0)
	listStorage := db.ListStorage

	// Setup list
	listStorage.RPush("list1", []string{"value1", "value2", "value3", "value4", "value5"})

	// Test LTrim
	err := listStorage.LTrim("list1", 1, 3)
	if err != nil {
		t.Errorf("LTrim failed: %v", err)
	}

	// Verify trimmed list
	items := listStorage.LRange("list1", 0, -1)
	expected := []string{"value2", "value3", "value4"}
	if len(items) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(items))
	}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected '%s' at index %d, got '%s'", expected[i], i, item)
		}
	}

	// Test LTrim that removes all elements
	err = listStorage.LTrim("list1", 10, 20)
	if err != nil {
		t.Errorf("LTrim failed: %v", err)
	}

	// List should be empty
	length := listStorage.LLen("list1")
	if length != 0 {
		t.Errorf("Expected length 0, got %d", length)
	}
}

func TestListStorage_LRem(t *testing.T) {
	db := NewDatabase(0)
	listStorage := db.ListStorage

	// Setup list with duplicates
	listStorage.RPush("list1", []string{"a", "b", "a", "c", "a", "b"})

	// Test LRem with count > 0 (remove first N occurrences)
	removed := listStorage.LRem("list1", 2, "a")
	if removed != 2 {
		t.Errorf("Expected 2 removed, got %d", removed)
	}

	// Verify result: ["b", "c", "a", "b"]
	items := listStorage.LRange("list1", 0, -1)
	expected := []string{"b", "c", "a", "b"}
	if len(items) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(items))
	}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected '%s' at index %d, got '%s'", expected[i], i, item)
		}
	}

	// Test LRem with count < 0 (remove last N occurrences)
	removed = listStorage.LRem("list1", -1, "b")
	if removed != 1 {
		t.Errorf("Expected 1 removed, got %d", removed)
	}

	// Verify result: ["b", "c", "a"]
	items = listStorage.LRange("list1", 0, -1)
	expected = []string{"b", "c", "a"}
	if len(items) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(items))
	}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected '%s' at index %d, got '%s'", expected[i], i, item)
		}
	}

	// Test LRem with count = 0 (remove all occurrences)
	listStorage.RPush("list2", []string{"x", "y", "x", "z", "x"})
	removed = listStorage.LRem("list2", 0, "x")
	if removed != 3 {
		t.Errorf("Expected 3 removed, got %d", removed)
	}

	// Verify result: ["y", "z"]
	items = listStorage.LRange("list2", 0, -1)
	expected = []string{"y", "z"}
	if len(items) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(items))
	}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected '%s' at index %d, got '%s'", expected[i], i, item)
		}
	}
}
