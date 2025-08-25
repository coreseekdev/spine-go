package commands

import (
	"spine-go/libspine/engine"
)

// RegisterAllCommands registers all built-in Redis commands
func RegisterAllCommands(registry *engine.CommandRegistry) error {
	// Register global commands
	if err := RegisterGlobalCommands(registry); err != nil {
		return err
	}

	// Register storage commands
	if err := RegisterStorageCommands(registry); err != nil {
		return err
	}

	return nil
}