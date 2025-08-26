package storage

import "time"

// StringStorage interface for string operations
type StringStorage interface {
	Set(key, value string, expiration *time.Time) error
	Get(key string) (string, bool)
	MSet(pairs map[string]string) error
	MGet(keys []string) map[string]string
	Exists(key string) bool
	Del(key string) bool
	Incr(key string) (int64, error)
	Decr(key string) (int64, error)
	IncrBy(key string, increment int64) (int64, error)
	DecrBy(key string, decrement int64) (int64, error)
	Append(key, value string) (int64, error)
	StrLen(key string) int64
}

// HashStorage interface for hash operations
type HashStorage interface {
	HSet(key, field, value string) (bool, error)
	HGet(key, field string) (string, bool)
	HMSet(key string, fields map[string]string) error
	HMGet(key string, fields []string) map[string]string
	HGetAll(key string) (map[string]string, error)
	HExists(key, field string) bool
	HDel(key string, fields []string) int64
	HLen(key string) int64
	HKeys(key string) []string
	HVals(key string) []string
}

// ListStorage interface for list operations
type ListStorage interface {
	LPush(key string, values []string) (int64, error)
	RPush(key string, values []string) (int64, error)
	LPop(key string) (string, bool)
	RPop(key string) (string, bool)
	LLen(key string) int64
	LIndex(key string, index int64) (string, bool)
	LSet(key string, index int64, value string) error
	LRange(key string, start, stop int64) []string
	LTrim(key string, start, stop int64) error
	LRem(key string, count int64, value string) int64
}

// SetStorage interface for set operations
type SetStorage interface {
	SAdd(key string, members []string) (int64, error)
	SRem(key string, members []string) (int64, error)
	SIsMember(key, member string) bool
	SMembers(key string) []string
	SCard(key string) int64
	SPop(key string, count int64) []string
	SRandMember(key string, count int64) []string
}

// ZSetStorage interface for sorted set operations
type ZSetStorage interface {
	ZAdd(key string, members map[string]float64) (int64, error)
	ZRem(key string, members []string) (int64, error)
	ZScore(key, member string) (float64, bool)
	ZRank(key, member string) (int64, bool)
	ZRevRank(key, member string) (int64, bool)
	ZRange(key string, start, stop int64, withScores bool) []interface{}
	ZRevRange(key string, start, stop int64, withScores bool) []interface{}
	ZRangeByScore(key string, min, max float64, withScores bool) []interface{}
	ZRevRangeByScore(key string, max, min float64, withScores bool) []interface{}
	ZCount(key string, min, max float64) int64
	ZCard(key string) int64
}

// CommonStorage interface for common operations across all data types
type CommonStorage interface {
	Exists(key string) bool
	Del(keys []string) int64
	Type(key string) ValueType
	TTL(key string) time.Duration
	Expire(key string, expiration time.Duration) bool
	ExpireAt(key string, timestamp time.Time) bool
	Persist(key string) bool
	SwapDB(db1, db2 int) error
}
