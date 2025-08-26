package storage

import (
	"time"
)

// CommonStorageImpl implements CommonStorage interface
type CommonStorageImpl struct {
	db *Database
}

// NewCommonStorage creates a new common storage instance
func NewCommonStorage(db *Database) CommonStorage {
	return &CommonStorageImpl{db: db}
}

func (c *CommonStorageImpl) Exists(key string) bool {
	c.db.mu.RLock()
	defer c.db.mu.RUnlock()

	value, exists := c.db.data[key]
	if !exists {
		return false
	}

	if value.IsExpired() {
		c.db.mu.RUnlock()
		c.db.mu.Lock()
		delete(c.db.data, key)
		delete(c.db.expiry, key)
		c.db.mu.Unlock()
		c.db.mu.RLock()
		return false
	}

	return true
}

func (c *CommonStorageImpl) Del(keys []string) int64 {
	c.db.mu.Lock()
	defer c.db.mu.Unlock()

	deleted := int64(0)
	for _, key := range keys {
		if _, exists := c.db.data[key]; exists {
			delete(c.db.data, key)
			delete(c.db.expiry, key)
			deleted++
		}
	}

	return deleted
}

func (c *CommonStorageImpl) Type(key string) ValueType {
	c.db.mu.RLock()
	defer c.db.mu.RUnlock()

	value, exists := c.db.data[key]
	if !exists || value.IsExpired() {
		return ValueType(-1) // Non-existent key
	}

	return value.Type
}

func (c *CommonStorageImpl) TTL(key string) time.Duration {
	c.db.mu.RLock()
	defer c.db.mu.RUnlock()

	value, exists := c.db.data[key]
	if !exists || value.IsExpired() {
		return -2 * time.Second // Key does not exist
	}

	if value.ExpiresAt == nil {
		return -1 * time.Second // Key exists but has no expiration
	}

	return time.Until(*value.ExpiresAt)
}

func (c *CommonStorageImpl) Expire(key string, expiration time.Duration) bool {
	c.db.mu.Lock()
	defer c.db.mu.Unlock()

	value, exists := c.db.data[key]
	if !exists || value.IsExpired() {
		return false
	}

	expiresAt := time.Now().Add(expiration)
	value.ExpiresAt = &expiresAt
	c.db.expiry[key] = &expiresAt

	return true
}

func (c *CommonStorageImpl) ExpireAt(key string, timestamp time.Time) bool {
	c.db.mu.Lock()
	defer c.db.mu.Unlock()

	value, exists := c.db.data[key]
	if !exists || value.IsExpired() {
		return false
	}

	value.ExpiresAt = &timestamp
	c.db.expiry[key] = &timestamp

	return true
}

func (c *CommonStorageImpl) Persist(key string) bool {
	c.db.mu.Lock()
	defer c.db.mu.Unlock()

	value, exists := c.db.data[key]
	if !exists || value.IsExpired() {
		return false
	}

	if value.ExpiresAt == nil {
		return false // Key already has no expiration
	}

	value.ExpiresAt = nil
	delete(c.db.expiry, key)

	return true
}

func (c *CommonStorageImpl) SwapDB(db1, db2 int) error {
	// For now, return a simple implementation
	// In a real implementation, this would swap the contents of two databases
	return nil
}
