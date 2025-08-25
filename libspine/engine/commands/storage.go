package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"spine-go/libspine/engine"
)

// RegisterStorageCommands 注册存储命令
func RegisterStorageCommands(registry *engine.CommandRegistry) error {
	// SET 命令
	setCmd := &SetCommand{}
	if err := registry.Register(setCmd); err != nil {
		return err
	}

	// GET 命令
	getCmd := &GetCommand{}
	if err := registry.Register(getCmd); err != nil {
		return err
	}

	// DEL 命令
	delCmd := &DelCommand{}
	if err := registry.Register(delCmd); err != nil {
		return err
	}
	registry.RegisterAlias("DEL", "DELETE") // 添加别名

	// EXISTS 命令
	existsCmd := &ExistsCommand{}
	if err := registry.Register(existsCmd); err != nil {
		return err
	}

	// TYPE 命令
	typeCmd := &TypeCommand{}
	if err := registry.Register(typeCmd); err != nil {
		return err
	}

	// EXPIRE 命令
	expireCmd := &ExpireCommand{}
	if err := registry.Register(expireCmd); err != nil {
		return err
	}

	// TTL 命令
	ttlCmd := &TTLCommand{}
	if err := registry.Register(ttlCmd); err != nil {
		return err
	}

	// KEYS 命令
	keysCmd := &KeysCommand{}
	if err := registry.Register(keysCmd); err != nil {
		return err
	}

	// FLUSHDB 命令
	flushdbCmd := &FlushDBCommand{}
	if err := registry.Register(flushdbCmd); err != nil {
		return err
	}

	// DBSIZE 命令
	dbsizeCmd := &DBSizeCommand{}
	if err := registry.Register(dbsizeCmd); err != nil {
		return err
	}

	return nil
}

// SetCommand implements the SET command
type SetCommand struct{}

func (c *SetCommand) Execute(ctx *engine.CommandContext) error {
	// Get the number of arguments first
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}
	
	if nargs < 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'set' command")
	}

	// Read key
	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'set' command")
	}

	key, ok := keyValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}

	// Read value
	valValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'set' command")
	}

	value, ok := valValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid value")
	}
	
	var expiration *time.Time

	// Parse optional arguments (EX, PX, EXAT, PXAT, NX, XX)
	for {
		optionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			// No more arguments
			break
		}

		option, ok := optionValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR syntax error")
		}

		option = strings.ToUpper(option)

		argValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR syntax error")
		}

		switch option {
		case "EX": // seconds
			arg, ok := argValue.AsString()
			if !ok {
				return ctx.RespWriter.WriteError("ERR invalid argument")
			}
			seconds, err := strconv.ParseInt(arg, 10, 64)
			if err != nil || seconds <= 0 {
				return ctx.RespWriter.WriteError("ERR invalid expire time in set")
			}
			exp := time.Now().Add(time.Duration(seconds) * time.Second)
			expiration = &exp
		case "PX": // milliseconds
			arg, ok := argValue.AsString()
			if !ok {
				return ctx.RespWriter.WriteError("ERR invalid argument")
			}
			milliseconds, err := strconv.ParseInt(arg, 10, 64)
			if err != nil || milliseconds <= 0 {
				return ctx.RespWriter.WriteError("ERR invalid expire time in set")
			}
			exp := time.Now().Add(time.Duration(milliseconds) * time.Millisecond)
			expiration = &exp
		default:
			return ctx.RespWriter.WriteError("ERR syntax error")
		}
	}

	err = ctx.Database.Set(key, value, expiration)
	if err != nil {
		return ctx.RespWriter.WriteError(fmt.Sprintf("ERR %s", err.Error()))
	}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *SetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:        "SET",
		Summary:     "Set key to hold the string value.",
		Syntax:      "SET key value [EX seconds] [PX milliseconds] [NX|XX]",
		Categories:  []engine.CommandCategory{engine.CategoryString, engine.CategoryWrite},
		MinArgs:     2,
		MaxArgs:     -1, // 可变参数
		ModifiesData: true,

	}
}

func (c *SetCommand) ModifiesData() bool {
	return true
}

// GetCommand implements the GET command
type GetCommand struct{}

func (c *GetCommand) Execute(ctx *engine.CommandContext) error {
	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'get' command")
	}

	key, ok := keyValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}
	value, exists := ctx.Database.Get(key)
	if !exists {
		return ctx.RespWriter.WriteNull()
	}

	return ctx.RespWriter.WriteBulkString(value)
}

func (c *GetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:        "GET",
		Summary:     "Get the value of key.",
		Syntax:      "GET key",
		Categories:  []engine.CommandCategory{engine.CategoryString, engine.CategoryRead},
		MinArgs:     1,
		MaxArgs:     1,
		ModifiesData: false,

	}
}

func (c *GetCommand) ModifiesData() bool {
	return false
}

// DelCommand implements the DEL command
type DelCommand struct{}

func (c *DelCommand) Execute(ctx *engine.CommandContext) error {
	// Get the number of arguments first
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}
	
	if nargs < 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'del' command")
	}

	var keys []string
	for i := 0; i < nargs; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid key")
		}
		key, ok := keyValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid key")
		}
		keys = append(keys, key)
	}

	count := ctx.Database.Del(keys...)
	return ctx.RespWriter.WriteInteger(int64(count))
}

func (c *DelCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:        "DEL",
		Summary:     "Delete a key.",
		Syntax:      "DEL key [key ...]",
		Categories:  []engine.CommandCategory{engine.CategoryGeneric, engine.CategoryWrite},
		MinArgs:     1,
		MaxArgs:     -1, // 可变参数
		ModifiesData: true,

	}
}

func (c *DelCommand) ModifiesData() bool {
	return true
}

// ExistsCommand implements the EXISTS command
type ExistsCommand struct{}

func (c *ExistsCommand) Execute(ctx *engine.CommandContext) error {
	// Get the number of arguments first
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}
	
	if nargs < 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'exists' command")
	}

	var keys []string
	for i := 0; i < nargs; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid key")
		}
		key, ok := keyValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid key")
		}
		keys = append(keys, key)
	}

	count := ctx.Database.Exists(keys...)
	return ctx.RespWriter.WriteInteger(int64(count))
}

func (c *ExistsCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:        "EXISTS",
		Summary:     "Determine if a key exists.",
		Syntax:      "EXISTS key [key ...]",
		Categories:  []engine.CommandCategory{engine.CategoryGeneric, engine.CategoryRead},
		MinArgs:     1,
		MaxArgs:     -1, // 可变参数
		ModifiesData: false,

	}
}

func (c *ExistsCommand) ModifiesData() bool {
	return false
}

// TypeCommand implements the TYPE command
type TypeCommand struct{}

func (c *TypeCommand) Execute(ctx *engine.CommandContext) error {
	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'type' command")
	}

	key, ok := keyValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}
	valueType, exists := ctx.Database.Type(key)
	if !exists {
		return ctx.RespWriter.WriteSimpleString("none")
	}

	var typeStr string
	switch valueType {
	case 0: // TypeString
		typeStr = "string"
	case 1: // TypeList
		typeStr = "list"
	case 2: // TypeSet
		typeStr = "set"
	case 3: // TypeZSet
		typeStr = "zset"
	case 4: // TypeHash
		typeStr = "hash"
	case 5: // TypeStream
		typeStr = "stream"
	default:
		typeStr = "unknown"
	}

	return ctx.RespWriter.WriteSimpleString( typeStr)
}

func (c *TypeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:        "TYPE",
		Summary:     "Determine the type stored at key.",
		Syntax:      "TYPE key",
		Categories:  []engine.CommandCategory{engine.CategoryGeneric, engine.CategoryRead},
		MinArgs:     1,
		MaxArgs:     1,
		ModifiesData: false,

	}
}

func (c *TypeCommand) ModifiesData() bool {
	return false
}

// ExpireCommand implements the EXPIRE command
type ExpireCommand struct{}

func (c *ExpireCommand) Execute(ctx *engine.CommandContext) error {
	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'expire' command")
	}

	key, ok := keyValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}

	secondsValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'expire' command")
	}

	secondsStr, ok := secondsValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid seconds")
	}

	seconds, err := strconv.ParseInt(secondsStr, 10, 64)
	if err != nil {
		return ctx.RespWriter.WriteError("ERR value is not an integer or out of range")
	}

	expiration := time.Now().Add(time.Duration(seconds) * time.Second)
	success := ctx.Database.Expire(key, expiration)

	if success {
		return ctx.RespWriter.WriteInteger(1)
	}
	return ctx.RespWriter.WriteInteger(0)
}

func (c *ExpireCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:        "EXPIRE",
		Summary:     "Set a key's time to live in seconds.",
		Syntax:      "EXPIRE key seconds",
		Categories:  []engine.CommandCategory{engine.CategoryGeneric, engine.CategoryWrite},
		MinArgs:     2,
		MaxArgs:     2,
		ModifiesData: true,

	}
}

func (c *ExpireCommand) ModifiesData() bool {
	return true
}

// TTLCommand implements the TTL command
type TTLCommand struct{}

func (c *TTLCommand) Execute(ctx *engine.CommandContext) error {
	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'ttl' command")
	}

	key, ok := keyValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}
	ttl, exists := ctx.Database.TTL(key)
	if !exists {
		return ctx.RespWriter.WriteInteger(-2) // Key doesn't exist
	}

	if ttl == -1 {
		return ctx.RespWriter.WriteInteger(-1) // No expiration
	}

	return ctx.RespWriter.WriteInteger(int64(ttl.Seconds()))
}

func (c *TTLCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:        "TTL",
		Summary:     "Get the time to live for a key.",
		Syntax:      "TTL key",
		Categories:  []engine.CommandCategory{engine.CategoryGeneric, engine.CategoryRead},
		MinArgs:     1,
		MaxArgs:     1,
		ModifiesData: false,

	}
}

func (c *TTLCommand) ModifiesData() bool {
	return false
}

// KeysCommand implements the KEYS command
type KeysCommand struct{}

func (c *KeysCommand) Execute(ctx *engine.CommandContext) error {
	patternValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'keys' command")
	}

	pattern, ok := patternValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid pattern")
	}
	keys := ctx.Database.Keys(pattern)

	// Convert []string to []interface{}
	keysInterface := make([]interface{}, len(keys))
	for i, k := range keys {
		keysInterface[i] = k
	}

	return ctx.RespWriter.WriteArray(keysInterface)
}

func (c *KeysCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:        "KEYS",
		Summary:     "Find all keys matching the given pattern.",
		Syntax:      "KEYS pattern",
		Categories:  []engine.CommandCategory{engine.CategoryGeneric, engine.CategoryRead},
		MinArgs:     1,
		MaxArgs:     1,
		ModifiesData: false,

	}
}

func (c *KeysCommand) ModifiesData() bool {
	return false
}

// FlushDBCommand implements the FLUSHDB command
type FlushDBCommand struct{}

func (c *FlushDBCommand) Execute(ctx *engine.CommandContext) error {
	ctx.Database.FlushDB()
	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *FlushDBCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:        "FLUSHDB",
		Summary:     "Remove all keys from the current database.",
		Syntax:      "FLUSHDB [ASYNC]",
		Categories:  []engine.CommandCategory{engine.CategoryGeneric, engine.CategoryWrite},
		MinArgs:     0,
		MaxArgs:     1,
		ModifiesData: true,

	}
}

func (c *FlushDBCommand) ModifiesData() bool {
	return true
}

// DBSizeCommand implements the DBSIZE command
type DBSizeCommand struct{}

func (c *DBSizeCommand) Execute(ctx *engine.CommandContext) error {
	size := ctx.Database.DBSize()
	return ctx.RespWriter.WriteInteger(int64(size))
}

func (c *DBSizeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:        "DBSIZE",
		Summary:     "Return the number of keys in the selected database.",
		Syntax:      "DBSIZE",
		Categories:  []engine.CommandCategory{engine.CategoryGeneric, engine.CategoryRead},
		MinArgs:     0,
		MaxArgs:     0,
		ModifiesData: false,

	}
}

func (c *DBSizeCommand) ModifiesData() bool {
	return false
}