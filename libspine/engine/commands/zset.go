package commands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

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

func (c *ZRankCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 2 {
		return fmt.Errorf("wrong number of arguments for 'zrank' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	memberValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	member, ok := memberValue.AsString()
	if !ok {
		return fmt.Errorf("invalid member")
	}

	zsetStorage := ctx.Database.ZSetStorage
	rank, exists := zsetStorage.ZRank(key, member)
	if !exists {
		return ctx.RespWriter.WriteNull()
	}

	return ctx.RespWriter.WriteInteger(rank)
}

func (c *ZRankCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZRANK",
		Summary:      "Determine the index of a member in a sorted set",
		Syntax:       "ZRANK key member",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *ZRankCommand) ModifiesData() bool { return false }

type ZRevRankCommand struct{}

func (c *ZRevRankCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 2 {
		return fmt.Errorf("wrong number of arguments for 'zrevrank' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	memberValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	member, ok := memberValue.AsString()
	if !ok {
		return fmt.Errorf("invalid member")
	}

	zsetStorage := ctx.Database.ZSetStorage
	rank, exists := zsetStorage.ZRevRank(key, member)
	if !exists {
		return ctx.RespWriter.WriteNull()
	}

	return ctx.RespWriter.WriteInteger(rank)
}

func (c *ZRevRankCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZREVRANK",
		Summary:      "Determine the index of a member in a sorted set, with scores ordered from high to low",
		Syntax:       "ZREVRANK key member",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *ZRevRankCommand) ModifiesData() bool { return false }

type ZRevRangeCommand struct{}

func (c *ZRevRangeCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 || nargs > 4 {
		return fmt.Errorf("wrong number of arguments for 'zrevrange' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	startValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	startStr, ok := startValue.AsString()
	if !ok {
		return fmt.Errorf("invalid start index")
	}
	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid start index")
	}

	stopValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	stopStr, ok := stopValue.AsString()
	if !ok {
		return fmt.Errorf("invalid stop index")
	}
	stop, err := strconv.ParseInt(stopStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid stop index")
	}

	withScores := false
	if nargs == 4 {
		optionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		option, ok := optionValue.AsString()
		if !ok {
			return fmt.Errorf("invalid option")
		}
		if strings.ToUpper(option) == "WITHSCORES" {
			withScores = true
		} else {
			return fmt.Errorf("syntax error")
		}
	}

	zsetStorage := ctx.Database.ZSetStorage
	result := zsetStorage.ZRevRange(key, start, stop, withScores)

	return ctx.RespWriter.WriteArray(result)
}

func (c *ZRevRangeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZREVRANGE",
		Summary:      "Return a range of members in a sorted set, by index, with scores ordered from high to low",
		Syntax:       "ZREVRANGE key start stop [WITHSCORES]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      4,
		ModifiesData: false,
	}
}

func (c *ZRevRangeCommand) ModifiesData() bool { return false }

type ZRangeByScoreCommand struct{}

func (c *ZRangeByScoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 || nargs > 4 {
		return fmt.Errorf("wrong number of arguments for 'zrangebyscore' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	minValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	minStr, ok := minValue.AsString()
	if !ok {
		return fmt.Errorf("invalid min score")
	}
	min, err := strconv.ParseFloat(minStr, 64)
	if err != nil {
		return fmt.Errorf("invalid min score")
	}

	maxValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	maxStr, ok := maxValue.AsString()
	if !ok {
		return fmt.Errorf("invalid max score")
	}
	max, err := strconv.ParseFloat(maxStr, 64)
	if err != nil {
		return fmt.Errorf("invalid max score")
	}

	withScores := false
	if nargs == 4 {
		optionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		option, ok := optionValue.AsString()
		if !ok {
			return fmt.Errorf("invalid option")
		}
		if strings.ToUpper(option) == "WITHSCORES" {
			withScores = true
		} else {
			return fmt.Errorf("syntax error")
		}
	}

	zsetStorage := ctx.Database.ZSetStorage
	result := zsetStorage.ZRangeByScore(key, min, max, withScores)

	return ctx.RespWriter.WriteArray(result)
}

func (c *ZRangeByScoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZRANGEBYSCORE",
		Summary:      "Return a range of members in a sorted set, by score",
		Syntax:       "ZRANGEBYSCORE key min max [WITHSCORES]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      4,
		ModifiesData: false,
	}
}

func (c *ZRangeByScoreCommand) ModifiesData() bool { return false }

type ZRevRangeByScoreCommand struct{}

func (c *ZRevRangeByScoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 || nargs > 4 {
		return fmt.Errorf("wrong number of arguments for 'zrevrangebyscore' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	maxValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	maxStr, ok := maxValue.AsString()
	if !ok {
		return fmt.Errorf("invalid max score")
	}
	max, err := strconv.ParseFloat(maxStr, 64)
	if err != nil {
		return fmt.Errorf("invalid max score")
	}

	minValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	minStr, ok := minValue.AsString()
	if !ok {
		return fmt.Errorf("invalid min score")
	}
	min, err := strconv.ParseFloat(minStr, 64)
	if err != nil {
		return fmt.Errorf("invalid min score")
	}

	withScores := false
	if nargs == 4 {
		optionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		option, ok := optionValue.AsString()
		if !ok {
			return fmt.Errorf("invalid option")
		}
		if strings.ToUpper(option) == "WITHSCORES" {
			withScores = true
		} else {
			return fmt.Errorf("syntax error")
		}
	}

	zsetStorage := ctx.Database.ZSetStorage
	result := zsetStorage.ZRevRangeByScore(key, max, min, withScores)

	return ctx.RespWriter.WriteArray(result)
}

func (c *ZRevRangeByScoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZREVRANGEBYSCORE",
		Summary:      "Return a range of members in a sorted set, by score, with scores ordered from high to low",
		Syntax:       "ZREVRANGEBYSCORE key max min [WITHSCORES]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      4,
		ModifiesData: false,
	}
}

func (c *ZRevRangeByScoreCommand) ModifiesData() bool { return false }

type ZCountCommand struct{}

func (c *ZCountCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'zcount' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	minValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	minStr, ok := minValue.AsString()
	if !ok {
		return fmt.Errorf("invalid min score")
	}
	min, err := strconv.ParseFloat(minStr, 64)
	if err != nil {
		return fmt.Errorf("invalid min score")
	}

	maxValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	maxStr, ok := maxValue.AsString()
	if !ok {
		return fmt.Errorf("invalid max score")
	}
	max, err := strconv.ParseFloat(maxStr, 64)
	if err != nil {
		return fmt.Errorf("invalid max score")
	}

	zsetStorage := ctx.Database.ZSetStorage
	count := zsetStorage.ZCount(key, min, max)

	return ctx.RespWriter.WriteInteger(count)
}

func (c *ZCountCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZCOUNT",
		Summary:      "Count the members in a sorted set with scores within the given range",
		Syntax:       "ZCOUNT key min max",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: false,
	}
}

func (c *ZCountCommand) ModifiesData() bool { return false }

type ZIncrByCommand struct{}

func (c *ZIncrByCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'zincrby' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	incrementValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	incrementStr, ok := incrementValue.AsString()
	if !ok {
		return fmt.Errorf("invalid increment")
	}
	increment, err := strconv.ParseFloat(incrementStr, 64)
	if err != nil {
		return fmt.Errorf("invalid increment")
	}

	memberValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	member, ok := memberValue.AsString()
	if !ok {
		return fmt.Errorf("invalid member")
	}

	zsetStorage := ctx.Database.ZSetStorage
	// Get current score or 0 if member doesn't exist
	currentScore, exists := zsetStorage.ZScore(key, member)
	if !exists {
		currentScore = 0
	}

	newScore := currentScore + increment
	// Add/update the member with new score
	members := map[string]float64{member: newScore}
	_, err = zsetStorage.ZAdd(key, members)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteBulkString(fmt.Sprintf("%.17g", newScore))
}

func (c *ZIncrByCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZINCRBY",
		Summary:      "Increment the score of a member in a sorted set",
		Syntax:       "ZINCRBY key increment member",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *ZIncrByCommand) ModifiesData() bool { return true }

type ZRemRangeByRankCommand struct{}

func (c *ZRemRangeByRankCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'zremrangebyrank' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	startValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	startStr, ok := startValue.AsString()
	if !ok {
		return fmt.Errorf("invalid start index")
	}
	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid start index")
	}

	stopValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	stopStr, ok := stopValue.AsString()
	if !ok {
		return fmt.Errorf("invalid stop index")
	}
	stop, err := strconv.ParseInt(stopStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid stop index")
	}

	zsetStorage := ctx.Database.ZSetStorage
	// Get members in range to remove
	membersToRemove := zsetStorage.ZRange(key, start, stop, false)
	if len(membersToRemove) == 0 {
		return ctx.RespWriter.WriteInteger(0)
	}

	// Convert to string slice
	memberNames := make([]string, len(membersToRemove))
	for i, member := range membersToRemove {
		memberNames[i] = member.(string)
	}

	// Remove the members
	removed, err := zsetStorage.ZRem(key, memberNames)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteInteger(removed)
}

func (c *ZRemRangeByRankCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZREMRANGEBYRANK",
		Summary:      "Remove all members in a sorted set within the given indexes",
		Syntax:       "ZREMRANGEBYRANK key start stop",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *ZRemRangeByRankCommand) ModifiesData() bool { return true }

type ZRemRangeByScoreCommand struct{}

func (c *ZRemRangeByScoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'zremrangebyscore' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	minValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	minStr, ok := minValue.AsString()
	if !ok {
		return fmt.Errorf("invalid min score")
	}
	min, err := strconv.ParseFloat(minStr, 64)
	if err != nil {
		return fmt.Errorf("invalid min score")
	}

	maxValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	maxStr, ok := maxValue.AsString()
	if !ok {
		return fmt.Errorf("invalid max score")
	}
	max, err := strconv.ParseFloat(maxStr, 64)
	if err != nil {
		return fmt.Errorf("invalid max score")
	}

	zsetStorage := ctx.Database.ZSetStorage
	// Get members in score range to remove
	membersToRemove := zsetStorage.ZRangeByScore(key, min, max, false)
	if len(membersToRemove) == 0 {
		return ctx.RespWriter.WriteInteger(0)
	}

	// Convert to string slice
	memberNames := make([]string, len(membersToRemove))
	for i, member := range membersToRemove {
		memberNames[i] = member.(string)
	}

	// Remove the members
	removed, err := zsetStorage.ZRem(key, memberNames)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteInteger(removed)
}

func (c *ZRemRangeByScoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZREMRANGEBYSCORE",
		Summary:      "Remove all members in a sorted set within the given scores",
		Syntax:       "ZREMRANGEBYSCORE key min max",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *ZRemRangeByScoreCommand) ModifiesData() bool { return true }

type ZInterCommand struct{}

func (c *ZInterCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'zinter' command")
	}

	numkeysValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	numkeysStr, ok := numkeysValue.AsString()
	if !ok {
		return fmt.Errorf("invalid numkeys")
	}
	numkeys, err := strconv.ParseInt(numkeysStr, 10, 64)
	if err != nil || numkeys <= 0 {
		return fmt.Errorf("invalid numkeys")
	}

	if int64(nargs) < 1+numkeys {
		return fmt.Errorf("wrong number of arguments for 'zinter' command")
	}

	keys := make([]string, numkeys)
	for i := int64(0); i < numkeys; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i] = key
	}

	// Parse optional WITHSCORES
	withScores := false
	if int64(nargs) > 1+numkeys {
		optionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		option, ok := optionValue.AsString()
		if !ok {
			return fmt.Errorf("invalid option")
		}
		if strings.ToUpper(option) == "WITHSCORES" {
			withScores = true
		} else {
			return fmt.Errorf("syntax error")
		}
	}

	zsetStorage := ctx.Database.ZSetStorage
	// Simplified intersection: get all members from first set and check if they exist in all other sets
	if len(keys) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Get all members from first set
	firstSetMembers := zsetStorage.ZRange(keys[0], 0, -1, true)
	if len(firstSetMembers) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Build intersection
	result := make([]interface{}, 0)
	for i := 0; i < len(firstSetMembers); i += 2 {
		member := firstSetMembers[i].(string)
		score := firstSetMembers[i+1]
		
		// Check if member exists in all other sets
		existsInAll := true
		totalScore := 0.0
		if scoreFloat, ok := score.(float64); ok {
			totalScore = scoreFloat
		} else if scoreStr, ok := score.(string); ok {
			if parsed, err := strconv.ParseFloat(scoreStr, 64); err == nil {
				totalScore = parsed
			}
		}
		
		for j := 1; j < len(keys); j++ {
			memberScore, exists := zsetStorage.ZScore(keys[j], member)
			if !exists {
				existsInAll = false
				break
			}
			totalScore += memberScore
		}
		
		if existsInAll {
			result = append(result, member)
			if withScores {
				result = append(result, fmt.Sprintf("%.17g", totalScore))
			}
		}
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *ZInterCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZINTER",
		Summary:      "Intersect multiple sorted sets",
		Syntax:       "ZINTER numkeys key [key ...] [WITHSCORES]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *ZInterCommand) ModifiesData() bool { return false }

type ZInterStoreCommand struct{}

func (c *ZInterStoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 {
		return fmt.Errorf("wrong number of arguments for 'zinterstore' command")
	}

	destValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dest, ok := destValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	numkeysValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	numkeysStr, ok := numkeysValue.AsString()
	if !ok {
		return fmt.Errorf("invalid numkeys")
	}
	numkeys, err := strconv.ParseInt(numkeysStr, 10, 64)
	if err != nil || numkeys <= 0 {
		return fmt.Errorf("invalid numkeys")
	}

	if int64(nargs) < 2+numkeys {
		return fmt.Errorf("wrong number of arguments for 'zinterstore' command")
	}

	keys := make([]string, numkeys)
	for i := int64(0); i < numkeys; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i] = key
	}

	zsetStorage := ctx.Database.ZSetStorage
	// Clear destination first
	allMembers := zsetStorage.ZRange(dest, 0, -1, false)
	if len(allMembers) > 0 {
		memberNames := make([]string, len(allMembers))
		for i, member := range allMembers {
			memberNames[i] = member.(string)
		}
		zsetStorage.ZRem(dest, memberNames)
	}

	// Perform intersection
	if len(keys) == 0 {
		return ctx.RespWriter.WriteInteger(0)
	}

	// Get all members from first set
	firstSetMembers := zsetStorage.ZRange(keys[0], 0, -1, true)
	if len(firstSetMembers) == 0 {
		return ctx.RespWriter.WriteInteger(0)
	}

	// Build intersection and store
	intersectionMembers := make(map[string]float64)
	for i := 0; i < len(firstSetMembers); i += 2 {
		member := firstSetMembers[i].(string)
		score := firstSetMembers[i+1]
		
		// Check if member exists in all other sets
		existsInAll := true
		totalScore := 0.0
		if scoreFloat, ok := score.(float64); ok {
			totalScore = scoreFloat
		} else if scoreStr, ok := score.(string); ok {
			if parsed, err := strconv.ParseFloat(scoreStr, 64); err == nil {
				totalScore = parsed
			}
		}
		
		for j := 1; j < len(keys); j++ {
			memberScore, exists := zsetStorage.ZScore(keys[j], member)
			if !exists {
				existsInAll = false
				break
			}
			totalScore += memberScore
		}
		
		if existsInAll {
			intersectionMembers[member] = totalScore
		}
	}

	// Store intersection result
	if len(intersectionMembers) > 0 {
		zsetStorage.ZAdd(dest, intersectionMembers)
	}

	return ctx.RespWriter.WriteInteger(int64(len(intersectionMembers)))
}

func (c *ZInterStoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZINTERSTORE",
		Summary:      "Intersect multiple sorted sets and store the resulting sorted set in a new key",
		Syntax:       "ZINTERSTORE destination numkeys key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *ZInterStoreCommand) ModifiesData() bool { return true }

type ZUnionCommand struct{}

func (c *ZUnionCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'zunion' command")
	}

	numkeysValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	numkeysStr, ok := numkeysValue.AsString()
	if !ok {
		return fmt.Errorf("invalid numkeys")
	}
	numkeys, err := strconv.ParseInt(numkeysStr, 10, 64)
	if err != nil || numkeys <= 0 {
		return fmt.Errorf("invalid numkeys")
	}

	if int64(nargs) < 1+numkeys {
		return fmt.Errorf("wrong number of arguments for 'zunion' command")
	}

	keys := make([]string, numkeys)
	for i := int64(0); i < numkeys; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i] = key
	}

	// Parse optional WITHSCORES
	withScores := false
	if int64(nargs) > 1+numkeys {
		optionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		option, ok := optionValue.AsString()
		if !ok {
			return fmt.Errorf("invalid option")
		}
		if strings.ToUpper(option) == "WITHSCORES" {
			withScores = true
		} else {
			return fmt.Errorf("syntax error")
		}
	}

	zsetStorage := ctx.Database.ZSetStorage
	// Build union of all sets
	unionMembers := make(map[string]float64)

	for _, key := range keys {
		members := zsetStorage.ZRange(key, 0, -1, true)
		for i := 0; i < len(members); i += 2 {
			member := members[i].(string)
			score := members[i+1]
			
			scoreFloat := 0.0
			if scoreVal, ok := score.(float64); ok {
				scoreFloat = scoreVal
			} else if scoreStr, ok := score.(string); ok {
				if parsed, err := strconv.ParseFloat(scoreStr, 64); err == nil {
					scoreFloat = parsed
				}
			}
			
			// Add scores if member already exists
			if existingScore, exists := unionMembers[member]; exists {
				unionMembers[member] = existingScore + scoreFloat
			} else {
				unionMembers[member] = scoreFloat
			}
		}
	}

	// Convert to sorted result
	type memberScore struct {
		member string
		score  float64
	}

	memberScores := make([]memberScore, 0, len(unionMembers))
	for member, score := range unionMembers {
		memberScores = append(memberScores, memberScore{member: member, score: score})
	}

	// Sort by score
	sort.Slice(memberScores, func(i, j int) bool {
		return memberScores[i].score < memberScores[j].score
	})

	// Build result array
	result := make([]interface{}, 0, len(memberScores)*2)
	for _, ms := range memberScores {
		result = append(result, ms.member)
		if withScores {
			result = append(result, fmt.Sprintf("%.17g", ms.score))
		}
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *ZUnionCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZUNION",
		Summary:      "Add multiple sorted sets",
		Syntax:       "ZUNION numkeys key [key ...] [WITHSCORES]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *ZUnionCommand) ModifiesData() bool { return false }

type ZUnionStoreCommand struct{}

func (c *ZUnionStoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 {
		return fmt.Errorf("wrong number of arguments for 'zunionstore' command")
	}

	destValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dest, ok := destValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	numkeysValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	numkeysStr, ok := numkeysValue.AsString()
	if !ok {
		return fmt.Errorf("invalid numkeys")
	}
	numkeys, err := strconv.ParseInt(numkeysStr, 10, 64)
	if err != nil || numkeys <= 0 {
		return fmt.Errorf("invalid numkeys")
	}

	if int64(nargs) < 2+numkeys {
		return fmt.Errorf("wrong number of arguments for 'zunionstore' command")
	}

	keys := make([]string, numkeys)
	for i := int64(0); i < numkeys; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i] = key
	}

	zsetStorage := ctx.Database.ZSetStorage
	// Clear destination first
	allMembers := zsetStorage.ZRange(dest, 0, -1, false)
	if len(allMembers) > 0 {
		memberNames := make([]string, len(allMembers))
		for i, member := range allMembers {
			memberNames[i] = member.(string)
		}
		zsetStorage.ZRem(dest, memberNames)
	}

	// Build union of all sets
	unionMembers := make(map[string]float64)

	for _, key := range keys {
		members := zsetStorage.ZRange(key, 0, -1, true)
		for i := 0; i < len(members); i += 2 {
			member := members[i].(string)
			score := members[i+1]
			
			scoreFloat := 0.0
			if scoreVal, ok := score.(float64); ok {
				scoreFloat = scoreVal
			} else if scoreStr, ok := score.(string); ok {
				if parsed, err := strconv.ParseFloat(scoreStr, 64); err == nil {
					scoreFloat = parsed
				}
			}
			
			// Add scores if member already exists
			if existingScore, exists := unionMembers[member]; exists {
				unionMembers[member] = existingScore + scoreFloat
			} else {
				unionMembers[member] = scoreFloat
			}
		}
	}

	// Store union result
	if len(unionMembers) > 0 {
		zsetStorage.ZAdd(dest, unionMembers)
	}

	return ctx.RespWriter.WriteInteger(int64(len(unionMembers)))
}

func (c *ZUnionStoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZUNIONSTORE",
		Summary:      "Add multiple sorted sets and store the resulting sorted set in a new key",
		Syntax:       "ZUNIONSTORE destination numkeys key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *ZUnionStoreCommand) ModifiesData() bool { return true }

type ZScanCommand struct{}

func (c *ZScanCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'zscan' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	cursorValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	cursorStr, ok := cursorValue.AsString()
	if !ok {
		return fmt.Errorf("invalid cursor")
	}
	cursor, err := strconv.ParseInt(cursorStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid cursor")
	}

	// Parse optional MATCH and COUNT parameters (simplified implementation)
	matchPattern := "*"
	count := int64(10)

	for i := 2; i < nargs; i += 2 {
		if i+1 >= nargs {
			break
		}
		optionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		option, ok := optionValue.AsString()
		if !ok {
			return fmt.Errorf("invalid option")
		}

		valueValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		value, ok := valueValue.AsString()
		if !ok {
			return fmt.Errorf("invalid option value")
		}

		switch strings.ToUpper(option) {
		case "MATCH":
			matchPattern = value
		case "COUNT":
			if parsedCount, err := strconv.ParseInt(value, 10, 64); err == nil {
				count = parsedCount
			}
		}
	}

	zsetStorage := ctx.Database.ZSetStorage
	// Get all members with scores
	allMembers := zsetStorage.ZRange(key, 0, -1, true)
	if len(allMembers) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{"0", []interface{}{}})
	}

	// Simple cursor-based pagination (simplified implementation)
	startIdx := cursor * 2 // Each member-score pair takes 2 slots
	if startIdx >= int64(len(allMembers)) {
		return ctx.RespWriter.WriteArray([]interface{}{"0", []interface{}{}})
	}

	endIdx := startIdx + (count * 2)
	if endIdx > int64(len(allMembers)) {
		endIdx = int64(len(allMembers))
	}

	// Build result with member-score pairs
	result := make([]interface{}, 0)
	for i := startIdx; i < endIdx; i += 2 {
		if i+1 < int64(len(allMembers)) {
			member := allMembers[i].(string)
			score := allMembers[i+1]
			
			// Simple pattern matching (only supports * wildcard)
			if matchPattern == "*" || strings.Contains(member, strings.Trim(matchPattern, "*")) {
				result = append(result, member)
				if scoreStr, ok := score.(string); ok {
					result = append(result, scoreStr)
				} else {
					result = append(result, fmt.Sprintf("%.17g", score))
				}
			}
		}
	}

	// Calculate next cursor
	nextCursor := "0"
	if endIdx < int64(len(allMembers)) {
		nextCursor = fmt.Sprintf("%d", endIdx/2)
	}

	return ctx.RespWriter.WriteArray([]interface{}{nextCursor, result})
}

func (c *ZScanCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZSCAN",
		Summary:      "Incrementally iterate sorted sets elements and associated scores",
		Syntax:       "ZSCAN key cursor [MATCH pattern] [COUNT count]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *ZScanCommand) ModifiesData() bool { return false }

type ZPopMinCommand struct{}

func (c *ZPopMinCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 || nargs > 2 {
		return fmt.Errorf("wrong number of arguments for 'zpopmin' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	count := int64(1)
	if nargs == 2 {
		countValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		countStr, ok := countValue.AsString()
		if !ok {
			return fmt.Errorf("invalid count")
		}
		count, err = strconv.ParseInt(countStr, 10, 64)
		if err != nil || count < 0 {
			return fmt.Errorf("invalid count")
		}
	}

	zsetStorage := ctx.Database.ZSetStorage
	if zsetStorage.ZCard(key) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Get minimum elements
	members := zsetStorage.ZRange(key, 0, count-1, true)
	if len(members) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Remove the popped members
	memberNames := make([]string, 0)
	for i := 0; i < len(members); i += 2 {
		memberNames = append(memberNames, members[i].(string))
	}
	zsetStorage.ZRem(key, memberNames)

	return ctx.RespWriter.WriteArray(members)
}

func (c *ZPopMinCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZPOPMIN",
		Summary:      "Remove and return members with the lowest scores in a sorted set",
		Syntax:       "ZPOPMIN key [count]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      1,
		MaxArgs:      2,
		ModifiesData: true,
	}
}

func (c *ZPopMinCommand) ModifiesData() bool { return true }

type ZPopMaxCommand struct{}

func (c *ZPopMaxCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 || nargs > 2 {
		return fmt.Errorf("wrong number of arguments for 'zpopmax' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	count := int64(1)
	if nargs == 2 {
		countValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		countStr, ok := countValue.AsString()
		if !ok {
			return fmt.Errorf("invalid count")
		}
		count, err = strconv.ParseInt(countStr, 10, 64)
		if err != nil || count < 0 {
			return fmt.Errorf("invalid count")
		}
	}

	zsetStorage := ctx.Database.ZSetStorage
	if zsetStorage.ZCard(key) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Get maximum elements (reverse range from end)
	members := zsetStorage.ZRevRange(key, 0, count-1, true)
	if len(members) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Remove the popped members
	memberNames := make([]string, 0)
	for i := 0; i < len(members); i += 2 {
		memberNames = append(memberNames, members[i].(string))
	}
	zsetStorage.ZRem(key, memberNames)

	return ctx.RespWriter.WriteArray(members)
}

func (c *ZPopMaxCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZPOPMAX",
		Summary:      "Remove and return members with the highest scores in a sorted set",
		Syntax:       "ZPOPMAX key [count]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      1,
		MaxArgs:      2,
		ModifiesData: true,
	}
}

func (c *ZPopMaxCommand) ModifiesData() bool { return true }

type BZPopMinCommand struct{}

func (c *BZPopMinCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'bzpopmin' command")
	}

	keys := make([]string, nargs-1)
	for i := 0; i < nargs-1; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i] = key
	}

	// Read timeout (ignored in this simplified implementation)
	_, err = ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}

	zsetStorage := ctx.Database.ZSetStorage
	// Try to pop from each key in order (non-blocking implementation)
	for _, key := range keys {
		if zsetStorage.ZCard(key) > 0 {
			// Get minimum element
			members := zsetStorage.ZRange(key, 0, 0, true)
			if len(members) >= 2 {
				member := members[0].(string)
				score := members[1]
				
				// Remove the popped member
				zsetStorage.ZRem(key, []string{member})
				
				// Return [key, member, score]
				result := []interface{}{key, member}
				if scoreStr, ok := score.(string); ok {
					result = append(result, scoreStr)
				} else {
					result = append(result, fmt.Sprintf("%.17g", score))
				}
				return ctx.RespWriter.WriteArray(result)
			}
		}
	}

	// No elements available (simplified: return null instead of blocking)
	return ctx.RespWriter.WriteNull()
}

func (c *BZPopMinCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "BZPOPMIN",
		Summary:      "Remove and get the member with the lowest score from one or more sorted sets, or block until one is available",
		Syntax:       "BZPOPMIN key [key ...] timeout",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *BZPopMinCommand) ModifiesData() bool { return true }

type BZPopMaxCommand struct{}

func (c *BZPopMaxCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'bzpopmax' command")
	}

	keys := make([]string, nargs-1)
	for i := 0; i < nargs-1; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i] = key
	}

	// Read timeout (ignored in this simplified implementation)
	_, err = ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}

	zsetStorage := ctx.Database.ZSetStorage
	// Try to pop from each key in order (non-blocking implementation)
	for _, key := range keys {
		if zsetStorage.ZCard(key) > 0 {
			// Get maximum element (reverse range from end)
			members := zsetStorage.ZRevRange(key, 0, 0, true)
			if len(members) >= 2 {
				member := members[0].(string)
				score := members[1]
				
				// Remove the popped member
				zsetStorage.ZRem(key, []string{member})
				
				// Return [key, member, score]
				result := []interface{}{key, member}
				if scoreStr, ok := score.(string); ok {
					result = append(result, scoreStr)
				} else {
					result = append(result, fmt.Sprintf("%.17g", score))
				}
				return ctx.RespWriter.WriteArray(result)
			}
		}
	}

	// No elements available (simplified: return null instead of blocking)
	return ctx.RespWriter.WriteNull()
}

func (c *BZPopMaxCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "BZPOPMAX",
		Summary:      "Remove and get the member with the highest score from one or more sorted sets, or block until one is available",
		Syntax:       "BZPOPMAX key [key ...] timeout",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *BZPopMaxCommand) ModifiesData() bool { return true }

type ZRandMemberCommand struct{}

func (c *ZRandMemberCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 || nargs > 3 {
		return fmt.Errorf("wrong number of arguments for 'zrandmember' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	count := int64(1)
	withScores := false

	if nargs >= 2 {
		countValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		countStr, ok := countValue.AsString()
		if !ok {
			return fmt.Errorf("invalid count")
		}
		count, err = strconv.ParseInt(countStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid count")
		}
	}

	if nargs == 3 {
		optionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		option, ok := optionValue.AsString()
		if !ok {
			return fmt.Errorf("invalid option")
		}
		if strings.ToUpper(option) == "WITHSCORES" {
			withScores = true
		} else {
			return fmt.Errorf("syntax error")
		}
	}

	zsetStorage := ctx.Database.ZSetStorage
	if zsetStorage.ZCard(key) == 0 {
		if count == 1 && !withScores {
			return ctx.RespWriter.WriteNull()
		}
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Get all members
	allMembers := zsetStorage.ZRange(key, 0, -1, true)
	if len(allMembers) == 0 {
		if count == 1 && !withScores {
			return ctx.RespWriter.WriteNull()
		}
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Simple random selection (using modulo - not cryptographically secure)
	memberCount := len(allMembers) / 2
	if count == 1 && !withScores {
		// Return single member
		randomIdx := (int64(len(key)) * 7) % int64(memberCount) // Simple pseudo-random
		member := allMembers[randomIdx*2].(string)
		return ctx.RespWriter.WriteBulkString(member)
	}

	// Return multiple members
	result := make([]interface{}, 0)
	absCount := count
	if absCount < 0 {
		absCount = -absCount
	}

	for i := int64(0); i < absCount && i < int64(memberCount); i++ {
		randomIdx := ((int64(len(key)) * 7) + i*13) % int64(memberCount) // Simple pseudo-random
		member := allMembers[randomIdx*2].(string)
		result = append(result, member)
		if withScores {
			score := allMembers[randomIdx*2+1]
			if scoreStr, ok := score.(string); ok {
				result = append(result, scoreStr)
			} else {
				result = append(result, fmt.Sprintf("%.17g", score))
			}
		}
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *ZRandMemberCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZRANDMEMBER",
		Summary:      "Get one or multiple random members from a sorted set",
		Syntax:       "ZRANDMEMBER key [count [WITHSCORES]]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      1,
		MaxArgs:      3,
		ModifiesData: false,
	}
}

func (c *ZRandMemberCommand) ModifiesData() bool { return false }

type ZMScoreCommand struct{}

func (c *ZMScoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'zmscore' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	members := make([]string, nargs-1)
	for i := 0; i < nargs-1; i++ {
		memberValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		member, ok := memberValue.AsString()
		if !ok {
			return fmt.Errorf("invalid member")
		}
		members[i] = member
	}

	zsetStorage := ctx.Database.ZSetStorage
	result := make([]interface{}, len(members))
	for i, member := range members {
		score, exists := zsetStorage.ZScore(key, member)
		if exists {
			result[i] = fmt.Sprintf("%.17g", score)
		} else {
			result[i] = nil
		}
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *ZMScoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZMSCORE",
		Summary:      "Get the scores associated with the specified members in a sorted set",
		Syntax:       "ZMSCORE key member [member ...]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *ZMScoreCommand) ModifiesData() bool { return false }

type ZDiffCommand struct{}

func (c *ZDiffCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'zdiff' command")
	}

	numkeysValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	numkeysStr, ok := numkeysValue.AsString()
	if !ok {
		return fmt.Errorf("invalid numkeys")
	}
	numkeys, err := strconv.Atoi(numkeysStr)
	if err != nil || numkeys < 1 {
		return fmt.Errorf("invalid numkeys")
	}

	if nargs < numkeys+1 {
		return fmt.Errorf("wrong number of arguments for 'zdiff' command")
	}

	keys := make([]string, numkeys)
	for i := 0; i < numkeys; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i] = key
	}

	withScores := false
	if nargs > numkeys+1 {
		optionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		option, ok := optionValue.AsString()
		if ok && strings.ToUpper(option) == "WITHSCORES" {
			withScores = true
		}
	}

	zsetStorage := ctx.Database.ZSetStorage
	if len(keys) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Get all members from first set
	firstSetMembers := zsetStorage.ZRange(keys[0], 0, -1, true)
	if len(firstSetMembers) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	result := make([]interface{}, 0)
	for i := 0; i < len(firstSetMembers); i += 2 {
		member := firstSetMembers[i].(string)
		score := firstSetMembers[i+1]

		// Check if member exists in any other set
		existsInOther := false
		for j := 1; j < len(keys); j++ {
			if _, exists := zsetStorage.ZScore(keys[j], member); exists {
				existsInOther = true
				break
			}
		}

		// If member doesn't exist in other sets, include in result
		if !existsInOther {
			result = append(result, member)
			if withScores {
				if scoreStr, ok := score.(string); ok {
					result = append(result, scoreStr)
				} else {
					result = append(result, fmt.Sprintf("%.17g", score))
				}
			}
		}
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *ZDiffCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZDIFF",
		Summary:      "Subtract multiple sorted sets",
		Syntax:       "ZDIFF numkeys key [key ...] [WITHSCORES]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *ZDiffCommand) ModifiesData() bool { return false }

type ZDiffStoreCommand struct{}

func (c *ZDiffStoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 {
		return fmt.Errorf("wrong number of arguments for 'zdiffstore' command")
	}

	destValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dest, ok := destValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	numkeysValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	numkeysStr, ok := numkeysValue.AsString()
	if !ok {
		return fmt.Errorf("invalid numkeys")
	}
	numkeys, err := strconv.Atoi(numkeysStr)
	if err != nil || numkeys < 1 {
		return fmt.Errorf("invalid numkeys")
	}

	if nargs < numkeys+2 {
		return fmt.Errorf("wrong number of arguments for 'zdiffstore' command")
	}

	keys := make([]string, numkeys)
	for i := 0; i < numkeys; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i] = key
	}

	zsetStorage := ctx.Database.ZSetStorage
	
	// Clear destination key
	allMembers := zsetStorage.ZRange(dest, 0, -1, false)
	if len(allMembers) > 0 {
		memberStrings := make([]string, 0, len(allMembers))
		for _, member := range allMembers {
			if memberStr, ok := member.(string); ok {
				memberStrings = append(memberStrings, memberStr)
			}
		}
		if len(memberStrings) > 0 {
			zsetStorage.ZRem(dest, memberStrings)
		}
	}

	if len(keys) == 0 {
		return ctx.RespWriter.WriteInteger(0)
	}

	// Get all members from first set
	firstSetMembers := zsetStorage.ZRange(keys[0], 0, -1, true)
	if len(firstSetMembers) == 0 {
		return ctx.RespWriter.WriteInteger(0)
	}

	count := int64(0)
	for i := 0; i < len(firstSetMembers); i += 2 {
		member := firstSetMembers[i].(string)
		score := firstSetMembers[i+1]

		// Check if member exists in any other set
		existsInOther := false
		for j := 1; j < len(keys); j++ {
			if _, exists := zsetStorage.ZScore(keys[j], member); exists {
				existsInOther = true
				break
			}
		}

		// If member doesn't exist in other sets, add to destination
		if !existsInOther {
			var scoreFloat float64
			if scoreStr, ok := score.(string); ok {
				if parsed, err := strconv.ParseFloat(scoreStr, 64); err == nil {
					scoreFloat = parsed
				}
			} else if scoreVal, ok := score.(float64); ok {
				scoreFloat = scoreVal
			}
			
			memberMap := map[string]float64{member: scoreFloat}
			zsetStorage.ZAdd(dest, memberMap)
			count++
		}
	}

	return ctx.RespWriter.WriteInteger(count)
}

func (c *ZDiffStoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZDIFFSTORE",
		Summary:      "Subtract multiple sorted sets and store the resulting sorted set in a new key",
		Syntax:       "ZDIFFSTORE destination numkeys key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *ZDiffStoreCommand) ModifiesData() bool { return true }

type ZMPopCommand struct{}

func (c *ZMPopCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 {
		return fmt.Errorf("wrong number of arguments for 'zmpop' command")
	}

	numkeysValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	numkeysStr, ok := numkeysValue.AsString()
	if !ok {
		return fmt.Errorf("invalid numkeys")
	}
	numkeys, err := strconv.Atoi(numkeysStr)
	if err != nil || numkeys < 1 {
		return fmt.Errorf("invalid numkeys")
	}

	if nargs < numkeys+2 {
		return fmt.Errorf("wrong number of arguments for 'zmpop' command")
	}

	keys := make([]string, numkeys)
	for i := 0; i < numkeys; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i] = key
	}

	directionValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	direction, ok := directionValue.AsString()
	if !ok {
		return fmt.Errorf("invalid direction")
	}
	direction = strings.ToUpper(direction)
	if direction != "MIN" && direction != "MAX" {
		return fmt.Errorf("syntax error")
	}

	count := int64(1)
	if nargs > numkeys+2 {
		countOptionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		countOption, ok := countOptionValue.AsString()
		if ok && strings.ToUpper(countOption) == "COUNT" {
			if nargs > numkeys+3 {
				countValue, err := ctx.ReqReader.NextValue()
				if err != nil {
					return err
				}
				countStr, ok := countValue.AsString()
				if !ok {
					return fmt.Errorf("invalid count")
				}
				if parsed, err := strconv.ParseInt(countStr, 10, 64); err == nil && parsed > 0 {
					count = parsed
				}
			}
		}
	}

	zsetStorage := ctx.Database.ZSetStorage
	
	// Try to pop from each key in order
	for _, key := range keys {
		if zsetStorage.ZCard(key) == 0 {
			continue
		}

		elements := make([]interface{}, 0)
		for i := int64(0); i < count; i++ {
			var members []interface{}
			if direction == "MIN" {
				members = zsetStorage.ZRange(key, 0, 0, true)
			} else {
				members = zsetStorage.ZRevRange(key, 0, 0, true)
			}
			
			if len(members) < 2 {
				break
			}
			
			member := members[0].(string)
			score := members[1]
			
			// Remove the popped member
			zsetStorage.ZRem(key, []string{member})
			
			// Add to result
			elements = append(elements, member)
			if scoreStr, ok := score.(string); ok {
				elements = append(elements, scoreStr)
			} else {
				elements = append(elements, fmt.Sprintf("%.17g", score))
			}
		}

		if len(elements) > 0 {
			result := []interface{}{key, elements}
			return ctx.RespWriter.WriteArray(result)
		}
	}

	// No elements available
	return ctx.RespWriter.WriteNull()
}

func (c *ZMPopCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZMPOP",
		Summary:      "Remove and return members with scores in a sorted set",
		Syntax:       "ZMPOP numkeys key [key ...] MIN|MAX [COUNT count]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      3,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *ZMPopCommand) ModifiesData() bool { return true }

type ZRangeStoreCommand struct{}

func (c *ZRangeStoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 4 {
		return fmt.Errorf("wrong number of arguments for 'zrangestore' command")
	}

	destValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dest, ok := destValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	srcValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	src, ok := srcValue.AsString()
	if !ok {
		return fmt.Errorf("invalid source key")
	}

	startValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	startStr, ok := startValue.AsString()
	if !ok {
		return fmt.Errorf("invalid start")
	}
	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid start")
	}

	stopValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	stopStr, ok := stopValue.AsString()
	if !ok {
		return fmt.Errorf("invalid stop")
	}
	stop, err := strconv.ParseInt(stopStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid stop")
	}

	// Parse optional arguments (REV, BYSCORE, BYLEX, LIMIT)
	reverse := false
	for i := 4; i < nargs; i++ {
		optValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		opt, ok := optValue.AsString()
		if ok && strings.ToUpper(opt) == "REV" {
			reverse = true
		}
		// Note: BYSCORE, BYLEX, LIMIT not implemented in this simplified version
	}

	zsetStorage := ctx.Database.ZSetStorage
	
	// Clear destination key
	allMembers := zsetStorage.ZRange(dest, 0, -1, false)
	if len(allMembers) > 0 {
		memberStrings := make([]string, 0, len(allMembers))
		for _, member := range allMembers {
			if memberStr, ok := member.(string); ok {
				memberStrings = append(memberStrings, memberStr)
			}
		}
		if len(memberStrings) > 0 {
			zsetStorage.ZRem(dest, memberStrings)
		}
	}

	// Get range from source
	var members []interface{}
	if reverse {
		members = zsetStorage.ZRevRange(src, start, stop, true)
	} else {
		members = zsetStorage.ZRange(src, start, stop, true)
	}

	if len(members) == 0 {
		return ctx.RespWriter.WriteInteger(0)
	}

	// Add members to destination
	count := int64(0)
	for i := 0; i < len(members); i += 2 {
		member := members[i].(string)
		score := members[i+1]
		
		var scoreFloat float64
		if scoreStr, ok := score.(string); ok {
			if parsed, err := strconv.ParseFloat(scoreStr, 64); err == nil {
				scoreFloat = parsed
			}
		} else if scoreVal, ok := score.(float64); ok {
			scoreFloat = scoreVal
		}
		
		memberMap := map[string]float64{member: scoreFloat}
		zsetStorage.ZAdd(dest, memberMap)
		count++
	}

	return ctx.RespWriter.WriteInteger(count)
}

func (c *ZRangeStoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ZRANGESTORE",
		Summary:      "Store a range of members from sorted set into another key",
		Syntax:       "ZRANGESTORE dst src min max [BYSCORE|BYLEX] [REV] [LIMIT offset count]",
		Categories:   []engine.CommandCategory{engine.CategoryZSet},
		MinArgs:      4,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *ZRangeStoreCommand) ModifiesData() bool { return true }
