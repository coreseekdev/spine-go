package commands

import (
	"fmt"
	"strconv"

	"spine-go/libspine/engine"
	"spine-go/libspine/engine/storage"
)

// RegisterSetCommands registers all set-related commands
func RegisterSetCommands(registry *engine.CommandRegistry) error {
	commands := []engine.CommandHandler{
		&SAddCommand{},
		&SRemCommand{},
		&SMembersCommand{},
		&SIsMemberCommand{},
		&SCardCommand{},
		&SPopCommand{},
		&SRandMemberCommand{},
		&SInterCommand{},
		&SInterStoreCommand{},
		&SUnionCommand{},
		&SUnionStoreCommand{},
		&SDiffCommand{},
		&SDiffStoreCommand{},
		&SMoveCommand{},
		&SScanCommand{},
		&SMIsMemberCommand{},
	}

	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.GetInfo().Name, err)
		}
	}

	return nil
}

// SAddCommand implements the SADD command
type SAddCommand struct{}

func (c *SAddCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'sadd' command")
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

	// Read members
	var members []string
	for i := 1; i < nargs; i++ {
		memberValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid member")
		}
		member, ok := memberValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid member")
		}
		members = append(members, member)
	}

	// Use set storage from context
	setStorage := ctx.Database.SetStorage
	addedCount, err := setStorage.SAdd(key, members)
	if err != nil {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return ctx.RespWriter.WriteInteger(int64(addedCount))
}

func (c *SAddCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SADD",
		Summary:      "Add one or more members to a set",
		Syntax:       "SADD key member [member ...]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *SAddCommand) ModifiesData() bool {
	return true
}

// SRemCommand implements the SREM command
type SRemCommand struct{}

func (c *SRemCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'srem' command")
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

	// Read members
	var members []string
	for i := 1; i < nargs; i++ {
		memberValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid member")
		}
		member, ok := memberValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid member")
		}
		members = append(members, member)
	}

	// Use set storage from context
	setStorage := ctx.Database.SetStorage
	removedCount, err := setStorage.SRem(key, members)
	if err != nil {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return ctx.RespWriter.WriteInteger(int64(removedCount))
}

func (c *SRemCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SREM",
		Summary:      "Remove one or more members from a set",
		Syntax:       "SREM key member [member ...]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *SRemCommand) ModifiesData() bool {
	return true
}

// SMembersCommand implements the SMEMBERS command
type SMembersCommand struct{}

func (c *SMembersCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'smembers' command")
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

	// Use set storage from context
	setStorage := ctx.Database.SetStorage
	members := setStorage.SMembers(key)

	// Build result array
	result := make([]interface{}, len(members))
	for i, member := range members {
		result[i] = member
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *SMembersCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SMEMBERS",
		Summary:      "Get all the members in a set",
		Syntax:       "SMEMBERS key",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *SMembersCommand) ModifiesData() bool {
	return false
}

// SIsMemberCommand implements the SISMEMBER command
type SIsMemberCommand struct{}

func (c *SIsMemberCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'sismember' command")
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

	// Get set
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteInteger(0)
	}

	if value.Type != storage.TypeSet {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	setData, ok := value.Data.(map[string]bool)
	if !ok {
		return ctx.RespWriter.WriteInteger(0)
	}

	if setData[member] {
		return ctx.RespWriter.WriteInteger(1)
	}

	return ctx.RespWriter.WriteInteger(0)
}

func (c *SIsMemberCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SISMEMBER",
		Summary:      "Determine if a given value is a member of a set",
		Syntax:       "SISMEMBER key member",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *SIsMemberCommand) ModifiesData() bool {
	return false
}

// SCardCommand implements the SCARD command
type SCardCommand struct{}

func (c *SCardCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'scard' command")
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

	// Get set
	value, exists := db.GetValue(key)
	if !exists {
		return ctx.RespWriter.WriteInteger(0)
	}

	if value.Type != storage.TypeSet {
		return ctx.RespWriter.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	setData, ok := value.Data.(map[string]bool)
	if !ok {
		return ctx.RespWriter.WriteInteger(0)
	}

	return ctx.RespWriter.WriteInteger(int64(len(setData)))
}

func (c *SCardCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SCARD",
		Summary:      "Get the number of members in a set",
		Syntax:       "SCARD key",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *SCardCommand) ModifiesData() bool {
	return false
}

// Placeholder implementations for remaining set commands
type SPopCommand struct{}

func (c *SPopCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 || nargs > 2 {
		return fmt.Errorf("wrong number of arguments for 'spop' command")
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

	setStorage := ctx.Database.SetStorage
	members := setStorage.SPop(key, count)

	if nargs == 1 {
		// Single element case
		if len(members) > 0 {
			return ctx.RespWriter.WriteBulkString(members[0])
		} else {
			return ctx.RespWriter.WriteNull()
		}
	} else {
		// Multiple elements case - return array
		result := make([]interface{}, len(members))
		for i, member := range members {
			result[i] = member
		}
		return ctx.RespWriter.WriteArray(result)
	}
}

func (c *SPopCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SPOP",
		Summary:      "Remove and return one or multiple random members from a set",
		Syntax:       "SPOP key [count]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      1,
		MaxArgs:      2,
		ModifiesData: true,
	}
}

func (c *SPopCommand) ModifiesData() bool { return true }

type SRandMemberCommand struct{}

func (c *SRandMemberCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 || nargs > 2 {
		return fmt.Errorf("wrong number of arguments for 'srandmember' command")
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
		if err != nil {
			return fmt.Errorf("invalid count")
		}
	}

	setStorage := ctx.Database.SetStorage
	members := setStorage.SRandMember(key, count)

	if nargs == 1 {
		// Single element case
		if len(members) > 0 {
			return ctx.RespWriter.WriteBulkString(members[0])
		} else {
			return ctx.RespWriter.WriteNull()
		}
	} else {
		// Multiple elements case - return array
		result := make([]interface{}, len(members))
		for i, member := range members {
			result[i] = member
		}
		return ctx.RespWriter.WriteArray(result)
	}
}

func (c *SRandMemberCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SRANDMEMBER",
		Summary:      "Get one or multiple random members from a set",
		Syntax:       "SRANDMEMBER key [count]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      1,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *SRandMemberCommand) ModifiesData() bool { return false }

type SInterCommand struct{}

func (c *SInterCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 {
		return fmt.Errorf("wrong number of arguments for 'sinter' command")
	}

	keys := make([]string, nargs)
	for i := 0; i < nargs; i++ {
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

	setStorage := ctx.Database.SetStorage
	if len(keys) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Start with first set
	result := setStorage.SMembers(keys[0])
	if len(result) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Intersect with remaining sets
	for i := 1; i < len(keys); i++ {
		setMembers := setStorage.SMembers(keys[i])
		if len(setMembers) == 0 {
			return ctx.RespWriter.WriteArray([]interface{}{})
		}

		// Create map for faster lookup
		memberMap := make(map[string]bool)
		for _, member := range setMembers {
			memberMap[member] = true
		}

		// Filter result to only include members in current set
		filtered := make([]string, 0)
		for _, member := range result {
			if memberMap[member] {
				filtered = append(filtered, member)
			}
		}
		result = filtered

		if len(result) == 0 {
			break
		}
	}

	// Convert to interface slice
	interfaces := make([]interface{}, len(result))
	for i, member := range result {
		interfaces[i] = member
	}

	return ctx.RespWriter.WriteArray(interfaces)
}

func (c *SInterCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SINTER",
		Summary:      "Intersect multiple sets",
		Syntax:       "SINTER key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *SInterCommand) ModifiesData() bool { return false }

type SInterStoreCommand struct{}

func (c *SInterStoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'sinterstore' command")
	}

	destValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dest, ok := destValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	keys := make([]string, nargs-1)
	for i := 1; i < nargs; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i-1] = key
	}

	setStorage := ctx.Database.SetStorage
	if len(keys) == 0 {
		setStorage.SRem(dest, setStorage.SMembers(dest))
		return ctx.RespWriter.WriteInteger(0)
	}

	// Start with first set
	result := setStorage.SMembers(keys[0])
	if len(result) == 0 {
		setStorage.SRem(dest, setStorage.SMembers(dest))
		return ctx.RespWriter.WriteInteger(0)
	}

	// Intersect with remaining sets
	for i := 1; i < len(keys); i++ {
		setMembers := setStorage.SMembers(keys[i])
		if len(setMembers) == 0 {
			setStorage.SRem(dest, setStorage.SMembers(dest))
			return ctx.RespWriter.WriteInteger(0)
		}

		// Create map for faster lookup
		memberMap := make(map[string]bool)
		for _, member := range setMembers {
			memberMap[member] = true
		}

		// Filter result to only include members in current set
		filtered := make([]string, 0)
		for _, member := range result {
			if memberMap[member] {
				filtered = append(filtered, member)
			}
		}
		result = filtered

		if len(result) == 0 {
			break
		}
	}

	// Clear destination and store result
	setStorage.SRem(dest, setStorage.SMembers(dest))
	if len(result) > 0 {
		setStorage.SAdd(dest, result)
	}

	return ctx.RespWriter.WriteInteger(int64(len(result)))
}

func (c *SInterStoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SINTERSTORE",
		Summary:      "Intersect multiple sets and store the resulting set in a key",
		Syntax:       "SINTERSTORE destination key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *SInterStoreCommand) ModifiesData() bool { return true }

type SUnionCommand struct{}

func (c *SUnionCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 {
		return fmt.Errorf("wrong number of arguments for 'sunion' command")
	}

	keys := make([]string, nargs)
	for i := 0; i < nargs; i++ {
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

	setStorage := ctx.Database.SetStorage
	unionMap := make(map[string]bool)

	// Union all sets
	for _, key := range keys {
		members := setStorage.SMembers(key)
		for _, member := range members {
			unionMap[member] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(unionMap))
	for member := range unionMap {
		result = append(result, member)
	}

	// Convert to interface slice
	interfaces := make([]interface{}, len(result))
	for i, member := range result {
		interfaces[i] = member
	}

	return ctx.RespWriter.WriteArray(interfaces)
}

func (c *SUnionCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SUNION",
		Summary:      "Add multiple sets",
		Syntax:       "SUNION key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *SUnionCommand) ModifiesData() bool { return false }

type SUnionStoreCommand struct{}

func (c *SUnionStoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'sunionstore' command")
	}

	destValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dest, ok := destValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	keys := make([]string, nargs-1)
	for i := 1; i < nargs; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i-1] = key
	}

	setStorage := ctx.Database.SetStorage
	unionMap := make(map[string]bool)

	// Union all sets
	for _, key := range keys {
		members := setStorage.SMembers(key)
		for _, member := range members {
			unionMap[member] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(unionMap))
	for member := range unionMap {
		result = append(result, member)
	}

	// Clear destination and store result
	setStorage.SRem(dest, setStorage.SMembers(dest))
	if len(result) > 0 {
		setStorage.SAdd(dest, result)
	}

	return ctx.RespWriter.WriteInteger(int64(len(result)))
}

func (c *SUnionStoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SUNIONSTORE",
		Summary:      "Add multiple sets and store the resulting set in a key",
		Syntax:       "SUNIONSTORE destination key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *SUnionStoreCommand) ModifiesData() bool { return true }

type SDiffCommand struct{}

func (c *SDiffCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 {
		return fmt.Errorf("wrong number of arguments for 'sdiff' command")
	}

	keys := make([]string, nargs)
	for i := 0; i < nargs; i++ {
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

	setStorage := ctx.Database.SetStorage
	if len(keys) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Start with first set
	result := setStorage.SMembers(keys[0])
	if len(result) == 0 {
		return ctx.RespWriter.WriteArray([]interface{}{})
	}

	// Remove members that exist in other sets
	for i := 1; i < len(keys); i++ {
		setMembers := setStorage.SMembers(keys[i])
		if len(setMembers) == 0 {
			continue
		}

		// Create map for faster lookup
		memberMap := make(map[string]bool)
		for _, member := range setMembers {
			memberMap[member] = true
		}

		// Filter result to exclude members in current set
		filtered := make([]string, 0)
		for _, member := range result {
			if !memberMap[member] {
				filtered = append(filtered, member)
			}
		}
		result = filtered
	}

	// Convert to interface slice
	interfaces := make([]interface{}, len(result))
	for i, member := range result {
		interfaces[i] = member
	}

	return ctx.RespWriter.WriteArray(interfaces)
}

func (c *SDiffCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SDIFF",
		Summary:      "Subtract multiple sets",
		Syntax:       "SDIFF key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *SDiffCommand) ModifiesData() bool { return false }

type SDiffStoreCommand struct{}

func (c *SDiffStoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'sdiffstore' command")
	}

	destValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dest, ok := destValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	keys := make([]string, nargs-1)
	for i := 1; i < nargs; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i-1] = key
	}

	setStorage := ctx.Database.SetStorage
	if len(keys) == 0 {
		setStorage.SRem(dest, setStorage.SMembers(dest))
		return ctx.RespWriter.WriteInteger(0)
	}

	// Start with first set
	result := setStorage.SMembers(keys[0])
	if len(result) == 0 {
		setStorage.SRem(dest, setStorage.SMembers(dest))
		return ctx.RespWriter.WriteInteger(0)
	}

	// Remove members that exist in other sets
	for i := 1; i < len(keys); i++ {
		setMembers := setStorage.SMembers(keys[i])
		if len(setMembers) == 0 {
			continue
		}

		// Create map for faster lookup
		memberMap := make(map[string]bool)
		for _, member := range setMembers {
			memberMap[member] = true
		}

		// Filter result to exclude members in current set
		filtered := make([]string, 0)
		for _, member := range result {
			if !memberMap[member] {
				filtered = append(filtered, member)
			}
		}
		result = filtered
	}

	// Clear destination and store result
	setStorage.SRem(dest, setStorage.SMembers(dest))
	if len(result) > 0 {
		setStorage.SAdd(dest, result)
	}

	return ctx.RespWriter.WriteInteger(int64(len(result)))
}

func (c *SDiffStoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SDIFFSTORE",
		Summary:      "Subtract multiple sets and store the resulting set in a key",
		Syntax:       "SDIFFSTORE destination key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *SDiffStoreCommand) ModifiesData() bool { return true }

type SMoveCommand struct{}

func (c *SMoveCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'smove' command")
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

	memberValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	member, ok := memberValue.AsString()
	if !ok {
		return fmt.Errorf("invalid member")
	}

	setStorage := ctx.Database.SetStorage

	// Check if member exists in source set
	if !setStorage.SIsMember(source, member) {
		return ctx.RespWriter.WriteInteger(0) // Member not in source
	}

	// Remove from source
	setStorage.SRem(source, []string{member})

	// Add to destination
	setStorage.SAdd(dest, []string{member})

	return ctx.RespWriter.WriteInteger(1) // Successfully moved
}

func (c *SMoveCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SMOVE",
		Summary:      "Move a member from one set to another",
		Syntax:       "SMOVE source destination member",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *SMoveCommand) ModifiesData() bool { return true }

type SScanCommand struct{}

func (c *SScanCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'sscan' command")
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

	// For simplicity, ignore cursor and return all members
	setStorage := ctx.Database.SetStorage
	members := setStorage.SMembers(key)

	// Convert to interface slice
	result := make([]interface{}, len(members))
	for i, member := range members {
		result[i] = member
	}

	// Return cursor 0 (end of iteration) and the members
	response := []interface{}{"0", result}
	return ctx.RespWriter.WriteArray(response)
}

func (c *SScanCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SSCAN",
		Summary:      "Incrementally iterate Set elements",
		Syntax:       "SSCAN key cursor [MATCH pattern] [COUNT count]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *SScanCommand) ModifiesData() bool { return false }

type SMIsMemberCommand struct{}

func (c *SMIsMemberCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'smismember' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	setStorage := ctx.Database.SetStorage
	results := make([]interface{}, 0, nargs-1)

	// Check membership for each member
	for i := 1; i < nargs; i++ {
		memberValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		member, ok := memberValue.AsString()
		if !ok {
			return fmt.Errorf("invalid member")
		}

		if setStorage.SIsMember(key, member) {
			results = append(results, int64(1))
		} else {
			results = append(results, int64(0))
		}
	}

	return ctx.RespWriter.WriteArray(results)
}

func (c *SMIsMemberCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SMISMEMBER",
		Summary:      "Returns the membership associated with the given elements for a set",
		Syntax:       "SMISMEMBER key member [member ...]",
		Categories:   []engine.CommandCategory{engine.CategorySet},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *SMIsMemberCommand) ModifiesData() bool { return false }
