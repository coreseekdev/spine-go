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
	registry.RegisterCommand("SET", setCmd)

	// GET 命令
	getCmd := &GetCommand{}
	registry.RegisterCommand("GET", getCmd)

	// DEL 命令
	delCmd := &DelCommand{}
	registry.RegisterCommand("DEL", delCmd)
	registry.RegisterAlias("DEL", "DELETE") // 添加别名

	// EXISTS 命令
	existsCmd := &ExistsCommand{}
	registry.RegisterCommand("EXISTS", existsCmd)

	// TYPE 命令
	typeCmd := &TypeCommand{}
	registry.RegisterCommand("TYPE", typeCmd)

	// EXPIRE 命令
	expireCmd := &ExpireCommand{}
	registry.RegisterCommand("EXPIRE", expireCmd)

	// TTL 命令
	ttlCmd := &TTLCommand{}
	registry.RegisterCommand("TTL", ttlCmd)

	// KEYS 命令
	keysCmd := &KeysCommand{}
	registry.RegisterCommand("KEYS", keysCmd)

	// FLUSHDB 命令
	flushdbCmd := &FlushDBCommand{}
	registry.RegisterCommand("FLUSHDB", flushdbCmd)

	// DBSIZE 命令
	dbsizeCmd := &DBSizeCommand{}
	registry.RegisterCommand("DBSIZE", dbsizeCmd)

	return nil
}

// SetCommand implements the SET command
type SetCommand struct{}

func (c *SetCommand) Execute(ctx *engine.CommandContext) error {
	if len(ctx.Args) < 2 {
		return writeRESP3Error(ctx.Writer, "ERR wrong number of arguments for 'set' command")
	}

	key := ctx.Args[0]
	value := ctx.Args[1]
	var expiration *time.Time

	// Parse optional arguments (EX, PX, EXAT, PXAT, NX, XX)
	for i := 2; i < len(ctx.Args); i += 2 {
		if i+1 >= len(ctx.Args) {
			return writeRESP3Error(ctx.Writer, "ERR syntax error")
		}

		option := strings.ToUpper(ctx.Args[i])
		arg := ctx.Args[i+1]

		switch option {
		case "EX": // seconds
			seconds, err := strconv.ParseInt(arg, 10, 64)
			if err != nil || seconds <= 0 {
				return writeRESP3Error(ctx.Writer, "ERR invalid expire time in set")
			}
			exp := time.Now().Add(time.Duration(seconds) * time.Second)
			expiration = &exp
		case "PX": // milliseconds
			milliseconds, err := strconv.ParseInt(arg, 10, 64)
			if err != nil || milliseconds <= 0 {
				return writeRESP3Error(ctx.Writer, "ERR invalid expire time in set")
			}
			exp := time.Now().Add(time.Duration(milliseconds) * time.Millisecond)
			expiration = &exp
		default:
			return writeRESP3Error(ctx.Writer, "ERR syntax error")
		}
	}

	err := ctx.Database.Set(key, value, expiration)
	if err != nil {
		return writeRESP3Error(ctx.Writer, fmt.Sprintf("ERR %s", err.Error()))
	}

	return writeRESP3SimpleString(ctx.Writer, "OK")
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
	if len(ctx.Args) != 1 {
		return writeRESP3Error(ctx.Writer, "ERR wrong number of arguments for 'get' command")
	}

	key := ctx.Args[0]
	value, exists := ctx.Database.Get(key)
	if !exists {
		return writeRESP3Null(ctx.Writer)
	}

	return writeRESP3BulkString(ctx.Writer, value)
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
	if len(ctx.Args) == 0 {
		return writeRESP3Error(ctx.Writer, "ERR wrong number of arguments for 'del' command")
	}

	count := ctx.Database.Del(ctx.Args...)
	return writeRESP3Integer(ctx.Writer, int64(count))
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
	if len(ctx.Args) == 0 {
		return writeRESP3Error(ctx.Writer, "ERR wrong number of arguments for 'exists' command")
	}

	count := ctx.Database.Exists(ctx.Args...)
	return writeRESP3Integer(ctx.Writer, int64(count))
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
	if len(ctx.Args) != 1 {
		return writeRESP3Error(ctx.Writer, "ERR wrong number of arguments for 'type' command")
	}

	key := ctx.Args[0]
	valueType, exists := ctx.Database.Type(key)
	if !exists {
		return writeRESP3SimpleString(ctx.Writer, "none")
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

	return writeRESP3SimpleString(ctx.Writer, typeStr)
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
	if len(ctx.Args) != 2 {
		return writeRESP3Error(ctx.Writer, "ERR wrong number of arguments for 'expire' command")
	}

	key := ctx.Args[0]
	seconds, err := strconv.ParseInt(ctx.Args[1], 10, 64)
	if err != nil {
		return writeRESP3Error(ctx.Writer, "ERR value is not an integer or out of range")
	}

	expiration := time.Now().Add(time.Duration(seconds) * time.Second)
	success := ctx.Database.Expire(key, expiration)

	if success {
		return writeRESP3Integer(ctx.Writer, 1)
	}
	return writeRESP3Integer(ctx.Writer, 0)
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
	if len(ctx.Args) != 1 {
		return writeRESP3Error(ctx.Writer, "ERR wrong number of arguments for 'ttl' command")
	}

	key := ctx.Args[0]
	ttl, exists := ctx.Database.TTL(key)
	if !exists {
		return writeRESP3Integer(ctx.Writer, -2) // Key doesn't exist
	}

	if ttl == -1 {
		return writeRESP3Integer(ctx.Writer, -1) // No expiration
	}

	return writeRESP3Integer(ctx.Writer, int64(ttl.Seconds()))
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
	if len(ctx.Args) != 1 {
		return writeRESP3Error(ctx.Writer, "ERR wrong number of arguments for 'keys' command")
	}

	pattern := ctx.Args[0]
	keys := ctx.Database.Keys(pattern)

	return writeRESP3Array(ctx.Writer, keys)
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
	return writeRESP3SimpleString(ctx.Writer, "OK")
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
	return writeRESP3Integer(ctx.Writer, int64(size))
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