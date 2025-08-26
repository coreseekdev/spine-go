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
