package storage

import ()

// HashStorageImpl implements HashStorage interface
type HashStorageImpl struct {
	db *Database
}

// NewHashStorage creates a new hash storage instance
func NewHashStorage(db *Database) HashStorage {
	return &HashStorageImpl{db: db}
}

func (h *HashStorageImpl) HSet(key, field, value string) (bool, error) {
	h.db.mu.Lock()
	defer h.db.mu.Unlock()

	var hashData map[string]string
	isNewField := true

	if val, exists := h.db.data[key]; exists && !val.IsExpired() {
		if val.Type != TypeHash {
			return false, ErrWrongType
		}
		if hd, ok := val.Data.(map[string]string); ok {
			hashData = hd
			_, isNewField = hashData[field]
			isNewField = !isNewField
		} else {
			hashData = make(map[string]string)
		}
	} else {
		hashData = make(map[string]string)
	}

	hashData[field] = value
	h.db.data[key] = &Value{
		Type:      TypeHash,
		Data:      hashData,
		ExpiresAt: nil,
	}
	delete(h.db.expiry, key)

	return isNewField, nil
}

func (h *HashStorageImpl) HGet(key, field string) (string, bool) {
	h.db.mu.RLock()
	defer h.db.mu.RUnlock()

	value, exists := h.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeHash {
		return "", false
	}

	if hashData, ok := value.Data.(map[string]string); ok {
		val, exists := hashData[field]
		return val, exists
	}

	return "", false
}

func (h *HashStorageImpl) HMSet(key string, fields map[string]string) error {
	h.db.mu.Lock()
	defer h.db.mu.Unlock()

	var hashData map[string]string

	if val, exists := h.db.data[key]; exists && !val.IsExpired() {
		if val.Type != TypeHash {
			return ErrWrongType
		}
		if hd, ok := val.Data.(map[string]string); ok {
			hashData = hd
		} else {
			hashData = make(map[string]string)
		}
	} else {
		hashData = make(map[string]string)
	}

	for field, value := range fields {
		hashData[field] = value
	}

	h.db.data[key] = &Value{
		Type:      TypeHash,
		Data:      hashData,
		ExpiresAt: nil,
	}
	delete(h.db.expiry, key)

	return nil
}

func (h *HashStorageImpl) HMGet(key string, fields []string) map[string]string {
	h.db.mu.RLock()
	defer h.db.mu.RUnlock()

	result := make(map[string]string)
	value, exists := h.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeHash {
		return result
	}

	if hashData, ok := value.Data.(map[string]string); ok {
		for _, field := range fields {
			if val, exists := hashData[field]; exists {
				result[field] = val
			}
		}
	}

	return result
}

func (h *HashStorageImpl) HGetAll(key string) (map[string]string, error) {
	h.db.mu.RLock()
	defer h.db.mu.RUnlock()

	value, exists := h.db.data[key]
	if !exists || value.IsExpired() {
		return make(map[string]string), nil
	}

	if value.Type != TypeHash {
		return nil, ErrWrongType
	}

	if hashData, ok := value.Data.(map[string]string); ok {
		result := make(map[string]string)
		for k, v := range hashData {
			result[k] = v
		}
		return result, nil
	}

	return make(map[string]string), nil
}

func (h *HashStorageImpl) HExists(key, field string) bool {
	h.db.mu.RLock()
	defer h.db.mu.RUnlock()

	value, exists := h.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeHash {
		return false
	}

	if hashData, ok := value.Data.(map[string]string); ok {
		_, exists := hashData[field]
		return exists
	}

	return false
}

func (h *HashStorageImpl) HDel(key string, fields []string) int64 {
	h.db.mu.Lock()
	defer h.db.mu.Unlock()

	value, exists := h.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeHash {
		return 0
	}

	hashData, ok := value.Data.(map[string]string)
	if !ok {
		return 0
	}

	deletedCount := int64(0)
	for _, field := range fields {
		if _, exists := hashData[field]; exists {
			delete(hashData, field)
			deletedCount++
		}
	}

	if len(hashData) == 0 {
		delete(h.db.data, key)
		delete(h.db.expiry, key)
	}

	return deletedCount
}

func (h *HashStorageImpl) HLen(key string) int64 {
	h.db.mu.RLock()
	defer h.db.mu.RUnlock()

	value, exists := h.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeHash {
		return 0
	}

	if hashData, ok := value.Data.(map[string]string); ok {
		return int64(len(hashData))
	}

	return 0
}

func (h *HashStorageImpl) HKeys(key string) []string {
	h.db.mu.RLock()
	defer h.db.mu.RUnlock()

	value, exists := h.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeHash {
		return []string{}
	}

	if hashData, ok := value.Data.(map[string]string); ok {
		keys := make([]string, 0, len(hashData))
		for k := range hashData {
			keys = append(keys, k)
		}
		return keys
	}

	return []string{}
}

func (h *HashStorageImpl) HVals(key string) []string {
	h.db.mu.RLock()
	defer h.db.mu.RUnlock()

	value, exists := h.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeHash {
		return []string{}
	}

	if hashData, ok := value.Data.(map[string]string); ok {
		values := make([]string, 0, len(hashData))
		for _, v := range hashData {
			values = append(values, v)
		}
		return values
	}

	return []string{}
}
