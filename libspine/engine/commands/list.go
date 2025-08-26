package commands

import (
	"fmt"
	"strconv"
	"strings"

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
		&LMPopCommand{},
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

func (c *LSetCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'lset' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	indexValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	indexStr, ok := indexValue.AsString()
	if !ok {
		return fmt.Errorf("invalid index")
	}

	index, err := strconv.ParseInt(indexStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid index")
	}

	elementValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	element, ok := elementValue.AsString()
	if !ok {
		return fmt.Errorf("invalid element")
	}

	listStorage := ctx.Database.ListStorage
	err = listStorage.LSet(key, index, element)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *LSetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LSET",
		Summary:      "Set the list element at index to element",
		Syntax:       "LSET key index element",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *LSetCommand) ModifiesData() bool { return true }

type LTrimCommand struct{}

func (c *LTrimCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'ltrim' command")
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

	listStorage := ctx.Database.ListStorage
	err = listStorage.LTrim(key, start, stop)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *LTrimCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LTRIM",
		Summary:      "Trim a list to the specified range",
		Syntax:       "LTRIM key start stop",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *LTrimCommand) ModifiesData() bool { return true }

type LRemCommand struct{}

func (c *LRemCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'lrem' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	countValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	countStr, ok := countValue.AsString()
	if !ok {
		return fmt.Errorf("invalid count")
	}

	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid count")
	}

	elementValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	element, ok := elementValue.AsString()
	if !ok {
		return fmt.Errorf("invalid element")
	}

	listStorage := ctx.Database.ListStorage
	removed := listStorage.LRem(key, count, element)

	return ctx.RespWriter.WriteInteger(removed)
}

func (c *LRemCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LREM",
		Summary:      "Remove elements from a list",
		Syntax:       "LREM key count element",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *LRemCommand) ModifiesData() bool { return true }

type LInsertCommand struct{}

func (c *LInsertCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 4 {
		return fmt.Errorf("wrong number of arguments for 'linsert' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	whereValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	where, ok := whereValue.AsString()
	if !ok {
		return fmt.Errorf("invalid where")
	}

	pivotValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	pivot, ok := pivotValue.AsString()
	if !ok {
		return fmt.Errorf("invalid pivot")
	}

	elementValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	element, ok := elementValue.AsString()
	if !ok {
		return fmt.Errorf("invalid element")
	}

	listStorage := ctx.Database.ListStorage
	length := listStorage.LLen(key)
	if length == 0 {
		return ctx.RespWriter.WriteInteger(0)
	}

	// Find pivot position
	elements := listStorage.LRange(key, 0, -1)
	pivotIndex := -1
	for i, elem := range elements {
		if elem == pivot {
			pivotIndex = i
			break
		}
	}

	if pivotIndex == -1 {
		return ctx.RespWriter.WriteInteger(-1)
	}

	// Insert element
	insertIndex := pivotIndex
	if strings.ToUpper(where) == "AFTER" {
		insertIndex++
	}

	// Rebuild list with inserted element
	newElements := make([]string, 0, len(elements)+1)
	newElements = append(newElements, elements[:insertIndex]...)
	newElements = append(newElements, element)
	newElements = append(newElements, elements[insertIndex:]...)

	// Clear and rebuild list
	listStorage.LTrim(key, 1, 0) // Clear list
	if len(newElements) > 0 {
		listStorage.RPush(key, newElements)
	}

	return ctx.RespWriter.WriteInteger(int64(len(newElements)))
}

func (c *LInsertCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LINSERT",
		Summary:      "Insert an element before or after another element in a list",
		Syntax:       "LINSERT key BEFORE|AFTER pivot element",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      4,
		MaxArgs:      4,
		ModifiesData: true,
	}
}

func (c *LInsertCommand) ModifiesData() bool { return true }

type LPushXCommand struct{}

func (c *LPushXCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'lpushx' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	listStorage := ctx.Database.ListStorage
	// Only push if list exists
	if listStorage.LLen(key) == 0 {
		return ctx.RespWriter.WriteInteger(0)
	}

	values := make([]string, nargs-1)
	for i := 1; i < nargs; i++ {
		valueArg, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		value, ok := valueArg.AsString()
		if !ok {
			return fmt.Errorf("invalid value")
		}
		values[i-1] = value
	}

	newLength, err := listStorage.LPush(key, values)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteInteger(newLength)
}

func (c *LPushXCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LPUSHX",
		Summary:      "Prepend one or multiple elements to a list, only if the list exists",
		Syntax:       "LPUSHX key element [element ...]",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *LPushXCommand) ModifiesData() bool { return true }

type RPushXCommand struct{}

func (c *RPushXCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'rpushx' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	listStorage := ctx.Database.ListStorage
	// Only push if list exists
	if listStorage.LLen(key) == 0 {
		return ctx.RespWriter.WriteInteger(0)
	}

	values := make([]string, nargs-1)
	for i := 1; i < nargs; i++ {
		valueArg, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		value, ok := valueArg.AsString()
		if !ok {
			return fmt.Errorf("invalid value")
		}
		values[i-1] = value
	}

	newLength, err := listStorage.RPush(key, values)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteInteger(newLength)
}

func (c *RPushXCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "RPUSHX",
		Summary:      "Append one or multiple elements to a list, only if the list exists",
		Syntax:       "RPUSHX key element [element ...]",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *RPushXCommand) ModifiesData() bool { return true }

type RPopLPushCommand struct{}

func (c *RPopLPushCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 2 {
		return fmt.Errorf("wrong number of arguments for 'rpoplpush' command")
	}

	sourceValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	source, ok := sourceValue.AsString()
	if !ok {
		return fmt.Errorf("invalid source key")
	}

	destValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dest, ok := destValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	listStorage := ctx.Database.ListStorage
	// Pop from source
	element, exists := listStorage.RPop(source)
	if !exists {
		return ctx.RespWriter.WriteNull()
	}

	// Push to destination
	_, err = listStorage.LPush(dest, []string{element})
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteBulkString(element)
}

func (c *RPopLPushCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "RPOPLPUSH",
		Summary:      "Remove the last element in a list, prepend it to another list and return it",
		Syntax:       "RPOPLPUSH source destination",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: true,
	}
}

func (c *RPopLPushCommand) ModifiesData() bool { return true }

type BLPopCommand struct{}

func (c *BLPopCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'blpop' command")
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

	listStorage := ctx.Database.ListStorage
	// Try to pop from each key in order (non-blocking implementation)
	for _, key := range keys {
		if listStorage.LLen(key) > 0 {
			element, exists := listStorage.LPop(key)
			if exists {
				result := []interface{}{key, element}
				return ctx.RespWriter.WriteArray(result)
			}
		}
	}

	// No elements available (simplified: return null instead of blocking)
	return ctx.RespWriter.WriteNull()
}

func (c *BLPopCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "BLPOP",
		Summary:      "Remove and get the first element in a list, or block until one is available",
		Syntax:       "BLPOP key [key ...] timeout",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *BLPopCommand) ModifiesData() bool { return true }

type BRPopCommand struct{}

func (c *BRPopCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'brpop' command")
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

	listStorage := ctx.Database.ListStorage
	// Try to pop from each key in order (non-blocking implementation)
	for _, key := range keys {
		if listStorage.LLen(key) > 0 {
			element, exists := listStorage.RPop(key)
			if exists {
				result := []interface{}{key, element}
				return ctx.RespWriter.WriteArray(result)
			}
		}
	}

	// No elements available (simplified: return null instead of blocking)
	return ctx.RespWriter.WriteNull()
}

func (c *BRPopCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "BRPOP",
		Summary:      "Remove and get the last element in a list, or block until one is available",
		Syntax:       "BRPOP key [key ...] timeout",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *BRPopCommand) ModifiesData() bool { return true }

type BRPopLPushCommand struct{}

func (c *BRPopLPushCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'brpoplpush' command")
	}

	sourceValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	source, ok := sourceValue.AsString()
	if !ok {
		return fmt.Errorf("invalid source key")
	}

	destValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dest, ok := destValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	// Read timeout (ignored in this simplified implementation)
	_, err = ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}

	listStorage := ctx.Database.ListStorage
	// Non-blocking implementation: try once
	if listStorage.LLen(source) > 0 {
		element, exists := listStorage.RPop(source)
		if exists {
			_, err = listStorage.LPush(dest, []string{element})
			if err != nil {
				return err
			}
			return ctx.RespWriter.WriteBulkString(element)
		}
	}

	// No elements available (simplified: return null instead of blocking)
	return ctx.RespWriter.WriteNull()
}

func (c *BRPopLPushCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "BRPOPLPUSH",
		Summary:      "Pop an element from a list, push it to another list and return it; or block until one is available",
		Syntax:       "BRPOPLPUSH source destination timeout",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *BRPopLPushCommand) ModifiesData() bool { return true }

type LPosCommand struct{}

func (c *LPosCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'lpos' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	elementValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	element, ok := elementValue.AsString()
	if !ok {
		return fmt.Errorf("invalid element")
	}

	// Get the list
	listStorage := ctx.Database.ListStorage
	length := listStorage.LLen(key)
	if length == 0 {
		return ctx.RespWriter.WriteNull()
	}

	// Search for the element
	for i := int64(0); i < length; i++ {
		value, exists := listStorage.LIndex(key, i)
		if exists && value == element {
			return ctx.RespWriter.WriteInteger(i)
		}
	}

	return ctx.RespWriter.WriteNull()
}

func (c *LPosCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LPOS",
		Summary:      "Return the index of matching elements in a list",
		Syntax:       "LPOS key element [RANK rank] [COUNT num-matches] [MAXLEN len]",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *LPosCommand) ModifiesData() bool { return false }

type LMoveCommand struct{}

func (c *LMoveCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 4 {
		return fmt.Errorf("wrong number of arguments for 'lmove' command")
	}

	sourceValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	source, ok := sourceValue.AsString()
	if !ok {
		return fmt.Errorf("invalid source key")
	}

	destValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dest, ok := destValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	whereFromValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	whereFrom, ok := whereFromValue.AsString()
	if !ok {
		return fmt.Errorf("invalid wherefrom argument")
	}

	whereToValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	whereTo, ok := whereToValue.AsString()
	if !ok {
		return fmt.Errorf("invalid whereto argument")
	}

	listStorage := ctx.Database.ListStorage

	// Pop from source
	var element string
	var exists bool
	if whereFrom == "LEFT" {
		element, exists = listStorage.LPop(source)
	} else if whereFrom == "RIGHT" {
		element, exists = listStorage.RPop(source)
	} else {
		return fmt.Errorf("invalid wherefrom argument: %s", whereFrom)
	}

	if !exists {
		return ctx.RespWriter.WriteNull()
	}

	// Push to destination
	if whereTo == "LEFT" {
		_, err = listStorage.LPush(dest, []string{element})
	} else if whereTo == "RIGHT" {
		_, err = listStorage.RPush(dest, []string{element})
	} else {
		return fmt.Errorf("invalid whereto argument: %s", whereTo)
	}

	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteBulkString(element)
}

func (c *LMoveCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LMOVE",
		Summary:      "Pop an element from a list, push it to another list and return it",
		Syntax:       "LMOVE source destination LEFT|RIGHT LEFT|RIGHT",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      4,
		MaxArgs:      4,
		ModifiesData: true,
	}
}

func (c *LMoveCommand) ModifiesData() bool { return true }

type LMPopCommand struct{}

func (c *LMPopCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 {
		return fmt.Errorf("wrong number of arguments for 'lmpop' command")
	}

	numkeysValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	numkeysStr, ok := numkeysValue.AsString()
	if !ok {
		return fmt.Errorf("invalid numkeys argument")
	}

	numkeys, err := strconv.Atoi(numkeysStr)
	if err != nil || numkeys <= 0 {
		return fmt.Errorf("invalid numkeys argument")
	}

	if nargs < 2+numkeys {
		return fmt.Errorf("wrong number of arguments for 'lmpop' command")
	}

	// Read keys
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

	// Read direction (LEFT or RIGHT)
	directionValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	direction, ok := directionValue.AsString()
	if !ok {
		return fmt.Errorf("invalid direction argument")
	}

	if direction != "LEFT" && direction != "RIGHT" {
		return fmt.Errorf("invalid direction: %s", direction)
	}

	// Optional COUNT argument
	count := 1
	if nargs > 2+numkeys {
		if nargs != 4+numkeys {
			return fmt.Errorf("wrong number of arguments for 'lmpop' command")
		}
		
		countKeyword, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		countKeywordStr, ok := countKeyword.AsString()
		if !ok || countKeywordStr != "COUNT" {
			return fmt.Errorf("syntax error")
		}

		countValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		countStr, ok := countValue.AsString()
		if !ok {
			return fmt.Errorf("invalid count argument")
		}

		count, err = strconv.Atoi(countStr)
		if err != nil || count <= 0 {
			return fmt.Errorf("invalid count argument")
		}
	}

	listStorage := ctx.Database.ListStorage

	// Try to pop from each key in order
	for _, key := range keys {
		if listStorage.LLen(key) == 0 {
			continue
		}

		elements := make([]string, 0, count)
		for i := 0; i < count; i++ {
			var element string
			var exists bool
			if direction == "LEFT" {
				element, exists = listStorage.LPop(key)
			} else {
				element, exists = listStorage.RPop(key)
			}

			if !exists {
				break
			}
			elements = append(elements, element)
		}

		if len(elements) > 0 {
			// Convert elements to interface slice
			elementInterfaces := make([]interface{}, len(elements))
			for i, elem := range elements {
				elementInterfaces[i] = elem
			}

			// Return [key, [elements]]
			result := []interface{}{key, elementInterfaces}
			return ctx.RespWriter.WriteArray(result)
		}
	}

	// No elements found in any key
	return ctx.RespWriter.WriteNull()
}

func (c *LMPopCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LMPOP",
		Summary:      "Pop elements from the first non-empty list key from the list of provided key names",
		Syntax:       "LMPOP numkeys key [key ...] LEFT|RIGHT [COUNT count]",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      3,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *LMPopCommand) ModifiesData() bool { return true }

type BLMoveCommand struct{}

func (c *BLMoveCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 5 {
		return fmt.Errorf("wrong number of arguments for 'blmove' command")
	}

	sourceValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	source, ok := sourceValue.AsString()
	if !ok {
		return fmt.Errorf("invalid source key")
	}

	destValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dest, ok := destValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	srcDirectionValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	srcDirection, ok := srcDirectionValue.AsString()
	if !ok {
		return fmt.Errorf("invalid source direction")
	}

	dstDirectionValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dstDirection, ok := dstDirectionValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination direction")
	}

	// Read timeout (ignored in this simplified implementation)
	_, err = ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}

	listStorage := ctx.Database.ListStorage
	// Non-blocking implementation: try once
	if listStorage.LLen(source) > 0 {
		var element string
		var exists bool

		// Pop from source
		if strings.ToUpper(srcDirection) == "LEFT" {
			element, exists = listStorage.LPop(source)
		} else {
			element, exists = listStorage.RPop(source)
		}

		if exists {
			// Push to destination
			if strings.ToUpper(dstDirection) == "LEFT" {
				_, err = listStorage.LPush(dest, []string{element})
			} else {
				_, err = listStorage.RPush(dest, []string{element})
			}
			if err != nil {
				return err
			}
			return ctx.RespWriter.WriteBulkString(element)
		}
	}

	// No elements available (simplified: return null instead of blocking)
	return ctx.RespWriter.WriteNull()
}

func (c *BLMoveCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "BLMOVE",
		Summary:      "Pop an element from a list, push it to another list and return it; or block until one is available",
		Syntax:       "BLMOVE source destination LEFT|RIGHT LEFT|RIGHT timeout",
		Categories:   []engine.CommandCategory{engine.CategoryList},
		MinArgs:      5,
		MaxArgs:      5,
		ModifiesData: true,
	}
}

func (c *BLMoveCommand) ModifiesData() bool { return true }
