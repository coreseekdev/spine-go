package engine

// GetCommandInfo retrieves command metadata by name
func (r *CommandRegistry) GetCommandInfo(name string) (*CommandInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmdName := name
	
	// Check for direct command
	if handler, exists := r.commands[cmdName]; exists {
		return handler.GetInfo(), true
	}

	// Check for alias
	if canonical, exists := r.aliases[cmdName]; exists {
		if handler, exists := r.commands[canonical]; exists {
			return handler.GetInfo(), true
		}
	}

	return nil, false
}

// GetCommandInfoByHash retrieves command metadata by hash
func (r *CommandRegistry) GetCommandInfoByHash(hash uint32) (*CommandInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if handler, exists := r.commandHashes[hash]; exists {
		return handler.GetInfo(), true
	}

	return nil, false
}
