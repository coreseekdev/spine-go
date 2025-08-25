package storage

import (
	"sort"
	"testing"
)

func TestSetStorage_SAdd_SMembers(t *testing.T) {
	db := NewDatabase(0)
	setStorage := db.SetStorage

	// Test SAdd
	added, err := setStorage.SAdd("set1", []string{"member1", "member2", "member3"})
	if err != nil {
		t.Errorf("SAdd failed: %v", err)
	}
	if added != 3 {
		t.Errorf("Expected 3 added members, got %d", added)
	}

	// Test SAdd with duplicate members
	added, err = setStorage.SAdd("set1", []string{"member2", "member4"})
	if err != nil {
		t.Errorf("SAdd failed: %v", err)
	}
	if added != 1 {
		t.Errorf("Expected 1 added member, got %d", added)
	}

	// Test SMembers
	members := setStorage.SMembers("set1")
	if len(members) != 4 {
		t.Errorf("Expected 4 members, got %d", len(members))
	}

	// Sort for consistent comparison
	sort.Strings(members)
	expected := []string{"member1", "member2", "member3", "member4"}
	sort.Strings(expected)

	for i, member := range members {
		if member != expected[i] {
			t.Errorf("Expected '%s' at index %d, got '%s'", expected[i], i, member)
		}
	}
}

func TestSetStorage_SRem(t *testing.T) {
	db := NewDatabase(0)
	setStorage := db.SetStorage

	// Setup set
	setStorage.SAdd("set1", []string{"member1", "member2", "member3", "member4"})

	// Test SRem
	removed, err := setStorage.SRem("set1", []string{"member2", "member4", "nonexistent"})
	if err != nil {
		t.Errorf("SRem failed: %v", err)
	}
	if removed != 2 {
		t.Errorf("Expected 2 removed members, got %d", removed)
	}

	// Verify remaining members
	members := setStorage.SMembers("set1")
	if len(members) != 2 {
		t.Errorf("Expected 2 remaining members, got %d", len(members))
	}

	sort.Strings(members)
	expected := []string{"member1", "member3"}
	sort.Strings(expected)

	for i, member := range members {
		if member != expected[i] {
			t.Errorf("Expected '%s' at index %d, got '%s'", expected[i], i, member)
		}
	}

	// Remove all remaining members
	removed, err = setStorage.SRem("set1", []string{"member1", "member3"})
	if err != nil {
		t.Errorf("SRem failed: %v", err)
	}
	if removed != 2 {
		t.Errorf("Expected 2 removed members, got %d", removed)
	}

	// Set should be empty
	members = setStorage.SMembers("set1")
	if len(members) != 0 {
		t.Errorf("Expected empty set, got %d members", len(members))
	}
}

func TestSetStorage_SIsMember(t *testing.T) {
	db := NewDatabase(0)
	setStorage := db.SetStorage

	// Test on non-existent set
	if setStorage.SIsMember("nonexistent", "member1") {
		t.Error("SIsMember should return false for non-existent set")
	}

	// Setup set
	setStorage.SAdd("set1", []string{"member1", "member2"})

	// Test existing member
	if !setStorage.SIsMember("set1", "member1") {
		t.Error("SIsMember should return true for existing member")
	}

	// Test non-existent member
	if setStorage.SIsMember("set1", "nonexistent") {
		t.Error("SIsMember should return false for non-existent member")
	}
}

func TestSetStorage_SCard(t *testing.T) {
	db := NewDatabase(0)
	setStorage := db.SetStorage

	// Test on non-existent set
	cardinality := setStorage.SCard("nonexistent")
	if cardinality != 0 {
		t.Errorf("Expected cardinality 0, got %d", cardinality)
	}

	// Setup set
	setStorage.SAdd("set1", []string{"member1", "member2", "member3"})

	// Test cardinality
	cardinality = setStorage.SCard("set1")
	if cardinality != 3 {
		t.Errorf("Expected cardinality 3, got %d", cardinality)
	}
}

func TestSetStorage_SPop(t *testing.T) {
	db := NewDatabase(0)
	setStorage := db.SetStorage

	// Test on non-existent set
	popped := setStorage.SPop("nonexistent", 1)
	if len(popped) != 0 {
		t.Errorf("Expected empty result, got %d members", len(popped))
	}

	// Setup set
	setStorage.SAdd("set1", []string{"member1", "member2", "member3", "member4"})

	// Test SPop with count 2
	popped = setStorage.SPop("set1", 2)
	if len(popped) != 2 {
		t.Errorf("Expected 2 popped members, got %d", len(popped))
	}

	// Verify members were removed
	remaining := setStorage.SMembers("set1")
	if len(remaining) != 2 {
		t.Errorf("Expected 2 remaining members, got %d", len(remaining))
	}

	// Verify popped members were actually in the set
	allMembers := []string{"member1", "member2", "member3", "member4"}
	for _, poppedMember := range popped {
		found := false
		for _, member := range allMembers {
			if member == poppedMember {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Popped member '%s' was not in original set", poppedMember)
		}
	}

	// Test SPop with count larger than set size
	popped = setStorage.SPop("set1", 10)
	if len(popped) != 2 {
		t.Errorf("Expected 2 popped members (all remaining), got %d", len(popped))
	}

	// Set should be empty now
	cardinality := setStorage.SCard("set1")
	if cardinality != 0 {
		t.Errorf("Expected empty set, got cardinality %d", cardinality)
	}
}

func TestSetStorage_SRandMember(t *testing.T) {
	db := NewDatabase(0)
	setStorage := db.SetStorage

	// Test on non-existent set
	random := setStorage.SRandMember("nonexistent", 1)
	if len(random) != 0 {
		t.Errorf("Expected empty result, got %d members", len(random))
	}

	// Setup set
	setStorage.SAdd("set1", []string{"member1", "member2", "member3", "member4"})

	// Test SRandMember with count 2
	random = setStorage.SRandMember("set1", 2)
	if len(random) != 2 {
		t.Errorf("Expected 2 random members, got %d", len(random))
	}

	// Verify members were NOT removed (unlike SPop)
	remaining := setStorage.SMembers("set1")
	if len(remaining) != 4 {
		t.Errorf("Expected 4 remaining members, got %d", len(remaining))
	}

	// Verify random members were actually in the set
	allMembers := []string{"member1", "member2", "member3", "member4"}
	for _, randomMember := range random {
		found := false
		for _, member := range allMembers {
			if member == randomMember {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Random member '%s' was not in original set", randomMember)
		}
	}

	// Test SRandMember with count larger than set size
	random = setStorage.SRandMember("set1", 10)
	if len(random) != 4 {
		t.Errorf("Expected 4 random members (all available), got %d", len(random))
	}
}
