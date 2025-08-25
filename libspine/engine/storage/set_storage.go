package storage

import (
	"math/rand"
)

// SetStorageImpl implements SetStorage interface
type SetStorageImpl struct {
	db *Database
}

// NewSetStorage creates a new set storage instance
func NewSetStorage(db *Database) SetStorage {
	return &SetStorageImpl{db: db}
}

func (s *SetStorageImpl) SAdd(key string, members []string) (int64, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()

	var setData map[string]struct{}

	if val, exists := s.db.data[key]; exists && !val.IsExpired() {
		if val.Type != TypeSet {
			return 0, ErrWrongType
		}
		if sd, ok := val.Data.(map[string]struct{}); ok {
			setData = sd
		} else {
			setData = make(map[string]struct{})
		}
	} else {
		setData = make(map[string]struct{})
	}

	added := int64(0)
	for _, member := range members {
		if _, exists := setData[member]; !exists {
			setData[member] = struct{}{}
			added++
		}
	}

	s.db.data[key] = &Value{
		Type:      TypeSet,
		Data:      setData,
		ExpiresAt: nil,
	}
	delete(s.db.expiry, key)

	return added, nil
}

func (s *SetStorageImpl) SRem(key string, members []string) (int64, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()

	value, exists := s.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeSet {
		return 0, nil
	}

	setData, ok := value.Data.(map[string]struct{})
	if !ok {
		return 0, nil
	}

	removed := int64(0)
	for _, member := range members {
		if _, exists := setData[member]; exists {
			delete(setData, member)
			removed++
		}
	}

	if len(setData) == 0 {
		delete(s.db.data, key)
		delete(s.db.expiry, key)
	}

	return removed, nil
}

func (s *SetStorageImpl) SIsMember(key, member string) bool {
	s.db.mu.RLock()
	defer s.db.mu.RUnlock()

	value, exists := s.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeSet {
		return false
	}

	if setData, ok := value.Data.(map[string]struct{}); ok {
		_, exists := setData[member]
		return exists
	}

	return false
}

func (s *SetStorageImpl) SMembers(key string) []string {
	s.db.mu.RLock()
	defer s.db.mu.RUnlock()

	value, exists := s.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeSet {
		return []string{}
	}

	if setData, ok := value.Data.(map[string]struct{}); ok {
		members := make([]string, 0, len(setData))
		for member := range setData {
			members = append(members, member)
		}
		return members
	}

	return []string{}
}

func (s *SetStorageImpl) SCard(key string) int64 {
	s.db.mu.RLock()
	defer s.db.mu.RUnlock()

	value, exists := s.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeSet {
		return 0
	}

	if setData, ok := value.Data.(map[string]struct{}); ok {
		return int64(len(setData))
	}

	return 0
}

func (s *SetStorageImpl) SPop(key string, count int64) []string {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()

	value, exists := s.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeSet {
		return []string{}
	}

	setData, ok := value.Data.(map[string]struct{})
	if !ok || len(setData) == 0 {
		return []string{}
	}

	members := make([]string, 0, len(setData))
	for member := range setData {
		members = append(members, member)
	}

	if count <= 0 {
		count = 1
	}
	if count > int64(len(members)) {
		count = int64(len(members))
	}

	// Shuffle and take count elements
	rand.Shuffle(len(members), func(i, j int) {
		members[i], members[j] = members[j], members[i]
	})

	result := members[:count]
	for _, member := range result {
		delete(setData, member)
	}

	if len(setData) == 0 {
		delete(s.db.data, key)
		delete(s.db.expiry, key)
	}

	return result
}

func (s *SetStorageImpl) SRandMember(key string, count int64) []string {
	s.db.mu.RLock()
	defer s.db.mu.RUnlock()

	value, exists := s.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeSet {
		return []string{}
	}

	if setData, ok := value.Data.(map[string]struct{}); ok {
		members := make([]string, 0, len(setData))
		for member := range setData {
			members = append(members, member)
		}

		if count <= 0 {
			count = 1
		}
		if count > int64(len(members)) {
			count = int64(len(members))
		}

		// Shuffle and take count elements
		rand.Shuffle(len(members), func(i, j int) {
			members[i], members[j] = members[j], members[i]
		})

		return members[:count]
	}

	return []string{}
}
