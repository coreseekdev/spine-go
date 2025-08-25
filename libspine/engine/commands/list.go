package commands

import (
	"fmt"
	"strconv"

	"spine-go/libspine/engine"
	"spine-go/libspine/engine/storage"
)

// RegisterListCommands registers all list-related commands
func RegisterListCommands(registry *engine.CommandRegistry) error {
	commands := []engine.CommandHandler{
		&LPushCommand{},
		&RPushCommand{},
		&LPopCommand{},
		&RPopCommand{},
		&LLenCommand{},
		&LIndexCommand{},
		&LSetCommand{},
		&LRangeCommand{},
		&LTrimCommand{},
		&LRemCommand{},
		&LInsertCommand{},
		&LPushXCommand{},
		&RPushXCommand{},
		&RPopLPushCommand{},
		&BLPopCommand{},
		&BRPopCommand{},
		&BRPopLPushCommand{},
		&LPosCommand{},
		&LMoveCommand{},
		&BLMoveCommand{},
	}

	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.GetInfo().Name, err)
		}
	}

	return nil
}

// LPushCommand implements the LPUSH command
type LPushCommand struct{}

func (c *LPushCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'lpush' command")
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

	// Use list storage from context
	listStorage := ctx.Database.ListStorage

	// Read values
	values := make([]string, nargs-1)
	for i := 1; i < nargs; i++ {
		valueValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid value")
		}
		value, ok := valueValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid value")
		}
		values[i-1] = value
	}

	// Push values to the left of the list
	length, err := listStorage.LPush(key, values)
	if err != nil {
		return ctx.RespWriter.WriteError("ERR " + err.Error())
	}

	return ctx.RespWriter.WriteInteger(length)
}

func (c *LPushCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LPUSH",
		Summary:      "Prepend one or multiple elements to a list",
		Syntax:       "LPUSH key element [element ...]",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *LPushCommand) ModifiesData() bool {
	return true
}

// RPushCommand implements the RPUSH command
type RPushCommand struct{}

func (c *RPushCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'rpush' command")
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

	// Get or create list
	var listData []string
	if value, exists := db.GetValue(key); exists {
		if value.Type != storage.TypeList {
			return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
		if l, ok := value.Data.([]string); ok {
			listData = l
		}
	}

	// Read values and append to list
	for i := 1; i < nargs; i++ {
		valueValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid value")
		}
		value, ok := valueValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid value")
		}

		// Append to list
		listData = append(listData, value)
	}

	// Store the list
	listValue := &storage.Value{
		Type: storage.TypeList,
		Data: listData,
	}
	db.SetValue(key, listValue)

	return ctx.RespWriter.WriteInteger(int64(len(listData)))
}

func (c *RPushCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "RPUSH",
		Summary:      "Append one or multiple elements to a list",
		Syntax:       "RPUSH key element [element ...]",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *RPushCommand) ModifiesData() bool {
	return true
}

// LPopCommand implements the LPOP command
type LPopCommand struct{}

func (c *LPopCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 1 || nargs > 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'lpop' command")
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

	// Read count (optional)
	count := 1
	if nargs == 2 {
		countValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid count")
		}
		countStr, ok := countValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid count")
		}
		countInt, err := strconv.ParseInt(countStr, 10, 64)
		if err != nil || countInt < 0 {
			return ctx.RespWriter.WriteError("ERR value is not an integer or out of range")
		}
		count = int(countInt)
	}

	// Get database
	db := ctx.Database

	// Get list
	value, exists := db.GetValue(key)
	if !exists {
		if nargs == 1 {
			return ctx.RespWriter.WriteNull()
		}
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	if value.Type != storage.TypeList {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	listData, ok := value.Data.([]string)
	if !ok || len(listData) == 0 {
		if nargs == 1 {
			return ctx.RespWriter.WriteNull()
		}
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Pop elements from the left
	if count > len(listData) {
		count = len(listData)
	}

	poppedElements := listData[:count]
	remainingList := listData[count:]

	// Update or delete the list
	if len(remainingList) == 0 {
		db.Del(key)
	} else {
		value.Data = remainingList
		db.SetValue(key, value)
	}

	// Return result
	if nargs == 1 {
		if len(poppedElements) > 0 {
			return ctx.RespWriter.WriteBulkString(poppedElements[0])
		}
		return ctx.RespWriter.WriteNull()
	} else {
		result := make([]interface{}, len(poppedElements))
		for i, elem := range poppedElements {
			result[i] = elem
		}
		return ctx.RespWriter.WriteArray(result)
	}
}

func (c *LPopCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LPOP",
		Summary:      "Remove and get the first elements in a list",
		Syntax:       "LPOP key [count]",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      1,
		MaxArgs:      2,
		ModifiesData: true,
	}
}

func (c *LPopCommand) ModifiesData() bool {
	return true
}

// RPopCommand implements the RPOP command
type RPopCommand struct{}

func (c *RPopCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 1 || nargs > 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'rpop' command")
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

	// Read count (optional)
	count := 1
	if nargs == 2 {
		countValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid count")
		}
		countStr, ok := countValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid count")
		}
		countInt, err := strconv.ParseInt(countStr, 10, 64)
		if err != nil || countInt < 0 {
			return ctx.RespWriter.WriteError("ERR value is not an integer or out of range")
		}
		count = int(countInt)
	}

	// Get database
	db := ctx.Database

	// Get list
	value, exists := db.GetValue(key)
	if !exists {
		if nargs == 1 {
			return ctx.RespWriter.WriteNull()
		}
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	if value.Type != storage.TypeList {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	listData, ok := value.Data.([]string)
	if !ok || len(listData) == 0 {
		if nargs == 1 {
			return ctx.RespWriter.WriteNull()
		}
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Pop elements from the right
	if count > len(listData) {
		count = len(listData)
	}

	startIndex := len(listData) - count
	poppedElements := listData[startIndex:]
	remainingList := listData[:startIndex]

	// Reverse the popped elements to maintain order
	for i, j := 0, len(poppedElements)-1; i < j; i, j = i+1, j-1 {
		poppedElements[i], poppedElements[j] = poppedElements[j], poppedElements[i]
	}

	// Update or delete the list
	if len(remainingList) == 0 {
		db.Del(key)
	} else {
		value.Data = remainingList
		db.SetValue(key, value)
	}

	// Return result
	if nargs == 1 {
		if len(poppedElements) > 0 {
			return ctx.RespWriter.WriteBulkString(poppedElements[0])
		}
		return ctx.RespWriter.WriteNull()
	} else {
		result := make([]interface{}, len(poppedElements))
		for i, elem := range poppedElements {
			result[i] = elem
		}
		return ctx.RespWriter.WriteArray(result)
	}
}

func (c *RPopCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "RPOP",
		Summary:      "Remove and get the last elements in a list",
		Syntax:       "RPOP key [count]",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      1,
		MaxArgs:      2,
		ModifiesData: true,
	}
}

func (c *RPopCommand) ModifiesData() bool {
	return true
}

// LLenCommand implements the LLEN command
type LLenCommand struct{}

func (c *LLenCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'llen' command")
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

	// Get list
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteInteger(0)
	}

	if value.Type != storage.TypeList {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	listData, ok := value.Data.([]string)
	if !ok {
		return ctx.RespWriter.WriteInteger(0)
	}

	return ctx.RespWriter.WriteInteger(int64(len(listData)))
}

func (c *LLenCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LLEN",
		Summary:      "Get the length of a list",
		Syntax:       "LLEN key",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *LLenCommand) ModifiesData() bool {
	return false
}

// LIndexCommand implements the LINDEX command
type LIndexCommand struct{}

func (c *LIndexCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'lindex' command")
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

	// Read index
	indexValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid index")
	}
	indexStr, ok := indexValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid index")
	}
	index, err := strconv.ParseInt(indexStr, 10, 64)
	if err != nil {
		return ctx.RespWriter.WriteError("ERR value is not an integer or out of range")
	}

	// Get database
	db := ctx.Database

	// Get list
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteNull()
	}

	if value.Type != storage.TypeList {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	listData, ok := value.Data.([]string)
	if !ok {
		return ctx.RespWriter.WriteNull()
	}

	// Handle negative indices
	if index < 0 {
		index = int64(len(listData)) + index
	}

	// Check bounds
	if index < 0 || index >= int64(len(listData)) {
		return ctx.RespWriter.WriteNull()
	}

	return ctx.RespWriter.WriteBulkString(listData[index])
}

func (c *LIndexCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LINDEX",
		Summary:      "Get an element from a list by its index",
		Syntax:       "LINDEX key index",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *LIndexCommand) ModifiesData() bool {
	return false
}

// LRangeCommand implements the LRANGE command
type LRangeCommand struct{}

func (c *LRangeCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 3 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'lrange' command")
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

	// Get database
	db := ctx.Database

	// Get list
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	if value.Type != storage.TypeList {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	listData, ok := value.Data.([]string)
	if !ok {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	length := int64(len(listData))

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
	result := make([]interface{}, stop-start+1)
	for i := start; i <= stop; i++ {
		result[i-start] = listData[i]
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *LRangeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LRANGE",
		Summary:      "Get a range of elements from a list",
		Syntax:       "LRANGE key start stop",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: false,
	}
}

func (c *LRangeCommand) ModifiesData() bool {
	return false
}

// Placeholder implementations for remaining list commands
type LSetCommand struct{}
func (c *LSetCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *LSetCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "LSET", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *LSetCommand) ModifiesData() bool { return true }

type LTrimCommand struct{}
func (c *LTrimCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *LTrimCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "LTRIM", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *LTrimCommand) ModifiesData() bool { return true }

type LRemCommand struct{}
func (c *LRemCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *LRemCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "LREM", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *LRemCommand) ModifiesData() bool { return true }

type LInsertCommand struct{}
func (c *LInsertCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *LInsertCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "LINSERT", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *LInsertCommand) ModifiesData() bool { return true }

type LPushXCommand struct{}
func (c *LPushXCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *LPushXCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "LPUSHX", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *LPushXCommand) ModifiesData() bool { return true }

type RPushXCommand struct{}
func (c *RPushXCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *RPushXCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "RPUSHX", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *RPushXCommand) ModifiesData() bool { return true }

type RPopLPushCommand struct{}
func (c *RPopLPushCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *RPopLPushCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "RPOPLPUSH", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *RPopLPushCommand) ModifiesData() bool { return true }

type BLPopCommand struct{}
func (c *BLPopCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *BLPopCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "BLPOP", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *BLPopCommand) ModifiesData() bool { return true }

type BRPopCommand struct{}
func (c *BRPopCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *BRPopCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "BRPOP", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *BRPopCommand) ModifiesData() bool { return true }

type BRPopLPushCommand struct{}
func (c *BRPopLPushCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *BRPopLPushCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "BRPOPLPUSH", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *BRPopLPushCommand) ModifiesData() bool { return true }

type LPosCommand struct{}
func (c *LPosCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *LPosCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "LPOS", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *LPosCommand) ModifiesData() bool { return false }

type LMoveCommand struct{}
func (c *LMoveCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *LMoveCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "LMOVE", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *LMoveCommand) ModifiesData() bool { return true }

type BLMoveCommand struct{}
func (c *BLMoveCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *BLMoveCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "BLMOVE", Categories: []engine.CommandCategory{engine.CategoryList}} }
func (c *BLMoveCommand) ModifiesData() bool { return true }
