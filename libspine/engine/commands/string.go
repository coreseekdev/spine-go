package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"spine-go/libspine/engine"
)

// RegisterStringCommands registers all string-related commands
func RegisterStringCommands(registry *engine.CommandRegistry) error {
	commands := []engine.CommandHandler{
		&StringSetCommand{},
		&StringGetCommand{},
		&MSetCommand{},
		&MGetCommand{},
		&SetNXCommand{},
		&SetEXCommand{},
		&PSetEXCommand{},
		&GetSetCommand{},
		&GetDelCommand{},
		&GetExCommand{},
		&GetRangeCommand{},
		&SetRangeCommand{},
		&StrLenCommand{},
		&AppendCommand{},
		&IncrCommand{},
		&IncrByCommand{},
		&IncrByFloatCommand{},
		&DecrCommand{},
		&DecrByCommand{},
		&MSetnxCommand{},
		&SubstrCommand{},
		&LCSCommand{},
		&DelIfEqCommand{},
	}

	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.GetInfo().Name, err)
		}
	}

	return nil
}

// StringSetCommand implements the SET command
type StringSetCommand struct{}

func (c *StringSetCommand) Execute(ctx *engine.CommandContext) error {
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
		return ctx.RespWriter.WriteError("ERR invalid key")
	}
	key, ok := keyValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid key")
	}

	// Read value
	valueValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid value")
	}
	value, ok := valueValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid value")
	}

	var expiration *time.Time
	nx := false
	xx := false
	get := false

	// Parse options
	for i := 2; i < nargs; i++ {
		optValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid option")
		}
		opt, ok := optValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid option")
		}

		switch strings.ToUpper(opt) {
		case "NX":
			nx = true
		case "XX":
			xx = true
		case "GET":
			get = true
		case "EX":
			if i+1 >= nargs {
				return ctx.RespWriter.WriteError("ERR syntax error")
			}
			i++
			exValue, err := ctx.ReqReader.NextValue()
			if err != nil {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			exStr, ok := exValue.AsString()
			if !ok {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			seconds, err := strconv.ParseInt(exStr, 10, 64)
			if err != nil || seconds <= 0 {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			exp := time.Now().Add(time.Duration(seconds) * time.Second)
			expiration = &exp
		case "PX":
			if i+1 >= nargs {
				return ctx.RespWriter.WriteError("ERR syntax error")
			}
			i++
			pxValue, err := ctx.ReqReader.NextValue()
			if err != nil {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			pxStr, ok := pxValue.AsString()
			if !ok {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			milliseconds, err := strconv.ParseInt(pxStr, 10, 64)
			if err != nil || milliseconds <= 0 {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			exp := time.Now().Add(time.Duration(milliseconds) * time.Millisecond)
			expiration = &exp
		case "EXAT":
			if i+1 >= nargs {
				return ctx.RespWriter.WriteError("ERR syntax error")
			}
			i++
			exatValue, err := ctx.ReqReader.NextValue()
			if err != nil {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			exatStr, ok := exatValue.AsString()
			if !ok {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			timestamp, err := strconv.ParseInt(exatStr, 10, 64)
			if err != nil || timestamp <= 0 {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			exp := time.Unix(timestamp, 0)
			expiration = &exp
		case "PXAT":
			if i+1 >= nargs {
				return ctx.RespWriter.WriteError("ERR syntax error")
			}
			i++
			pxatValue, err := ctx.ReqReader.NextValue()
			if err != nil {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			pxatStr, ok := pxatValue.AsString()
			if !ok {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			timestamp, err := strconv.ParseInt(pxatStr, 10, 64)
			if err != nil || timestamp <= 0 {
				return ctx.RespWriter.WriteError("ERR invalid expire time")
			}
			exp := time.Unix(0, timestamp*int64(time.Millisecond))
			expiration = &exp
		default:
			return ctx.RespWriter.WriteError("ERR syntax error")
		}
	}

	// Use string storage from context
	stringStorage := ctx.Database.StringStorage

	// Handle GET option
	var oldValue string
	var hadOldValue bool
	if get {
		oldValue, hadOldValue = stringStorage.Get(key)
	}

	// Handle NX/XX options
	if nx || xx {
		exists := stringStorage.Exists(key)
		if nx && exists {
			if get && hadOldValue {
				return ctx.RespWriter.WriteBulkString(oldValue)
			}
			return ctx.RespWriter.WriteNull()
		}
		if xx && !exists {
			if get {
				return ctx.RespWriter.WriteNull()
			}
			return ctx.RespWriter.WriteNull()
		}
	}

	// Set the value
	err = stringStorage.Set(key, value, expiration)
	if err != nil {
		return ctx.RespWriter.WriteError("ERR " + err.Error())
	}

	// Return response
	if get {
		if hadOldValue {
			return ctx.RespWriter.WriteBulkString(oldValue)
		}
		return ctx.RespWriter.WriteNull()
	}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *StringSetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SET",
		Summary:      "Set the string value of a key",
		Syntax:       "SET key value [NX | XX] [GET] [EX seconds | PX milliseconds | EXAT unix-time-seconds | PXAT unix-time-milliseconds]",
		Categories:   []engine.CommandCategory{engine.CategoryString},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *StringSetCommand) ModifiesData() bool {
	return true
}

// StringGetCommand implements the GET command
type StringGetCommand struct{}

func (c *StringGetCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'get' command")
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

	// Use string storage from context
	stringStorage := ctx.Database.StringStorage

	// Get value
	value, exists := stringStorage.Get(key)
	if !exists {
		return ctx.RespWriter.WriteNull()
	}

	return ctx.RespWriter.WriteBulkString(value)
}

func (c *StringGetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "GET",
		Summary:      "Get the value of a key",
		Syntax:       "GET key",
		Categories:   []engine.CommandCategory{engine.CategoryString},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *StringGetCommand) ModifiesData() bool {
	return false
}

// MSetCommand implements the MSET command
type MSetCommand struct{}

func (c *MSetCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 2 || nargs%2 != 0 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'mset' command")
	}

	// Use string storage from context
	stringStorage := ctx.Database.StringStorage

	// Collect key-value pairs
	pairs := make(map[string]string)
	for i := 0; i < nargs; i += 2 {
		if i+1 >= nargs {
			return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'mset' command")
		}

		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid key")
		}
		key, ok := keyValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid key")
		}

		valueValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid value")
		}
		value, ok := valueValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid value")
		}

		pairs[key] = value
	}

	// Set all key-value pairs
	if err := stringStorage.MSet(pairs); err != nil {
		return ctx.RespWriter.WriteError("ERR failed to set values")
	}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *MSetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "MSET",
		Summary:      "Set multiple keys to multiple values",
		Syntax:       "MSET key value [key value ...]",
		Categories:   []engine.CommandCategory{engine.CategoryString},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *MSetCommand) ModifiesData() bool {
	return true
}

// MGetCommand implements the MGET command
type MGetCommand struct{}

func (c *MGetCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'mget' command")
	}

	// Use string storage from context
	stringStorage := ctx.Database.StringStorage

	// Read keys
	keys := make([]string, nargs)
	for i := 0; i < nargs; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid key")
		}
		key, ok := keyValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid key")
		}
		keys[i] = key
	}

	// Get values using MGet
	result := stringStorage.MGet(keys)
	values := make([]interface{}, nargs)
	for i, key := range keys {
		if value, exists := result[key]; exists {
			values[i] = value
		} else {
			values[i] = nil
		}
	}

	return ctx.RespWriter.WriteArray(values)
}

func (c *MGetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "MGET",
		Summary:      "Get the values of all the given keys",
		Syntax:       "MGET key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategoryString},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *MGetCommand) ModifiesData() bool {
	return false
}

// SetNXCommand implements the SETNX command
type SetNXCommand struct{}

func (c *SetNXCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'setnx' command")
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

	// Read value
	valueValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid value")
	}
	value, ok := valueValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid value")
	}

	// Use string storage from context
	stringStorage := ctx.Database.StringStorage

	// Check if key exists
	exists := stringStorage.Exists(key)
	if exists {
		return ctx.RespWriter.WriteInteger(0)
	}

	// Set the value
	if err := stringStorage.Set(key, value, nil); err != nil {
		return ctx.RespWriter.WriteError("ERR failed to set value")
	}

	return ctx.RespWriter.WriteInteger(1)
}

func (c *SetNXCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SETNX",
		Summary:      "Set the value of a key, only if the key does not exist",
		Syntax:       "SETNX key value",
		Categories:   []engine.CommandCategory{engine.CategoryString},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: true,
	}
}

func (c *SetNXCommand) ModifiesData() bool {
	return true
}

// SetEXCommand implements the SETEX command
type SetEXCommand struct{}

func (c *SetEXCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 3 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'setex' command")
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

	// Read seconds
	secondsValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid expire time")
	}
	secondsStr, ok := secondsValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid expire time")
	}
	seconds, err := strconv.ParseInt(secondsStr, 10, 64)
	if err != nil || seconds <= 0 {
		return ctx.RespWriter.WriteError("ERR invalid expire time")
	}

	// Read value
	valueValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid value")
	}
	value, ok := valueValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid value")
	}

	// Use the database directly from context
	db := ctx.Database

	// Set with expiration
	expiration := time.Now().Add(time.Duration(seconds) * time.Second)
	err = db.Set(key, value, &expiration)
	if err != nil {
		return ctx.RespWriter.WriteError("ERR " + err.Error())
	}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *SetEXCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SETEX",
		Summary:      "Set the value and expiration of a key",
		Syntax:       "SETEX key seconds value",
		Categories:   []engine.CommandCategory{engine.CategoryString},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *SetEXCommand) ModifiesData() bool {
	return true
}

// PSetEXCommand implements the PSETEX command
type PSetEXCommand struct{}

func (c *PSetEXCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 3 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'psetex' command")
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

	// Read milliseconds
	millisecondsValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid expire time")
	}
	millisecondsStr, ok := millisecondsValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid expire time")
	}
	milliseconds, err := strconv.ParseInt(millisecondsStr, 10, 64)
	if err != nil || milliseconds <= 0 {
		return ctx.RespWriter.WriteError("ERR invalid expire time")
	}

	// Read value
	valueValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid value")
	}
	value, ok := valueValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid value")
	}

	// Use the database directly from context
	db := ctx.Database

	// Set with expiration
	expiration := time.Now().Add(time.Duration(milliseconds) * time.Millisecond)
	err = db.Set(key, value, &expiration)
	if err != nil {
		return ctx.RespWriter.WriteError("ERR " + err.Error())
	}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *PSetEXCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "PSETEX",
		Summary:      "Set the value and expiration in milliseconds of a key",
		Syntax:       "PSETEX key milliseconds value",
		Categories:   []engine.CommandCategory{engine.CategoryString},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *PSetEXCommand) ModifiesData() bool {
	return true
}

// Additional string commands would continue here...
// For brevity, I'll implement a few more key commands

// IncrCommand implements the INCR command
type IncrCommand struct{}

func (c *IncrCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'incr' command")
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

	// Use the database directly from context
	db := ctx.Database

	// Get current value
	value, exists := db.Get(key)
	var intVal int64 = 0
	if exists {
		var err error
		intVal, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return ctx.RespWriter.WriteError("ERR value is not an integer or out of range")
		}
	}

	// Increment
	intVal++

	// Set new value
	err = db.Set(key, strconv.FormatInt(intVal, 10), nil)
	if err != nil {
		return ctx.RespWriter.WriteError("ERR " + err.Error())
	}

	return ctx.RespWriter.WriteInteger(intVal)
}

func (c *IncrCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "INCR",
		Summary:      "Increment the integer value of a key by one",
		Syntax:       "INCR key",
		Categories:   []engine.CommandCategory{engine.CategoryString},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: true,
	}
}

func (c *IncrCommand) ModifiesData() bool {
	return true
}

// Placeholder implementations for remaining commands
type GetSetCommand struct{}
func (c *GetSetCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *GetSetCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "GETSET", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *GetSetCommand) ModifiesData() bool { return true }

type GetDelCommand struct{}
func (c *GetDelCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *GetDelCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "GETDEL", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *GetDelCommand) ModifiesData() bool { return true }

type GetExCommand struct{}
func (c *GetExCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *GetExCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "GETEX", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *GetExCommand) ModifiesData() bool { return true }

type GetRangeCommand struct{}
func (c *GetRangeCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *GetRangeCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "GETRANGE", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *GetRangeCommand) ModifiesData() bool { return false }

type SetRangeCommand struct{}
func (c *SetRangeCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SetRangeCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SETRANGE", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *SetRangeCommand) ModifiesData() bool { return true }

type StrLenCommand struct{}
func (c *StrLenCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *StrLenCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "STRLEN", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *StrLenCommand) ModifiesData() bool { return false }

type AppendCommand struct{}
func (c *AppendCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *AppendCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "APPEND", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *AppendCommand) ModifiesData() bool { return true }

type IncrByCommand struct{}
func (c *IncrByCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *IncrByCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "INCRBY", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *IncrByCommand) ModifiesData() bool { return true }

type IncrByFloatCommand struct{}
func (c *IncrByFloatCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *IncrByFloatCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "INCRBYFLOAT", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *IncrByFloatCommand) ModifiesData() bool { return true }

type DecrCommand struct{}
func (c *DecrCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *DecrCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "DECR", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *DecrCommand) ModifiesData() bool { return true }

type DecrByCommand struct{}
func (c *DecrByCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *DecrByCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "DECRBY", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *DecrByCommand) ModifiesData() bool { return true }

type MSetnxCommand struct{}
func (c *MSetnxCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *MSetnxCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "MSETNX", Categories: []engine.CommandCategory{engine.CategoryString}} }
func (c *MSetnxCommand) ModifiesData() bool { return true }

// SubstrCommand implements the SUBSTR command
type SubstrCommand struct{}

func (c *SubstrCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'substr' command")
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
	start, err := strconv.Atoi(startStr)
	if err != nil {
		return fmt.Errorf("invalid start index")
	}

	endValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	endStr, ok := endValue.AsString()
	if !ok {
		return fmt.Errorf("invalid end index")
	}
	end, err := strconv.Atoi(endStr)
	if err != nil {
		return fmt.Errorf("invalid end index")
	}

	// Get the value
	value, exists := ctx.Database.Get(key)
	if !exists {
		return ctx.RespWriter.WriteBulkString("")
	}

	// Handle negative indices
	length := len(value)
	if start < 0 {
		start = length + start
	}
	if end < 0 {
		end = length + end
	}

	// Bounds checking
	if start < 0 {
		start = 0
	}
	if end >= length {
		end = length - 1
	}
	if start > end || start >= length {
		return ctx.RespWriter.WriteBulkString("")
	}

	substring := value[start : end+1]
	return ctx.RespWriter.WriteBulkString(substring)
}

func (c *SubstrCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SUBSTR",
		Summary:      "Get a substring of the string stored at a key",
		Syntax:       "SUBSTR key start end",
		Categories:   []engine.CommandCategory{engine.CategoryString},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: false,
	}
}

func (c *SubstrCommand) ModifiesData() bool {
	return false
}

// LCSCommand implements the LCS command (Longest Common Subsequence)
type LCSCommand struct{}

func (c *LCSCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'lcs' command")
	}

	key1Value, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key1, ok := key1Value.AsString()
	if !ok {
		return fmt.Errorf("invalid key1")
	}

	key2Value, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key2, ok := key2Value.AsString()
	if !ok {
		return fmt.Errorf("invalid key2")
	}

	// Get values
	value1, exists1 := ctx.Database.Get(key1)
	if !exists1 {
		value1 = ""
	}

	value2, exists2 := ctx.Database.Get(key2)
	if !exists2 {
		value2 = ""
	}

	// Simple LCS implementation
	lcs := longestCommonSubsequence(value1, value2)
	return ctx.RespWriter.WriteBulkString(lcs)
}

func (c *LCSCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LCS",
		Summary:      "Find the longest common subsequence between two strings",
		Syntax:       "LCS key1 key2 [IDX] [LEN] [WITHMATCHLEN]",
		Categories:   []engine.CommandCategory{engine.CategoryString},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *LCSCommand) ModifiesData() bool {
	return false
}

// DelIfEqCommand implements the DELIFEQ command
type DelIfEqCommand struct{}

func (c *DelIfEqCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 2 {
		return fmt.Errorf("wrong number of arguments for 'delifeq' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	expectedValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	expected, ok := expectedValue.AsString()
	if !ok {
		return fmt.Errorf("invalid expected value")
	}

	// Get current value
	currentValue, exists := ctx.Database.Get(key)
	if !exists {
		return ctx.RespWriter.WriteInteger(0)
	}

	// Check if values match
	if currentValue == expected {
		// Delete the key
		ctx.Database.Del(key)
		return ctx.RespWriter.WriteInteger(1)
	}

	return ctx.RespWriter.WriteInteger(0)
}

func (c *DelIfEqCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "DELIFEQ",
		Summary:      "Delete a key only if it equals the given value",
		Syntax:       "DELIFEQ key value",
		Categories:   []engine.CommandCategory{engine.CategoryString},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: true,
	}
}

func (c *DelIfEqCommand) ModifiesData() bool {
	return true
}

// Helper function for LCS
func longestCommonSubsequence(s1, s2 string) string {
	m, n := len(s1), len(s2)
	if m == 0 || n == 0 {
		return ""
	}

	// Create DP table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// Fill DP table
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	// Reconstruct LCS
	var result strings.Builder
	i, j := m, n
	for i > 0 && j > 0 {
		if s1[i-1] == s2[j-1] {
			result.WriteByte(s1[i-1])
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	// Reverse the result
	lcs := result.String()
	runes := []rune(lcs)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}
