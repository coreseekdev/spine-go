package storage

import (
	"testing"
)

func TestZSetStorage_ZAdd_ZScore(t *testing.T) {
	db := NewDatabase(0)
	zsetStorage := db.ZSetStorage

	// Test ZAdd
	members := map[string]float64{
		"member1": 1.0,
		"member2": 2.5,
		"member3": 3.0,
	}
	added, err := zsetStorage.ZAdd("zset1", members)
	if err != nil {
		t.Errorf("ZAdd failed: %v", err)
	}
	if added != 3 {
		t.Errorf("Expected 3 added members, got %d", added)
	}

	// Test ZScore
	score, exists := zsetStorage.ZScore("zset1", "member2")
	if !exists {
		t.Error("Member should exist")
	}
	if score != 2.5 {
		t.Errorf("Expected score 2.5, got %f", score)
	}

	// Test ZScore for non-existent member
	_, exists = zsetStorage.ZScore("zset1", "nonexistent")
	if exists {
		t.Error("Non-existent member should not exist")
	}

	// Test ZAdd with existing member (update score)
	updateMembers := map[string]float64{
		"member2": 4.0,
		"member4": 5.0,
	}
	added, err = zsetStorage.ZAdd("zset1", updateMembers)
	if err != nil {
		t.Errorf("ZAdd failed: %v", err)
	}
	if added != 1 { // Only member4 is new
		t.Errorf("Expected 1 added member, got %d", added)
	}

	// Verify updated score
	score, exists = zsetStorage.ZScore("zset1", "member2")
	if !exists {
		t.Error("Member should exist")
	}
	if score != 4.0 {
		t.Errorf("Expected updated score 4.0, got %f", score)
	}
}

func TestZSetStorage_ZRem(t *testing.T) {
	db := NewDatabase(0)
	zsetStorage := db.ZSetStorage

	// Setup zset
	members := map[string]float64{
		"member1": 1.0,
		"member2": 2.0,
		"member3": 3.0,
		"member4": 4.0,
	}
	zsetStorage.ZAdd("zset1", members)

	// Test ZRem
	removed, err := zsetStorage.ZRem("zset1", []string{"member2", "member4", "nonexistent"})
	if err != nil {
		t.Errorf("ZRem failed: %v", err)
	}
	if removed != 2 {
		t.Errorf("Expected 2 removed members, got %d", removed)
	}

	// Verify removals
	_, exists := zsetStorage.ZScore("zset1", "member2")
	if exists {
		t.Error("member2 should be removed")
	}
	_, exists = zsetStorage.ZScore("zset1", "member4")
	if exists {
		t.Error("member4 should be removed")
	}

	// Verify remaining members
	cardinality := zsetStorage.ZCard("zset1")
	if cardinality != 2 {
		t.Errorf("Expected cardinality 2, got %d", cardinality)
	}
}

func TestZSetStorage_ZRank_ZRevRank(t *testing.T) {
	db := NewDatabase(0)
	zsetStorage := db.ZSetStorage

	// Setup zset with known order
	members := map[string]float64{
		"member1": 1.0,
		"member2": 2.0,
		"member3": 3.0,
	}
	zsetStorage.ZAdd("zset1", members)

	// Test ZRank (0-based, ascending order)
	rank, exists := zsetStorage.ZRank("zset1", "member1")
	if !exists {
		t.Error("Member should exist")
	}
	if rank != 0 {
		t.Errorf("Expected rank 0, got %d", rank)
	}

	rank, exists = zsetStorage.ZRank("zset1", "member3")
	if !exists {
		t.Error("Member should exist")
	}
	if rank != 2 {
		t.Errorf("Expected rank 2, got %d", rank)
	}

	// Test ZRevRank (0-based, descending order)
	revRank, exists := zsetStorage.ZRevRank("zset1", "member1")
	if !exists {
		t.Error("Member should exist")
	}
	if revRank != 2 {
		t.Errorf("Expected reverse rank 2, got %d", revRank)
	}

	revRank, exists = zsetStorage.ZRevRank("zset1", "member3")
	if !exists {
		t.Error("Member should exist")
	}
	if revRank != 0 {
		t.Errorf("Expected reverse rank 0, got %d", revRank)
	}

	// Test non-existent member
	_, exists = zsetStorage.ZRank("zset1", "nonexistent")
	if exists {
		t.Error("Non-existent member should not exist")
	}
}

func TestZSetStorage_ZRange_ZRevRange(t *testing.T) {
	db := NewDatabase(0)
	zsetStorage := db.ZSetStorage

	// Setup zset
	members := map[string]float64{
		"member1": 1.0,
		"member2": 2.0,
		"member3": 3.0,
		"member4": 4.0,
	}
	zsetStorage.ZAdd("zset1", members)

	// Test ZRange without scores
	result := zsetStorage.ZRange("zset1", 0, 2, false)
	expected := []interface{}{"member1", "member2", "member3"}
	if len(result) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(result))
	}
	for i, item := range result {
		if item != expected[i] {
			t.Errorf("Expected '%v' at index %d, got '%v'", expected[i], i, item)
		}
	}

	// Test ZRange with scores
	result = zsetStorage.ZRange("zset1", 0, 1, true)
	expected = []interface{}{"member1", 1.0, "member2", 2.0}
	if len(result) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(result))
	}
	for i, item := range result {
		if item != expected[i] {
			t.Errorf("Expected '%v' at index %d, got '%v'", expected[i], i, item)
		}
	}

	// Test ZRevRange without scores
	result = zsetStorage.ZRevRange("zset1", 0, 2, false)
	expected = []interface{}{"member4", "member3", "member2"}
	if len(result) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(result))
	}
	for i, item := range result {
		if item != expected[i] {
			t.Errorf("Expected '%v' at index %d, got '%v'", expected[i], i, item)
		}
	}

	// Test negative indices
	result = zsetStorage.ZRange("zset1", -2, -1, false)
	expected = []interface{}{"member3", "member4"}
	if len(result) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(result))
	}
	for i, item := range result {
		if item != expected[i] {
			t.Errorf("Expected '%v' at index %d, got '%v'", expected[i], i, item)
		}
	}
}

func TestZSetStorage_ZRangeByScore_ZRevRangeByScore(t *testing.T) {
	db := NewDatabase(0)
	zsetStorage := db.ZSetStorage

	// Setup zset
	members := map[string]float64{
		"member1": 1.0,
		"member2": 2.5,
		"member3": 3.0,
		"member4": 4.5,
		"member5": 5.0,
	}
	zsetStorage.ZAdd("zset1", members)

	// Test ZRangeByScore without scores
	result := zsetStorage.ZRangeByScore("zset1", 2.0, 4.0, false)
	expected := []interface{}{"member2", "member3"}
	if len(result) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(result))
	}
	for i, item := range result {
		if item != expected[i] {
			t.Errorf("Expected '%v' at index %d, got '%v'", expected[i], i, item)
		}
	}

	// Test ZRangeByScore with scores
	result = zsetStorage.ZRangeByScore("zset1", 2.0, 3.0, true)
	expected = []interface{}{"member2", 2.5, "member3", 3.0}
	if len(result) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(result))
	}
	for i, item := range result {
		if item != expected[i] {
			t.Errorf("Expected '%v' at index %d, got '%v'", expected[i], i, item)
		}
	}

	// Test ZRevRangeByScore
	result = zsetStorage.ZRevRangeByScore("zset1", 4.0, 2.0, false)
	expected = []interface{}{"member3", "member2"}
	if len(result) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(result))
	}
	for i, item := range result {
		if item != expected[i] {
			t.Errorf("Expected '%v' at index %d, got '%v'", expected[i], i, item)
		}
	}
}

func TestZSetStorage_ZCount(t *testing.T) {
	db := NewDatabase(0)
	zsetStorage := db.ZSetStorage

	// Test on non-existent zset
	count := zsetStorage.ZCount("nonexistent", 1.0, 5.0)
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Setup zset
	members := map[string]float64{
		"member1": 1.0,
		"member2": 2.5,
		"member3": 3.0,
		"member4": 4.5,
		"member5": 5.0,
	}
	zsetStorage.ZAdd("zset1", members)

	// Test ZCount
	count = zsetStorage.ZCount("zset1", 2.0, 4.0)
	if count != 2 { // member2 (2.5) and member3 (3.0)
		t.Errorf("Expected count 2, got %d", count)
	}

	// Test ZCount with inclusive bounds
	count = zsetStorage.ZCount("zset1", 2.5, 4.5)
	if count != 3 { // member2 (2.5), member3 (3.0), member4 (4.5)
		t.Errorf("Expected count 3, got %d", count)
	}

	// Test ZCount with no matches
	count = zsetStorage.ZCount("zset1", 10.0, 20.0)
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

func TestZSetStorage_ZCard(t *testing.T) {
	db := NewDatabase(0)
	zsetStorage := db.ZSetStorage

	// Test on non-existent zset
	cardinality := zsetStorage.ZCard("nonexistent")
	if cardinality != 0 {
		t.Errorf("Expected cardinality 0, got %d", cardinality)
	}

	// Setup zset
	members := map[string]float64{
		"member1": 1.0,
		"member2": 2.0,
		"member3": 3.0,
	}
	zsetStorage.ZAdd("zset1", members)

	// Test ZCard
	cardinality = zsetStorage.ZCard("zset1")
	if cardinality != 3 {
		t.Errorf("Expected cardinality 3, got %d", cardinality)
	}

	// Remove a member and test again
	zsetStorage.ZRem("zset1", []string{"member2"})
	cardinality = zsetStorage.ZCard("zset1")
	if cardinality != 2 {
		t.Errorf("Expected cardinality 2, got %d", cardinality)
	}
}
