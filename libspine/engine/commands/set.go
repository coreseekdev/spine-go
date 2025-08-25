package commands

import (
	"fmt"

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
func (c *SPopCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SPopCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SPOP", Categories: []engine.CommandCategory{engine.CategorySet}} }
func (c *SPopCommand) ModifiesData() bool { return true }

type SRandMemberCommand struct{}
func (c *SRandMemberCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SRandMemberCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SRANDMEMBER", Categories: []engine.CommandCategory{engine.CategorySet}} }
func (c *SRandMemberCommand) ModifiesData() bool { return false }

type SInterCommand struct{}
func (c *SInterCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SInterCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SINTER", Categories: []engine.CommandCategory{engine.CategorySet}} }
func (c *SInterCommand) ModifiesData() bool { return false }

type SInterStoreCommand struct{}
func (c *SInterStoreCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SInterStoreCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SINTERSTORE", Categories: []engine.CommandCategory{engine.CategorySet}} }
func (c *SInterStoreCommand) ModifiesData() bool { return true }

type SUnionCommand struct{}
func (c *SUnionCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SUnionCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SUNION", Categories: []engine.CommandCategory{engine.CategorySet}} }
func (c *SUnionCommand) ModifiesData() bool { return false }

type SUnionStoreCommand struct{}
func (c *SUnionStoreCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SUnionStoreCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SUNIONSTORE", Categories: []engine.CommandCategory{engine.CategorySet}} }
func (c *SUnionStoreCommand) ModifiesData() bool { return true }

type SDiffCommand struct{}
func (c *SDiffCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SDiffCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SDIFF", Categories: []engine.CommandCategory{engine.CategorySet}} }
func (c *SDiffCommand) ModifiesData() bool { return false }

type SDiffStoreCommand struct{}
func (c *SDiffStoreCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SDiffStoreCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SDIFFSTORE", Categories: []engine.CommandCategory{engine.CategorySet}} }
func (c *SDiffStoreCommand) ModifiesData() bool { return true }

type SMoveCommand struct{}
func (c *SMoveCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SMoveCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SMOVE", Categories: []engine.CommandCategory{engine.CategorySet}} }
func (c *SMoveCommand) ModifiesData() bool { return true }

type SScanCommand struct{}
func (c *SScanCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SScanCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SSCAN", Categories: []engine.CommandCategory{engine.CategorySet}} }
func (c *SScanCommand) ModifiesData() bool { return false }

type SMIsMemberCommand struct{}
func (c *SMIsMemberCommand) Execute(ctx *engine.CommandContext) error { return ctx.RespWriter.WriteError("ERR not implemented") }
func (c *SMIsMemberCommand) GetInfo() *engine.CommandInfo { return &engine.CommandInfo{Name: "SMISMEMBER", Categories: []engine.CommandCategory{engine.CategorySet}} }
func (c *SMIsMemberCommand) ModifiesData() bool { return false }
