package storage

import ()

// ListStorageImpl implements ListStorage interface
type ListStorageImpl struct {
	db *Database
}

// NewListStorage creates a new list storage instance
func NewListStorage(db *Database) ListStorage {
	return &ListStorageImpl{db: db}
}

func (l *ListStorageImpl) LPush(key string, values []string) (int64, error) {
	l.db.mu.Lock()
	defer l.db.mu.Unlock()

	var listData []string

	if val, exists := l.db.data[key]; exists && !val.IsExpired() {
		if val.Type != TypeList {
			return 0, ErrWrongType
		}
		if ld, ok := val.Data.([]string); ok {
			listData = ld
		} else {
			listData = []string{}
		}
	} else {
		listData = []string{}
	}

	// Prepend values in reverse order to maintain Redis behavior
	// Redis LPUSH value1 value2 results in [value2, value1]
	for _, value := range values {
		listData = append([]string{value}, listData...)
	}

	l.db.data[key] = &Value{
		Type:      TypeList,
		Data:      listData,
		ExpiresAt: nil,
	}
	delete(l.db.expiry, key)

	return int64(len(listData)), nil
}

func (l *ListStorageImpl) RPush(key string, values []string) (int64, error) {
	l.db.mu.Lock()
	defer l.db.mu.Unlock()

	var listData []string

	if val, exists := l.db.data[key]; exists && !val.IsExpired() {
		if val.Type != TypeList {
			return 0, ErrWrongType
		}
		if ld, ok := val.Data.([]string); ok {
			listData = ld
		} else {
			listData = []string{}
		}
	} else {
		listData = []string{}
	}

	listData = append(listData, values...)

	l.db.data[key] = &Value{
		Type:      TypeList,
		Data:      listData,
		ExpiresAt: nil,
	}
	delete(l.db.expiry, key)

	return int64(len(listData)), nil
}

func (l *ListStorageImpl) LPop(key string) (string, bool) {
	l.db.mu.Lock()
	defer l.db.mu.Unlock()

	value, exists := l.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeList {
		return "", false
	}

	listData, ok := value.Data.([]string)
	if !ok || len(listData) == 0 {
		return "", false
	}

	result := listData[0]
	listData = listData[1:]

	if len(listData) == 0 {
		delete(l.db.data, key)
		delete(l.db.expiry, key)
	} else {
		l.db.data[key].Data = listData
	}

	return result, true
}

func (l *ListStorageImpl) RPop(key string) (string, bool) {
	l.db.mu.Lock()
	defer l.db.mu.Unlock()

	value, exists := l.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeList {
		return "", false
	}

	listData, ok := value.Data.([]string)
	if !ok || len(listData) == 0 {
		return "", false
	}

	result := listData[len(listData)-1]
	listData = listData[:len(listData)-1]

	if len(listData) == 0 {
		delete(l.db.data, key)
		delete(l.db.expiry, key)
	} else {
		l.db.data[key].Data = listData
	}

	return result, true
}

func (l *ListStorageImpl) LLen(key string) int64 {
	l.db.mu.RLock()
	defer l.db.mu.RUnlock()

	value, exists := l.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeList {
		return 0
	}

	if listData, ok := value.Data.([]string); ok {
		return int64(len(listData))
	}

	return 0
}

func (l *ListStorageImpl) LIndex(key string, index int64) (string, bool) {
	l.db.mu.RLock()
	defer l.db.mu.RUnlock()

	value, exists := l.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeList {
		return "", false
	}

	listData, ok := value.Data.([]string)
	if !ok {
		return "", false
	}

	length := int64(len(listData))
	if index < 0 {
		index = length + index
	}

	if index < 0 || index >= length {
		return "", false
	}

	return listData[index], true
}

func (l *ListStorageImpl) LSet(key string, index int64, value string) error {
	l.db.mu.Lock()
	defer l.db.mu.Unlock()

	val, exists := l.db.data[key]
	if !exists || val.IsExpired() || val.Type != TypeList {
		return ErrNoSuchKey
	}

	listData, ok := val.Data.([]string)
	if !ok {
		return ErrNoSuchKey
	}

	length := int64(len(listData))
	if index < 0 {
		index = length + index
	}

	if index < 0 || index >= length {
		return ErrIndexOutOfRange
	}

	listData[index] = value
	return nil
}

func (l *ListStorageImpl) LRange(key string, start, stop int64) []string {
	l.db.mu.RLock()
	defer l.db.mu.RUnlock()

	value, exists := l.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeList {
		return []string{}
	}

	listData, ok := value.Data.([]string)
	if !ok {
		return []string{}
	}

	length := int64(len(listData))
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}

	if start > stop || start >= length {
		return []string{}
	}

	result := make([]string, stop-start+1)
	copy(result, listData[start:stop+1])
	return result
}

func (l *ListStorageImpl) LTrim(key string, start, stop int64) error {
	l.db.mu.Lock()
	defer l.db.mu.Unlock()

	value, exists := l.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeList {
		return nil
	}

	listData, ok := value.Data.([]string)
	if !ok {
		return nil
	}

	length := int64(len(listData))
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}

	if start > stop || start >= length {
		delete(l.db.data, key)
		delete(l.db.expiry, key)
		return nil
	}

	trimmed := make([]string, stop-start+1)
	copy(trimmed, listData[start:stop+1])
	l.db.data[key].Data = trimmed

	return nil
}

func (l *ListStorageImpl) LRem(key string, count int64, value string) int64 {
	l.db.mu.Lock()
	defer l.db.mu.Unlock()

	val, exists := l.db.data[key]
	if !exists || val.IsExpired() || val.Type != TypeList {
		return 0
	}

	listData, ok := val.Data.([]string)
	if !ok {
		return 0
	}

	var newList []string
	removed := int64(0)

	if count == 0 {
		// Remove all occurrences
		for _, item := range listData {
			if item != value {
				newList = append(newList, item)
			} else {
				removed++
			}
		}
	} else if count > 0 {
		// Remove first count occurrences
		for _, item := range listData {
			if item == value && removed < count {
				removed++
			} else {
				newList = append(newList, item)
			}
		}
	} else {
		// Remove last |count| occurrences
		count = -count
		for i := len(listData) - 1; i >= 0; i-- {
			item := listData[i]
			if item == value && removed < count {
				removed++
			} else {
				newList = append([]string{item}, newList...)
			}
		}
	}

	if len(newList) == 0 {
		delete(l.db.data, key)
		delete(l.db.expiry, key)
	} else {
		l.db.data[key].Data = newList
	}

	return removed
}
