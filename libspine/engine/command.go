package engine

import (
	"context"
	"strings"
	"sync"

	"spine-go/libspine/engine/storage"
	"spine-go/libspine/transport"
)

// CommandCategory represents command categories
type CommandCategory string

const (
	CategoryRead      CommandCategory = "READ"
	CategoryWrite     CommandCategory = "WRITE"
	CategoryList      CommandCategory = "LIST"
	CategorySet       CommandCategory = "SET"
	CategoryHash      CommandCategory = "HASH"
	CategoryString    CommandCategory = "STRING"
	CategoryConnection CommandCategory = "CONNECTION"
	CategoryServer    CommandCategory = "SERVER"
	CategoryGeneric   CommandCategory = "GENERIC"
)

// CommandInfo contains metadata about a command
type CommandInfo struct {
	Name        string            // Command name (uppercase)
	Summary     string            // Brief description
	Syntax      string            // Command syntax
	Categories  []CommandCategory // Command categories
	MinArgs     int               // Minimum number of arguments
	MaxArgs     int               // Maximum number of arguments (-1 for unlimited)
	ModifiesData bool             // Whether this command modifies data
}

// CommandHandler defines the interface for command handlers
type CommandHandler interface {
	// Execute executes the command
	Execute(ctx *CommandContext) error
	
	// GetInfo returns command metadata
	GetInfo() *CommandInfo
	
	// ModifiesData returns true if this command modifies data
	ModifiesData() bool
}

// CommandContext provides context for command execution
type CommandContext struct {
	Engine   *Engine           // Engine instance
	Context  context.Context   // Request context
	Command  string            // Command name
	Args     []string          // Command arguments
	Reader   transport.Reader  // Request reader
	Writer   transport.Writer  // Response writer
	Database *storage.Database // Current database
}

// CommandRegistry manages command registration and lookup
type CommandRegistry struct {
	mu       sync.RWMutex
	commands map[string]CommandHandler // command name -> handler
	aliases  map[string]string         // alias -> canonical name
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]CommandHandler),
		aliases:  make(map[string]string),
	}
}

// Register registers a command handler
func (r *CommandRegistry) Register(handler CommandHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := handler.GetInfo()
	cmdName := strings.ToUpper(info.Name)

	r.commands[cmdName] = handler
	return nil
}

// RegisterCommand registers a command handler with explicit name
func (r *CommandRegistry) RegisterCommand(name string, handler CommandHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmdName := strings.ToUpper(name)
	r.commands[cmdName] = handler
}

// RegisterAlias registers a command alias
func (r *CommandRegistry) RegisterAlias(alias, canonical string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.aliases[strings.ToUpper(alias)] = strings.ToUpper(canonical)
}

// GetCommand retrieves a command handler by name
func (r *CommandRegistry) GetCommand(name string) (CommandHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmdName := strings.ToUpper(name)

	// Check for direct command
	if handler, exists := r.commands[cmdName]; exists {
		return handler, true
	}

	// Check for alias
	if canonical, exists := r.aliases[cmdName]; exists {
		if handler, exists := r.commands[canonical]; exists {
			return handler, true
		}
	}

	return nil, false
}

// ListCommands returns all registered commands
func (r *CommandRegistry) ListCommands() map[string]*CommandInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*CommandInfo)
	for name, handler := range r.commands {
		result[name] = handler.GetInfo()
	}
	return result
}

// GetCommandsByCategory returns commands in a specific category
func (r *CommandRegistry) GetCommandsByCategory(category CommandCategory) []*CommandInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*CommandInfo
	for _, handler := range r.commands {
		info := handler.GetInfo()
		for _, cat := range info.Categories {
			if cat == category {
				result = append(result, info)
				break
			}
		}
	}
	return result
}