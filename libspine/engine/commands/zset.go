package commands

import (
	"fmt"
	"sort"
	"strconv"

	"spine-go/libspine/engine"
	"spine-go/libspine/engine/storage"
)

// ZSetMember represents a member in a sorted set with its score
type ZSetMember struct {
	Member string
	Score  float64
}

// ZSetData represents the internal structure of a sorted set
type ZSetData struct {
	Members map[string]float64 // member -> score
	Scores  []ZSetMember       // sorted by score for range operations
}

// RegisterZSetCommands registers all sorted set-related commands
func RegisterZSetCommands(registry *engine.CommandRegistry) error {
	commands := []engine.CommandHandler{
		&ZAddCommand{},
		&ZRemCommand{},
		&ZScoreCommand{},
		&ZRankCommand{},
		&ZRevRankCommand{},
		&ZRangeCommand{},
		&ZRevRangeCommand{},
		&ZRangeByScoreCommand{},
		&ZRevRangeByScoreCommand{},
		&ZCountCommand{},
		&ZCardCommand{},
		&ZIncrByCommand{},
		&ZRemRangeByRankCommand{},
		&ZRemRangeByScoreCommand{},
		&ZInterCommand{},
		&ZInterStoreCommand{},
		&ZUnionCommand{},
		&ZUnionStoreCommand{},
		&ZScanCommand{},
		&ZPopMinCommand{},
		&ZPopMaxCommand{},
		&BZPopMinCommand{},
		&BZPopMaxCommand{},
		&ZRandMemberCommand{},
		&ZMScoreCommand{},
	}

	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.GetInfo().Name, err)
		}
	}

	return nil
}

// Helper function to maintain sorted order
func (zd *ZSetData) updateSortedList() {
	zd.Scores = make([]ZSetMember, 0, len(zd.Members))
	for member, score := range zd.Members {
		zd.Scores = append(zd.Scores, ZSetMember{Member: member, Score: score})
	}
	sort.Slice(zd.Scores, func(i, j int) bool {
		if zd.Scores[i].Score == zd.Scores[j].Score {
			return zd.Scores[i].Member < zd.Scores[j].Member // lexicographic order for same score
		}
		return zd.Scores[i].Score < zd.Scores[j].Score
	})
}

// ZAddCommand implements the ZADD command
type ZAddCommand struct{}

func (c *ZAddCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 3 || nargs%2 == 0 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'zadd' command")
	}

	// Read key
	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}
	key, ok := keyValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}

	// Read score-member pairs
	members := make(map[string]float64)
	for i := 1; i < nargs; i += 2 {
		scoreValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid score")
		}
		scoreStr, ok := scoreValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid score")
		}
		score, err := strconv.ParseFloat(scoreStr, 64)
		if err != nil {
			return ctx.RespWriter.WriteError("ERR value is not a valid float")
		}

		memberValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid member")
		}
		member, ok := memberValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid member")
		}

		members[member] = score
	}

	// Use zset storage from context
	zsetStorage := ctx.Database.ZSetStorage
	addedCount, err := zsetStorage.ZAdd(key, members)
	if err != nil {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return ctx.RespWriter.WriteInteger(int64(addedCount))
}

func (c *ZAddCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZADD",
		Summary:      "Add one or more members to a sorted set, or update its score if it already exists",
		Syntax:       "ZADD key score member [score member ...]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *ZAddCommand) ModifiesData() bool {
	return true
}

// ZRemCommand implements the ZREM command
type ZRemCommand struct{}

func (c *ZRemCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'zrem' command")
	}

	// Read key
	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}
	key, ok := keyValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}

	// Get database
	db := ctx.Database

	// Get sorted set
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteInteger(0)
	}

	if value.Type != storage.TypeZSet {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	zsetData, ok := value.Data.(*ZSetData)
	if !ok {
		return ctx.RespWriter.WriteInteger(0)
	}

	// Remove members
	removedCount := 0
	for i := 1; i < nargs; i++ {
		memberValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid member")
		}
		member, ok := memberValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid member")
		}

		if _, exists := zsetData.Members[member]; exists {
			delete(zsetData.Members, member)
			removedCount++
		}
	}

	// Update or delete the sorted set
	if len(zsetData.Members) == 0 {
		db.Del(key)
	} else {
		zsetData.updateSortedList()
		db.SetValue(key, value)
	}

	return ctx.RespWriter.WriteInteger(int64(removedCount))
}

func (c *ZRemCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZREM",
		Summary:      "Remove one or more members from a sorted set",
		Syntax:       "ZREM key member [member ...]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *ZRemCommand) ModifiesData() bool {
	return true
}

// ZScoreCommand implements the ZSCORE command
type ZScoreCommand struct{}

func (c *ZScoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'zscore' command")
	}

	// Read key
	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}
	key, ok := keyValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}

	// Read member
	memberValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid member")
	}
	member, ok := memberValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid member")
	}

	// Get database
	db := ctx.Database

	// Get sorted set
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteNull()
	}

	if value.Type != storage.TypeZSet {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	zsetData, ok := value.Data.(*ZSetData)
	if !ok {
		return ctx.RespWriter.WriteNull()
	}

	if score, exists := zsetData.Members[member]; exists {
		return ctx.RespWriter.WriteBulkString(strconv.FormatFloat(score, 'f', -1, 64))
	}

	return ctx.RespWriter.WriteNull()
}

func (c *ZScoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZSCORE",
		Summary:      "Get the score associated with the given member in a sorted set",
		Syntax:       "ZSCORE key member",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *ZScoreCommand) ModifiesData() bool {
	return false
}

// ZCardCommand implements the ZCARD command
type ZCardCommand struct{}

func (c *ZCardCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'zcard' command")
	}

	// Read key
	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}
	key, ok := keyValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}

	// Get database
	db := ctx.Database

	// Get sorted set
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteInteger(0)
	}

	if value.Type != storage.TypeZSet {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	zsetData, ok := value.Data.(*ZSetData)
	if !ok {
		return ctx.RespWriter.WriteInteger(0)
	}

	return ctx.RespWriter.WriteInteger(int64(len(zsetData.Members)))
}

func (c *ZCardCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZCARD",
		Summary:      "Get the number of members in a sorted set",
		Syntax:       "ZCARD key",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *ZCardCommand) ModifiesData() bool {
	return false
}

// ZRangeCommand implements the ZRANGE command
type ZRangeCommand struct{}

func (c *ZRangeCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 3 || nargs > 4 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'zrange' command")
	}

	// Read key
	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}
	key, ok := keyValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}

	// Read start
	startValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid start")
	}
	startStr, ok := startValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid start")
	}
	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return ctx.RespWriter.WriteError("ERR value is not an integer or out of range")
	}

	// Read stop
	stopValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid stop")
	}
	stopStr, ok := stopValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid stop")
	}
	stop, err := strconv.ParseInt(stopStr, 10, 64)
	if err != nil {
		return ctx.RespWriter.WriteError("ERR value is not an integer or out of range")
	}

	// Check for WITHSCORES option
	withScores := false
	if nargs == 4 {
		optionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid option")
		}
		option, ok := optionValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid option")
		}
		if option == "WITHSCORES" {
			withScores = true
		} else {
			return ctx.RespWriter.WriteError("ERR syntax error")
		}
	}

	// Get database
	db := ctx.Database

	// Get sorted set
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	if value.Type != storage.TypeZSet {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	zsetData, ok := value.Data.(*ZSetData)
	if !ok {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	length := int64(len(zsetData.Scores))

	// Handle negative indices
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// Normalize bounds
	if start < 0 {
		start = 0
	}
	if start >= length {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}
	if stop >= length {
		stop = length - 1
	}
	if stop < start {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Extract range
	var result []interface{}
	for i := start; i <= stop; i++ {
		result = append(result, zsetData.Scores[i].Member)
		if withScores {
			result = append(result, strconv.FormatFloat(zsetData.Scores[i].Score, 'f', -1, 64))
		}
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *ZRangeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZRANGE",
		Summary:      "Return a range of members in a sorted set, by index",
		Syntax:       "ZRANGE key start stop [WITHSCORES]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      4,
		ModifiesData: false,
	}
}

func (c *ZRangeCommand) ModifiesData() bool {
	return false
}

// Placeholder implementations for remaining sorted set commands
type ZRankCommand struct{}
func (c *ZRankCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZRankCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZRANK", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZRankCommand) ModifiesData() bool { return false }

type ZRevRankCommand struct{}
func (c *ZRevRankCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZRevRankCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZREVRANK", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZRevRankCommand) ModifiesData() bool { return false }

type ZRevRangeCommand struct{}
func (c *ZRevRangeCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZRevRangeCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZREVRANGE", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZRevRangeCommand) ModifiesData() bool { return false }

type ZRangeByScoreCommand struct{}
func (c *ZRangeByScoreCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZRangeByScoreCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZRANGEBYSCORE", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZRangeByScoreCommand) ModifiesData() bool { return false }

type ZRevRangeByScoreCommand struct{}
func (c *ZRevRangeByScoreCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZRevRangeByScoreCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZREVRANGEBYSCORE", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZRevRangeByScoreCommand) ModifiesData() bool { return false }

type ZCountCommand struct{}
func (c *ZCountCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZCountCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZCOUNT", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZCountCommand) ModifiesData() bool { return false }

type ZIncrByCommand struct{}
func (c *ZIncrByCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZIncrByCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZINCRBY", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZIncrByCommand) ModifiesData() bool { return true }

type ZRemRangeByRankCommand struct{}
func (c *ZRemRangeByRankCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZRemRangeByRankCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZREMRANGEBYRANK", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZRemRangeByRankCommand) ModifiesData() bool { return true }

type ZRemRangeByScoreCommand struct{}
func (c *ZRemRangeByScoreCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZRemRangeByScoreCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZREMRANGEBYSCORE", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZRemRangeByScoreCommand) ModifiesData() bool { return true }

type ZInterCommand struct{}
func (c *ZInterCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZInterCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZINTER", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZInterCommand) ModifiesData() bool { return false }

type ZInterStoreCommand struct{}
func (c *ZInterStoreCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZInterStoreCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZINTERSTORE", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZInterStoreCommand) ModifiesData() bool { return true }

type ZUnionCommand struct{}
func (c *ZUnionCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZUnionCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZUNION", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZUnionCommand) ModifiesData() bool { return false }

type ZUnionStoreCommand struct{}
func (c *ZUnionStoreCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZUnionStoreCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZUNIONSTORE", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZUnionStoreCommand) ModifiesData() bool { return true }

type ZScanCommand struct{}
func (c *ZScanCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZScanCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZSCAN", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZScanCommand) ModifiesData() bool { return false }

type ZPopMinCommand struct{}
func (c *ZPopMinCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZPopMinCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZPOPMIN", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZPopMinCommand) ModifiesData() bool { return true }

type ZPopMaxCommand struct{}
func (c *ZPopMaxCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZPopMaxCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZPOPMAX", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZPopMaxCommand) ModifiesData() bool { return true }

type BZPopMinCommand struct{}
func (c *BZPopMinCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *BZPopMinCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "BZPOPMIN", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *BZPopMinCommand) ModifiesData() bool { return true }

type BZPopMaxCommand struct{}
func (c *BZPopMaxCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *BZPopMaxCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "BZPOPMAX", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *BZPopMaxCommand) ModifiesData() bool { return true }

type ZRandMemberCommand struct{}
func (c *ZRandMemberCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZRandMemberCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZRANDMEMBER", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZRandMemberCommand) ModifiesData() bool { return false }

type ZMScoreCommand struct{}
func (c *ZMScoreCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *ZMScoreCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "ZMSCORE", Categories: []engine.CommandCategory{engine.CategoryZSet}} }
func (c *ZMScoreCommand) ModifiesData() bool { return false }
