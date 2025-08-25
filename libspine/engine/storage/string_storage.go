package storage

import (
	"strconv"
	"time"
)

// StringStorageImpl implements StringStorage interface
type StringStorageImpl struct {
	db *Database
}

// NewStringStorage creates a new string storage instance
func NewStringStorage(db *Database) StringStorage {
	return &StringStorageImpl{db: db}
}

func (s *StringStorageImpl) Set(key, value string, expiration *time.Time) error {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()

	s.db.data[key] = &Value{
		Type:      TypeString,
		Data:      value,
		ExpiresAt: expiration,
	}

	if expiration != nil {
		s.db.expiry[key] = expiration
	} else {
		delete(s.db.expiry, key)
	}

	return nil
}

func (s *StringStorageImpl) Get(key string) (string, bool) {
	s.db.mu.RLock()
	defer s.db.mu.RUnlock()

	value, exists := s.db.data[key]
	if !exists {
		return "", false
	}

	if value.IsExpired() {
		s.db.mu.RUnlock()
		s.db.mu.Lock()
		delete(s.db.data, key)
		delete(s.db.expiry, key)
		s.db.mu.Unlock()
		s.db.mu.RLock()
		return "", false
	}

	if value.Type != TypeString {
		return "", false
	}

	if str, ok := value.Data.(string); ok {
		return str, true
	}

	return "", false
}

func (s *StringStorageImpl) MSet(pairs map[string]string) error {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()

	for key, value := range pairs {
		s.db.data[key] = &Value{
			Type:      TypeString,
			Data:      value,
			ExpiresAt: nil,
		}
		delete(s.db.expiry, key)
	}

	return nil
}

func (s *StringStorageImpl) MGet(keys []string) map[string]string {
	s.db.mu.RLock()
	defer s.db.mu.RUnlock()

	result := make(map[string]string)
	for _, key := range keys {
		if value, exists := s.db.data[key]; exists && !value.IsExpired() && value.Type == TypeString {
			if str, ok := value.Data.(string); ok {
				result[key] = str
			}
		}
	}

	return result
}

func (s *StringStorageImpl) Exists(key string) bool {
	s.db.mu.RLock()
	defer s.db.mu.RUnlock()

	value, exists := s.db.data[key]
	if !exists {
		return false
	}

	if value.IsExpired() {
		s.db.mu.RUnlock()
		s.db.mu.Lock()
		delete(s.db.data, key)
		delete(s.db.expiry, key)
		s.db.mu.Unlock()
		s.db.mu.RLock()
		return false
	}

	return value.Type == TypeString
}

func (s *StringStorageImpl) Del(key string) bool {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()

	if value, exists := s.db.data[key]; exists && value.Type == TypeString {
		delete(s.db.data, key)
		delete(s.db.expiry, key)
		return true
	}

	return false
}

func (s *StringStorageImpl) Incr(key string) (int64, error) {
	return s.IncrBy(key, 1)
}

func (s *StringStorageImpl) Decr(key string) (int64, error) {
	return s.DecrBy(key, 1)
}

func (s *StringStorageImpl) IncrBy(key string, increment int64) (int64, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()

	var currentValue int64 = 0
	if value, exists := s.db.data[key]; exists && !value.IsExpired() {
		if value.Type != TypeString {
			return 0, ErrWrongType
		}
		if str, ok := value.Data.(string); ok {
			if parsed, err := strconv.ParseInt(str, 10, 64); err == nil {
				currentValue = parsed
			} else {
				return 0, ErrNotInteger
			}
		}
	}

	newValue := currentValue + increment
	s.db.data[key] = &Value{
		Type:      TypeString,
		Data:      strconv.FormatInt(newValue, 10),
		ExpiresAt: nil,
	}
	delete(s.db.expiry, key)

	return newValue, nil
}

func (s *StringStorageImpl) DecrBy(key string, decrement int64) (int64, error) {
	return s.IncrBy(key, -decrement)
}

func (s *StringStorageImpl) Append(key, value string) (int64, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()

	var currentValue string
	if val, exists := s.db.data[key]; exists && !val.IsExpired() {
		if val.Type != TypeString {
			return 0, ErrWrongType
		}
		if str, ok := val.Data.(string); ok {
			currentValue = str
		}
	}

	newValue := currentValue + value
	s.db.data[key] = &Value{
		Type:      TypeString,
		Data:      newValue,
		ExpiresAt: nil,
	}
	delete(s.db.expiry, key)

	return int64(len(newValue)), nil
}

func (s *StringStorageImpl) StrLen(key string) int64 {
	s.db.mu.RLock()
	defer s.db.mu.RUnlock()

	value, exists := s.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeString {
		return 0
	}

	if str, ok := value.Data.(string); ok {
		return int64(len(str))
	}

	return 0
}
