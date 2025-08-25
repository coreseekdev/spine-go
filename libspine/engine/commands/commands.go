package commands

import "spine-go/libspine/engine"

// RegisterAllCommands registers all available commands
func RegisterAllCommands(registry *engine.CommandRegistry) error {
	// Register global commands
	if err := RegisterGlobalCommands(registry); err != nil {
		return err
	}

	// Register storage commands
	if err := RegisterStorageCommands(registry); err != nil {
		return err
	}

	// Register string commands
	if err := RegisterStringCommands(registry); err != nil {
		return err
	}

	// Register hash commands
	if err := RegisterHashCommands(registry); err != nil {
		return err
	}

	// Register list commands
	if err := RegisterListCommands(registry); err != nil {
		return err
	}

	// Register set commands
	if err := RegisterSetCommands(registry); err != nil {
		return err
	}

	// Register sorted set commands
	if err := RegisterZSetCommands(registry); err != nil {
		return err
	}

	// Register pub/sub commands
	if err := RegisterPubSubCommands(registry); err != nil {
		return err
	}

	return nil
}