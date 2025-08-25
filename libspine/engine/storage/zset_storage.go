package storage

import (
	"sort"
)

// ZSetMember represents a member in a sorted set
type ZSetMember struct {
	Member string
	Score  float64
}

// ZSetData represents the internal structure of a sorted set
type ZSetData struct {
	Members map[string]float64 // member -> score
	Scores  []ZSetMember       // sorted by score for range operations
}

// ZSetStorageImpl implements ZSetStorage interface
type ZSetStorageImpl struct {
	db *Database
}

// NewZSetStorage creates a new sorted set storage instance
func NewZSetStorage(db *Database) ZSetStorage {
	return &ZSetStorageImpl{db: db}
}

func (z *ZSetStorageImpl) ZAdd(key string, members map[string]float64) (int64, error) {
	z.db.mu.Lock()
	defer z.db.mu.Unlock()

	var zsetData *ZSetData

	if val, exists := z.db.data[key]; exists && !val.IsExpired() {
		if val.Type != TypeZSet {
			return 0, ErrWrongType
		}
		if zd, ok := val.Data.(*ZSetData); ok {
			zsetData = zd
		} else {
			zsetData = &ZSetData{
				Members: make(map[string]float64),
				Scores:  []ZSetMember{},
			}
		}
	} else {
		zsetData = &ZSetData{
			Members: make(map[string]float64),
			Scores:  []ZSetMember{},
		}
	}

	added := int64(0)
	for member, score := range members {
		if _, exists := zsetData.Members[member]; !exists {
			added++
		}
		zsetData.Members[member] = score
	}

	// Rebuild sorted scores
	zsetData.Scores = make([]ZSetMember, 0, len(zsetData.Members))
	for member, score := range zsetData.Members {
		zsetData.Scores = append(zsetData.Scores, ZSetMember{Member: member, Score: score})
	}
	sort.Slice(zsetData.Scores, func(i, j int) bool {
		if zsetData.Scores[i].Score == zsetData.Scores[j].Score {
			return zsetData.Scores[i].Member < zsetData.Scores[j].Member
		}
		return zsetData.Scores[i].Score < zsetData.Scores[j].Score
	})

	z.db.data[key] = &Value{
		Type:      TypeZSet,
		Data:      zsetData,
		ExpiresAt: nil,
	}
	delete(z.db.expiry, key)

	return added, nil
}

func (z *ZSetStorageImpl) ZRem(key string, members []string) (int64, error) {
	z.db.mu.Lock()
	defer z.db.mu.Unlock()

	value, exists := z.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeZSet {
		return 0, nil
	}

	zsetData, ok := value.Data.(*ZSetData)
	if !ok {
		return 0, nil
	}

	removed := int64(0)
	for _, member := range members {
		if _, exists := zsetData.Members[member]; exists {
			delete(zsetData.Members, member)
			removed++
		}
	}

	if len(zsetData.Members) == 0 {
		delete(z.db.data, key)
		delete(z.db.expiry, key)
		return removed, nil
	}

	// Rebuild sorted scores
	zsetData.Scores = make([]ZSetMember, 0, len(zsetData.Members))
	for member, score := range zsetData.Members {
		zsetData.Scores = append(zsetData.Scores, ZSetMember{Member: member, Score: score})
	}
	sort.Slice(zsetData.Scores, func(i, j int) bool {
		if zsetData.Scores[i].Score == zsetData.Scores[j].Score {
			return zsetData.Scores[i].Member < zsetData.Scores[j].Member
		}
		return zsetData.Scores[i].Score < zsetData.Scores[j].Score
	})

	return removed, nil
}

func (z *ZSetStorageImpl) ZScore(key, member string) (float64, bool) {
	z.db.mu.RLock()
	defer z.db.mu.RUnlock()

	value, exists := z.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeZSet {
		return 0, false
	}

	if zsetData, ok := value.Data.(*ZSetData); ok {
		score, exists := zsetData.Members[member]
		return score, exists
	}

	return 0, false
}

func (z *ZSetStorageImpl) ZRank(key, member string) (int64, bool) {
	z.db.mu.RLock()
	defer z.db.mu.RUnlock()

	value, exists := z.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeZSet {
		return 0, false
	}

	if zsetData, ok := value.Data.(*ZSetData); ok {
		for i, m := range zsetData.Scores {
			if m.Member == member {
				return int64(i), true
			}
		}
	}

	return 0, false
}

func (z *ZSetStorageImpl) ZRevRank(key, member string) (int64, bool) {
	z.db.mu.RLock()
	defer z.db.mu.RUnlock()

	value, exists := z.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeZSet {
		return 0, false
	}

	if zsetData, ok := value.Data.(*ZSetData); ok {
		for i, m := range zsetData.Scores {
			if m.Member == member {
				return int64(len(zsetData.Scores) - 1 - i), true
			}
		}
	}

	return 0, false
}

func (z *ZSetStorageImpl) ZRange(key string, start, stop int64, withScores bool) []interface{} {
	z.db.mu.RLock()
	defer z.db.mu.RUnlock()

	value, exists := z.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeZSet {
		return []interface{}{}
	}

	zsetData, ok := value.Data.(*ZSetData)
	if !ok {
		return []interface{}{}
	}

	length := int64(len(zsetData.Scores))
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
		return []interface{}{}
	}

	var result []interface{}
	for i := start; i <= stop; i++ {
		member := zsetData.Scores[i]
		result = append(result, member.Member)
		if withScores {
			result = append(result, member.Score)
		}
	}

	return result
}

func (z *ZSetStorageImpl) ZRevRange(key string, start, stop int64, withScores bool) []interface{} {
	z.db.mu.RLock()
	defer z.db.mu.RUnlock()

	value, exists := z.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeZSet {
		return []interface{}{}
	}

	zsetData, ok := value.Data.(*ZSetData)
	if !ok {
		return []interface{}{}
	}

	length := int64(len(zsetData.Scores))
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
		return []interface{}{}
	}

	var result []interface{}
	for i := start; i <= stop; i++ {
		member := zsetData.Scores[length-1-i]
		result = append(result, member.Member)
		if withScores {
			result = append(result, member.Score)
		}
	}

	return result
}

func (z *ZSetStorageImpl) ZRangeByScore(key string, min, max float64, withScores bool) []interface{} {
	z.db.mu.RLock()
	defer z.db.mu.RUnlock()

	value, exists := z.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeZSet {
		return []interface{}{}
	}

	zsetData, ok := value.Data.(*ZSetData)
	if !ok {
		return []interface{}{}
	}

	var result []interface{}
	for _, member := range zsetData.Scores {
		if member.Score >= min && member.Score <= max {
			result = append(result, member.Member)
			if withScores {
				result = append(result, member.Score)
			}
		}
	}

	return result
}

func (z *ZSetStorageImpl) ZRevRangeByScore(key string, max, min float64, withScores bool) []interface{} {
	z.db.mu.RLock()
	defer z.db.mu.RUnlock()

	value, exists := z.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeZSet {
		return []interface{}{}
	}

	zsetData, ok := value.Data.(*ZSetData)
	if !ok {
		return []interface{}{}
	}

	var result []interface{}
	for i := len(zsetData.Scores) - 1; i >= 0; i-- {
		member := zsetData.Scores[i]
		if member.Score >= min && member.Score <= max {
			result = append(result, member.Member)
			if withScores {
				result = append(result, member.Score)
			}
		}
	}

	return result
}

func (z *ZSetStorageImpl) ZCount(key string, min, max float64) int64 {
	z.db.mu.RLock()
	defer z.db.mu.RUnlock()

	value, exists := z.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeZSet {
		return 0
	}

	zsetData, ok := value.Data.(*ZSetData)
	if !ok {
		return 0
	}

	count := int64(0)
	for _, member := range zsetData.Scores {
		if member.Score >= min && member.Score <= max {
			count++
		}
	}

	return count
}

func (z *ZSetStorageImpl) ZCard(key string) int64 {
	z.db.mu.RLock()
	defer z.db.mu.RUnlock()

	value, exists := z.db.data[key]
	if !exists || value.IsExpired() || value.Type != TypeZSet {
		return 0
	}

	if zsetData, ok := value.Data.(*ZSetData); ok {
		return int64(len(zsetData.Members))
	}

	return 0
}
