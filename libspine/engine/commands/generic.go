package commands

import (
	"fmt"
	"strconv"

	"spine-go/libspine/engine"
)

// RegisterGenericCommands registers all generic commands
func RegisterGenericCommands(registry *engine.CommandRegistry) error {
	commands := []engine.CommandHandler{
		&SortCommand{},
		&SortROCommand{},
		&WaitCommand{},
		&CopyCommand{},
		&DumpCommand{},
		&ExpireTimeCommand{},
		&MigrateCommand{},
		&MoveCommand{},
		&ObjectCommand{},
		&PExpireTimeCommand{},
		&RestoreCommand{},
		&TouchCommand{},
		&UnlinkCommand{},
	}

	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.GetInfo().Name, err)
		}
	}

	return nil
}

// SortCommand implements the SORT command
type SortCommand struct{}

func (c *SortCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 {
		return fmt.Errorf("wrong number of arguments for 'sort' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	// For now, just return empty array (sorting not implemented)
	_ = key
	return ctx.RespWriter.WriteArray([]interface{}{})
}

func (c *SortCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SORT",
		Summary:      "Sort the elements in a list, set or sorted set",
		Syntax:       "SORT key [BY pattern] [LIMIT offset count] [GET pattern [GET pattern ...]] [ASC | DESC] [ALPHA] [STORE destination]",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *SortCommand) ModifiesData() bool {
	return false
}

// SortROCommand implements the SORT_RO command
type SortROCommand struct{}

func (c *SortROCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 {
		return fmt.Errorf("wrong number of arguments for 'sort_ro' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	// For now, just return empty array (sorting not implemented)
	_ = key
	return ctx.RespWriter.WriteArray([]interface{}{})
}

func (c *SortROCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SORT_RO",
		Summary:      "Sort the elements in a list, set or sorted set (read-only variant)",
		Syntax:       "SORT_RO key [BY pattern] [LIMIT offset count] [GET pattern [GET pattern ...]] [ASC | DESC] [ALPHA]",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *SortROCommand) ModifiesData() bool {
	return false
}

// WaitCommand implements the WAIT command
type WaitCommand struct{}

func (c *WaitCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 2 {
		return fmt.Errorf("wrong number of arguments for 'wait' command")
	}

	numReplicasValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	numReplicasStr, ok := numReplicasValue.AsString()
	if !ok {
		return fmt.Errorf("invalid number of replicas")
	}
	numReplicas, err := strconv.Atoi(numReplicasStr)
	if err != nil {
		return fmt.Errorf("invalid number of replicas")
	}

	timeoutValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	timeoutStr, ok := timeoutValue.AsString()
	if !ok {
		return fmt.Errorf("invalid timeout")
	}
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		return fmt.Errorf("invalid timeout")
	}

	// For now, just return 0 (no replication implemented)
	_ = numReplicas
	_ = timeout
	return ctx.RespWriter.WriteInteger(0)
}

func (c *WaitCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "WAIT",
		Summary:      "Wait for the synchronous replication of all the write commands sent in the context of the current connection",
		Syntax:       "WAIT numreplicas timeout",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *WaitCommand) ModifiesData() bool {
	return false
}

// CopyCommand implements the COPY command
type CopyCommand struct{}

func (c *CopyCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'copy' command")
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

	// For now, just return 0 (copy not implemented)
	_ = source
	_ = dest
	return ctx.RespWriter.WriteInteger(0)
}

func (c *CopyCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "COPY",
		Summary:      "Copy a key",
		Syntax:       "COPY source destination [DB destination-db] [REPLACE]",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *CopyCommand) ModifiesData() bool { return true }

// DumpCommand implements the DUMP command
type DumpCommand struct{}

func (c *DumpCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 1 {
		return fmt.Errorf("wrong number of arguments for 'dump' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	// For now, just return null (dump not implemented)
	_ = key
	return ctx.RespWriter.WriteNull()
}

func (c *DumpCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "DUMP",
		Summary:      "Return a serialized version of the value stored at the specified key",
		Syntax:       "DUMP key",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *DumpCommand) ModifiesData() bool { return false }

// ExpireTimeCommand implements the EXPIRETIME command
type ExpireTimeCommand struct{}

func (c *ExpireTimeCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 1 {
		return fmt.Errorf("wrong number of arguments for 'expiretime' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	// For now, just return -1 (no expiration)
	_ = key
	return ctx.RespWriter.WriteInteger(-1)
}

func (c *ExpireTimeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "EXPIRETIME",
		Summary:      "Get the expiration Unix timestamp for a key",
		Syntax:       "EXPIRETIME key",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *ExpireTimeCommand) ModifiesData() bool { return false }

// MigrateCommand implements the MIGRATE command
type MigrateCommand struct{}

func (c *MigrateCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 6 {
		return fmt.Errorf("wrong number of arguments for 'migrate' command")
	}

	// Read all required arguments but don't implement migration
	for i := 0; i < nargs; i++ {
		_, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
	}

	// For now, just return OK (migration not implemented)
	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *MigrateCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "MIGRATE",
		Summary:      "Atomically transfer a key from a Redis instance to another one",
		Syntax:       "MIGRATE host port key|\"\" destination-db timeout [COPY | REPLACE] [KEYS key [key ...]]",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      6,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *MigrateCommand) ModifiesData() bool { return true }

// MoveCommand implements the MOVE command
type MoveCommand struct{}

func (c *MoveCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 2 {
		return fmt.Errorf("wrong number of arguments for 'move' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	dbValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dbStr, ok := dbValue.AsString()
	if !ok {
		return fmt.Errorf("invalid db")
	}
	_, err = strconv.Atoi(dbStr)
	if err != nil {
		return fmt.Errorf("invalid db")
	}

	// For now, just return 0 (move not implemented)
	_ = key
	return ctx.RespWriter.WriteInteger(0)
}

func (c *MoveCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "MOVE",
		Summary:      "Move a key to another database",
		Syntax:       "MOVE key db",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: true,
	}
}

func (c *MoveCommand) ModifiesData() bool { return true }

// ObjectCommand implements the OBJECT command
type ObjectCommand struct{}

func (c *ObjectCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 {
		return fmt.Errorf("wrong number of arguments for 'object' command")
	}

	subcommandValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	subcommand, ok := subcommandValue.AsString()
	if !ok {
		return fmt.Errorf("invalid subcommand")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	// For now, just return null (object inspection not implemented)
	_ = subcommand
	_ = key
	return ctx.RespWriter.WriteNull()
}

func (c *ObjectCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "OBJECT",
		Summary:      "Inspect the internals of Redis objects",
		Syntax:       "OBJECT subcommand [arguments [arguments ...]]",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      2,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *ObjectCommand) ModifiesData() bool { return false }

// PExpireTimeCommand implements the PEXPIRETIME command
type PExpireTimeCommand struct{}

func (c *PExpireTimeCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 1 {
		return fmt.Errorf("wrong number of arguments for 'pexpiretime' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	// For now, just return -1 (no expiration)
	_ = key
	return ctx.RespWriter.WriteInteger(-1)
}

func (c *PExpireTimeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "PEXPIRETIME",
		Summary:      "Get the expiration Unix timestamp for a key in milliseconds",
		Syntax:       "PEXPIRETIME key",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *PExpireTimeCommand) ModifiesData() bool { return false }

// RestoreCommand implements the RESTORE command
type RestoreCommand struct{}

func (c *RestoreCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 {
		return fmt.Errorf("wrong number of arguments for 'restore' command")
	}

	// Read all arguments but don't implement restore
	for i := 0; i < nargs; i++ {
		_, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
	}

	// For now, just return OK (restore not implemented)
	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *RestoreCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "RESTORE",
		Summary:      "Create a key using the provided serialized value, previously obtained using DUMP",
		Syntax:       "RESTORE key ttl serialized-value [REPLACE] [ABSTTL] [IDLETIME seconds] [FREQ frequency]",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      3,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *RestoreCommand) ModifiesData() bool { return true }

// TouchCommand implements the TOUCH command
type TouchCommand struct{}

func (c *TouchCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 {
		return fmt.Errorf("wrong number of arguments for 'touch' command")
	}

	count := int64(0)
	for i := 0; i < nargs; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		
		// Check if key exists
		if ctx.Database.CommonStorage.Exists(key) {
			count++
		}
	}

	return ctx.RespWriter.WriteInteger(count)
}

func (c *TouchCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "TOUCH",
		Summary:      "Alters the last access time of a key(s). Returns the number of existing keys specified",
		Syntax:       "TOUCH key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *TouchCommand) ModifiesData() bool { return false }

// UnlinkCommand implements the UNLINK command
type UnlinkCommand struct{}

func (c *UnlinkCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 {
		return fmt.Errorf("wrong number of arguments for 'unlink' command")
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

	// Use the same deletion logic as DEL command
	count := ctx.Database.CommonStorage.Del(keys)
	return ctx.RespWriter.WriteInteger(count)
}

func (c *UnlinkCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "UNLINK",
		Summary:      "Delete a key asynchronously in another thread. Otherwise it is just as DEL, but non blocking",
		Syntax:       "UNLINK key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *UnlinkCommand) ModifiesData() bool { return true }
