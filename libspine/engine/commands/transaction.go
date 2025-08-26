package commands

import (
	"fmt"

	"spine-go/libspine/engine"
)

// RegisterTransactionCommands registers all transaction-related commands
func RegisterTransactionCommands(registry *engine.CommandRegistry) error {
	commands := []engine.CommandHandler{
		&MultiCommand{},
		&ExecCommand{},
		&DiscardCommand{},
		&WatchCommand{},
		&UnwatchCommand{},
	}

	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.GetInfo().Name, err)
		}
	}

	return nil
}

// MultiCommand implements the MULTI command
type MultiCommand struct{}

func (c *MultiCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 0 {
		return fmt.Errorf("wrong number of arguments for 'multi' command")
	}

	// Set transaction state in connection metadata
	if ctx.TransportCtx.ConnInfo.Metadata == nil {
		ctx.TransportCtx.ConnInfo.Metadata = make(map[string]interface{})
	}
	ctx.TransportCtx.ConnInfo.Metadata["transaction_state"] = "multi"
	ctx.TransportCtx.ConnInfo.Metadata["transaction_queue"] = []interface{}{}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *MultiCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "MULTI",
		Summary:      "Mark the start of a transaction block",
		Syntax:       "MULTI",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      0,
		MaxArgs:      0,
		ModifiesData: false,
	}
}

func (c *MultiCommand) ModifiesData() bool {
	return false
}

// ExecCommand implements the EXEC command
type ExecCommand struct{}

func (c *ExecCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 0 {
		return fmt.Errorf("wrong number of arguments for 'exec' command")
	}

	// Check if we're in a transaction
	if ctx.TransportCtx.ConnInfo.Metadata == nil {
		return ctx.RespWriter.WriteError("EXEC without MULTI")
	}

	state, exists := ctx.TransportCtx.ConnInfo.Metadata["transaction_state"]
	if !exists || state != "multi" {
		return ctx.RespWriter.WriteError("EXEC without MULTI")
	}

	// Get queued commands
	_, exists = ctx.TransportCtx.ConnInfo.Metadata["transaction_queue"]
	if !exists {
		// queue = []interface{}{}
	}

	// Clear transaction state
	delete(ctx.TransportCtx.ConnInfo.Metadata, "transaction_state")
	delete(ctx.TransportCtx.ConnInfo.Metadata, "transaction_queue")

	// For now, just return empty array (no commands were actually queued)
	// In a full implementation, we would execute all queued commands atomically
	results := []interface{}{}
	
	return ctx.RespWriter.WriteArray(results)
}

func (c *ExecCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "EXEC",
		Summary:      "Execute all commands issued after MULTI",
		Syntax:       "EXEC",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      0,
		MaxArgs:      0,
		ModifiesData: true,
	}
}

func (c *ExecCommand) ModifiesData() bool {
	return true
}

// DiscardCommand implements the DISCARD command
type DiscardCommand struct{}

func (c *DiscardCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 0 {
		return fmt.Errorf("wrong number of arguments for 'discard' command")
	}

	// Check if we're in a transaction
	if ctx.TransportCtx.ConnInfo.Metadata == nil {
		return ctx.RespWriter.WriteError("DISCARD without MULTI")
	}

	state, exists := ctx.TransportCtx.ConnInfo.Metadata["transaction_state"]
	if !exists || state != "multi" {
		return ctx.RespWriter.WriteError("DISCARD without MULTI")
	}

	// Clear transaction state
	delete(ctx.TransportCtx.ConnInfo.Metadata, "transaction_state")
	delete(ctx.TransportCtx.ConnInfo.Metadata, "transaction_queue")
	
	// Clear any watched keys
	delete(ctx.TransportCtx.ConnInfo.Metadata, "watched_keys")

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *DiscardCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "DISCARD",
		Summary:      "Discard all commands issued after MULTI",
		Syntax:       "DISCARD",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      0,
		MaxArgs:      0,
		ModifiesData: false,
	}
}

func (c *DiscardCommand) ModifiesData() bool {
	return false
}

// WatchCommand implements the WATCH command
type WatchCommand struct{}

func (c *WatchCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs == 0 {
		return fmt.Errorf("wrong number of arguments for 'watch' command")
	}

	// Initialize metadata if needed
	if ctx.TransportCtx.ConnInfo.Metadata == nil {
		ctx.TransportCtx.ConnInfo.Metadata = make(map[string]interface{})
	}

	// Get current watched keys
	watchedKeys := []string{}
	if existing, exists := ctx.TransportCtx.ConnInfo.Metadata["watched_keys"]; exists {
		if keys, ok := existing.([]string); ok {
			watchedKeys = keys
		}
	}

	// Add new keys to watch
	for i := 0; i < nargs; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		watchedKeys = append(watchedKeys, key)
	}

	// Store watched keys in metadata
	ctx.TransportCtx.ConnInfo.Metadata["watched_keys"] = watchedKeys

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *WatchCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "WATCH",
		Summary:      "Watch the given keys to determine execution of the MULTI/EXEC block",
		Syntax:       "WATCH key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *WatchCommand) ModifiesData() bool {
	return false
}

// UnwatchCommand implements the UNWATCH command
type UnwatchCommand struct{}

func (c *UnwatchCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 0 {
		return fmt.Errorf("wrong number of arguments for 'unwatch' command")
	}

	// Clear watched keys
	if ctx.TransportCtx.ConnInfo.Metadata != nil {
		delete(ctx.TransportCtx.ConnInfo.Metadata, "watched_keys")
	}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *UnwatchCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "UNWATCH",
		Summary:      "Forget about all watched keys",
		Syntax:       "UNWATCH",
		Categories:   []engine.CommandCategory{engine.CategoryGeneric},
		MinArgs:      0,
		MaxArgs:      0,
		ModifiesData: false,
	}
}

func (c *UnwatchCommand) ModifiesData() bool {
	return false
}
