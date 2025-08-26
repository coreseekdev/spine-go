package commands

import (
	"fmt"
	"strconv"

	"spine-go/libspine/engine"
	"spine-go/libspine/engine/storage"
)

// RegisterHashCommands registers all hash-related commands
func RegisterHashCommands(registry *engine.CommandRegistry) error {
	commands := []engine.CommandHandler{
		&HSetCommand{},
		&HGetCommand{},
		&HMSetCommand{},
		&HMGetCommand{},
		&HGetAllCommand{},
		&HDelCommand{},
		&HExistsCommand{},
		&HKeysCommand{},
		&HValsCommand{},
		&HLenCommand{},
		&HSetNXCommand{},
		&HIncrByCommand{},
		&HIncrByFloatCommand{},
		&HScanCommand{},
		&HStrLenCommand{},
		&HRandFieldCommand{},
	}

	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.GetInfo().Name, err)
		}
	}

	return nil
}

// HSetCommand implements the HSET command
type HSetCommand struct{}

func (c *HSetCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 3 || nargs%2 == 0 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'hset' command")
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

	// Use hash storage from context
	hashStorage := ctx.Database.HashStorage

	// Count new fields
	newFields := 0

	// Read field-value pairs
	for i := 1; i < nargs; i += 2 {
		fieldValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid field")
		}
		field, ok := fieldValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid field")
		}

		valueValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid value")
		}
		value, ok := valueValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid value")
		}

		// Set field and check if it's new
		isNewField, err := hashStorage.HSet(key, field, value)
		if err != nil {
			return ctx.RespWriter.WriteError("ERR " + err.Error())
		}
		if isNewField {
			newFields++
		}
	}

	return ctx.RespWriter.WriteInteger(int64(newFields))
}

func (c *HSetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HSET",
		Summary:      "Set the string value of a hash field",
		Syntax:       "HSET key field value [field value ...]",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      3,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *HSetCommand) ModifiesData() bool {
	return true
}

// HGetCommand implements the HGET command
type HGetCommand struct{}

func (c *HGetCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'hget' command")
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

	// Read field
	fieldValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid field")
	}
	field, ok := fieldValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid field")
	}

	// Use hash storage from context
	hashStorage := ctx.Database.HashStorage

	// Get field value
	value, exists := hashStorage.HGet(key, field)
	if !exists {
		return ctx.RespWriter.WriteNull()
	}

	return ctx.RespWriter.WriteBulkString(value)
}

func (c *HGetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HGET",
		Summary:      "Get the value of a hash field",
		Syntax:       "HGET key field",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *HGetCommand) ModifiesData() bool {
	return false
}

// HMSetCommand implements the HMSET command
type HMSetCommand struct{}

func (c *HMSetCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 3 || nargs%2 == 0 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'hmset' command")
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

	// Use hash storage from context
	hashStorage := ctx.Database.HashStorage

	// Collect field-value pairs
	fields := make(map[string]string)
	for i := 1; i < nargs; i += 2 {
		fieldValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid field")
		}
		field, ok := fieldValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid field")
		}

		valueValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid value")
		}
		value, ok := valueValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid value")
		}

		fields[field] = value
	}

	// Set all fields
	if err := hashStorage.HMSet(key, fields); err != nil {
		return ctx.RespWriter.WriteError("ERR " + err.Error())
	}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *HMSetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HMSET",
		Summary:      "Set multiple hash fields to multiple values",
		Syntax:       "HMSET key field value [field value ...]",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      3,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *HMSetCommand) ModifiesData() bool {
	return true
}

// HMGetCommand implements the HMGET command
type HMGetCommand struct{}

func (c *HMGetCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'hmget' command")
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

	// Get hash
	var hashData map[string]string
	if value, exists := db.GetValue(key); exists {
		if value.Type != storage.TypeHash {
			return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
		if h, ok := value.Data.(map[string]string); ok {
			hashData = h
		}
	}

	// Read fields and get values
	values := make([]interface{}, nargs-1)
	for i := 1; i < nargs; i++ {
		fieldValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid field")
		}
		field, ok := fieldValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid field")
		}

		if hashData != nil {
			if value, exists := hashData[field]; exists {
				values[i-1] = value
			} else {
				values[i-1] = nil
			}
		} else {
			values[i-1] = nil
		}
	}

	return ctx.RespWriter.WriteArray(values)
}

func (c *HMGetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HMGET",
		Summary:      "Get the values of all the given hash fields",
		Syntax:       "HMGET key field [field ...]",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *HMGetCommand) ModifiesData() bool {
	return false
}

// HGetAllCommand implements the HGETALL command
type HGetAllCommand struct{}

func (c *HGetAllCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'hgetall' command")
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

	// Get hash
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	if value.Type != storage.TypeHash {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	hashData, ok := value.Data.(map[string]string)
	if !ok {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Build result array with field-value pairs
	result := make([]interface{}, 0, len(hashData)*2)
	for field, value := range hashData {
		result = append(result, field, value)
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *HGetAllCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HGETALL",
		Summary:      "Get all the fields and values in a hash",
		Syntax:       "HGETALL key",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *HGetAllCommand) ModifiesData() bool {
	return false
}

// HDelCommand implements the HDEL command
type HDelCommand struct{}

func (c *HDelCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'hdel' command")
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

	// Get hash
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteInteger(0)
	}

	if value.Type != storage.TypeHash {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	hashData, ok := value.Data.(map[string]string)
	if !ok {
		return ctx.RespWriter.WriteInteger(0)
	}

	// Delete fields
	deletedCount := 0
	for i := 1; i < nargs; i++ {
		fieldValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid field")
		}
		field, ok := fieldValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid field")
		}

		if _, exists := hashData[field]; exists {
			delete(hashData, field)
			deletedCount++
		}
	}

	// Update or delete the hash
	if len(hashData) == 0 {
		db.Del(key)
	} else {
		value.Data = hashData
		db.SetValue(key, value)
	}

	return ctx.RespWriter.WriteInteger(int64(deletedCount))
}

func (c *HDelCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HDEL",
		Summary:      "Delete one or more hash fields",
		Syntax:       "HDEL key field [field ...]",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *HDelCommand) ModifiesData() bool {
	return true
}

// HExistsCommand implements the HEXISTS command
type HExistsCommand struct{}

func (c *HExistsCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'hexists' command")
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

	// Read field
	fieldValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid field")
	}
	field, ok := fieldValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid field")
	}

	// Get database
	db := ctx.Database

	// Get hash
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteInteger(0)
	}

	if value.Type != storage.TypeHash {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	hashData, ok := value.Data.(map[string]string)
	if !ok {
		return ctx.RespWriter.WriteInteger(0)
	}

	if _, exists := hashData[field]; exists {
		return ctx.RespWriter.WriteInteger(1)
	}

	return ctx.RespWriter.WriteInteger(0)
}

func (c *HExistsCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HEXISTS",
		Summary:      "Determine if a hash field exists",
		Syntax:       "HEXISTS key field",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *HExistsCommand) ModifiesData() bool {
	return false
}

// HLenCommand implements the HLEN command
type HLenCommand struct{}

func (c *HLenCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'hlen' command")
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

	// Get hash
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteInteger(0)
	}

	if value.Type != storage.TypeHash {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	hashData, ok := value.Data.(map[string]string)
	if !ok {
		return ctx.RespWriter.WriteInteger(0)
	}

	return ctx.RespWriter.WriteInteger(int64(len(hashData)))
}

func (c *HLenCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HLEN",
		Summary:      "Get the number of fields in a hash",
		Syntax:       "HLEN key",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *HLenCommand) ModifiesData() bool {
	return false
}

type HKeysCommand struct{}

func (c *HKeysCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 1 {
		return fmt.Errorf("wrong number of arguments for 'hkeys' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	hashStorage := ctx.Database.HashStorage
	fields := hashStorage.HKeys(key)

	// Convert to interface slice
	result := make([]interface{}, len(fields))
	for i, field := range fields {
		result[i] = field
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *HKeysCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HKEYS",
		Summary:      "Get all the fields in a hash",
		Syntax:       "HKEYS key",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *HKeysCommand) ModifiesData() bool { return false }

type HValsCommand struct{}

func (c *HValsCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 1 {
		return fmt.Errorf("wrong number of arguments for 'hvals' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	hashStorage := ctx.Database.HashStorage
	values := hashStorage.HVals(key)

	// Convert to interface slice
	result := make([]interface{}, len(values))
	for i, value := range values {
		result[i] = value
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *HValsCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HVALS",
		Summary:      "Get all the values in a hash",
		Syntax:       "HVALS key",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *HValsCommand) ModifiesData() bool { return false }

type HSetNXCommand struct{}

func (c *HSetNXCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'hsetnx' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	fieldValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	field, ok := fieldValue.AsString()
	if !ok {
		return fmt.Errorf("invalid field")
	}

	valueValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	value, ok := valueValue.AsString()
	if !ok {
		return fmt.Errorf("invalid value")
	}

	hashStorage := ctx.Database.HashStorage

	// Check if field already exists
	if hashStorage.HExists(key, field) {
		return ctx.RespWriter.WriteInteger(0) // Field exists, no operation
	}

	// Set the field since it doesn't exist
	_, err = hashStorage.HSet(key, field, value)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteInteger(1) // Field was set
}

func (c *HSetNXCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HSETNX",
		Summary:      "Set the value of a hash field, only if the field does not exist",
		Syntax:       "HSETNX key field value",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *HSetNXCommand) ModifiesData() bool { return true }

// Placeholder implementations for remaining hash commands
type HIncrByCommand struct{}

func (c *HIncrByCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'hincrby' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	fieldValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	field, ok := fieldValue.AsString()
	if !ok {
		return fmt.Errorf("invalid field")
	}

	incrValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	incrStr, ok := incrValue.AsString()
	if !ok {
		return fmt.Errorf("invalid increment")
	}

	increment, err := strconv.ParseInt(incrStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid increment value")
	}

	hashStorage := ctx.Database.HashStorage

	// Get current value
	currentStr, exists := hashStorage.HGet(key, field)
	var current int64 = 0
	if exists {
		current, err = strconv.ParseInt(currentStr, 10, 64)
		if err != nil {
			return fmt.Errorf("hash value is not an integer")
		}
	}

	// Calculate new value
	newValue := current + increment

	// Set new value
	_, err = hashStorage.HSet(key, field, strconv.FormatInt(newValue, 10))
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteInteger(newValue)
}

func (c *HIncrByCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HINCRBY",
		Summary:      "Increment the integer value of a hash field by the given number",
		Syntax:       "HINCRBY key field increment",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *HIncrByCommand) ModifiesData() bool { return true }

type HIncrByFloatCommand struct{}

func (c *HIncrByFloatCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'hincrbyfloat' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	fieldValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	field, ok := fieldValue.AsString()
	if !ok {
		return fmt.Errorf("invalid field")
	}

	incrValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	incrStr, ok := incrValue.AsString()
	if !ok {
		return fmt.Errorf("invalid increment")
	}

	increment, err := strconv.ParseFloat(incrStr, 64)
	if err != nil {
		return fmt.Errorf("invalid increment value")
	}

	hashStorage := ctx.Database.HashStorage

	// Get current value
	currentStr, exists := hashStorage.HGet(key, field)
	var current float64 = 0.0
	if exists {
		current, err = strconv.ParseFloat(currentStr, 64)
		if err != nil {
			return fmt.Errorf("hash value is not a valid float")
		}
	}

	// Calculate new value
	newValue := current + increment

	// Set new value
	_, err = hashStorage.HSet(key, field, strconv.FormatFloat(newValue, 'g', -1, 64))
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteBulkString(strconv.FormatFloat(newValue, 'g', -1, 64))
}

func (c *HIncrByFloatCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HINCRBYFLOAT",
		Summary:      "Increment the float value of a hash field by the given amount",
		Syntax:       "HINCRBYFLOAT key field increment",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *HIncrByFloatCommand) ModifiesData() bool { return true }

type HScanCommand struct{}

func (c *HScanCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'hscan' command")
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
	_, ok = cursorValue.AsString()
	if !ok {
		return fmt.Errorf("invalid cursor")
	}

	// For simplicity, ignore cursor and return all fields
	fields, err := ctx.Database.HashStorage.HGetAll(key)
	if err != nil {
		return err
	}

	// Convert to array format for SCAN response
	result := make([]interface{}, 0, len(fields)*2)
	for field, value := range fields {
		result = append(result, field, value)
	}

	// Return cursor 0 (end of iteration) and the fields
	response := []interface{}{"0", result}
	return ctx.RespWriter.WriteArray(response)
}

func (c *HScanCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HSCAN",
		Summary:      "Incrementally iterate hash fields and associated values",
		Syntax:       "HSCAN key cursor [MATCH pattern] [COUNT count]",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *HScanCommand) ModifiesData() bool { return false }

type HStrLenCommand struct{}

func (c *HStrLenCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 2 {
		return fmt.Errorf("wrong number of arguments for 'hstrlen' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	fieldValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	field, ok := fieldValue.AsString()
	if !ok {
		return fmt.Errorf("invalid field")
	}

	hashStorage := ctx.Database.HashStorage
	value, exists := hashStorage.HGet(key, field)
	if !exists {
		return ctx.RespWriter.WriteInteger(0)
	}

	return ctx.RespWriter.WriteInteger(int64(len(value)))
}
func (c *HStrLenCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{Name: "HSTRLEN", Categories: []engine.CommandCategory{engine.CategoryHash}}
}
func (c *HStrLenCommand) ModifiesData() bool { return false }

type HRandFieldCommand struct{}

func (c *HRandFieldCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 || nargs > 3 {
		return fmt.Errorf("wrong number of arguments for 'hrandfield' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	count := 1
	withValues := false

	if nargs >= 2 {
		countValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		countStr, ok := countValue.AsString()
		if !ok {
			return fmt.Errorf("invalid count")
		}

		// Parse count (simplified - just use 1 for now)
		_ = countStr
		count = 1
	}

	if nargs == 3 {
		withValuesValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		withValuesStr, ok := withValuesValue.AsString()
		if !ok {
			return fmt.Errorf("invalid withvalues argument")
		}
		if withValuesStr == "WITHVALUES" {
			withValues = true
		}
	}

	// Get all fields from hash
	fields, err := ctx.Database.HashStorage.HGetAll(key)
	if err != nil {
		return err
	}

	if len(fields) == 0 {
		if count == 1 {
			return ctx.RespWriter.WriteNull()
		}
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// For simplicity, just return the first field
	for field, value := range fields {
		if withValues {
			return ctx.RespWriter.WriteArray([]interface{}{field, value})
		} else {
			return ctx.RespWriter.WriteBulkString(field)
		}
	}

	return ctx.RespWriter.WriteNull()
}

func (c *HRandFieldCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HRANDFIELD",
		Summary:      "Get one or multiple random fields from a hash",
		Syntax:       "HRANDFIELD key [count [WITHVALUES]]",
		Categories:   []engine.CommandCategory{engine.CategoryHash},
		MinArgs:      1,
		MaxArgs:      3,
		ModifiesData: false,
	}
}

func (c *HRandFieldCommand) ModifiesData() bool { return false }
