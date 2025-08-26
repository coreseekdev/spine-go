package engine

import (
	"context"
	"hash/fnv"
	"strings"
	"sync"

	"spine-go/libspine/engine/resp"
	"spine-go/libspine/engine/storage"
	"spine-go/libspine/transport"
)

// CommandCategory represents command categories
type CommandCategory string

const (
	CategoryRead       CommandCategory = "READ"
	CategoryWrite      CommandCategory = "WRITE"
	CategoryList       CommandCategory = "LIST"
	CategorySet        CommandCategory = "SET"
	CategoryHash       CommandCategory = "HASH"
	CategoryString     CommandCategory = "STRING"
	CategoryZSet       CommandCategory = "ZSET"
	CategoryBitmap     CommandCategory = "BITMAP"
	CategoryConnection CommandCategory = "CONNECTION"
	CategoryServer     CommandCategory = "SERVER"
	CategoryGeneric    CommandCategory = "GENERIC"
)

// ArgReadingMode defines how command arguments should be read
type ArgReadingMode int

const (
	// LazyArgReading indicates arguments should be read on-demand by the command handler
	LazyArgReading ArgReadingMode = iota

	// PreReadAllArgs indicates all arguments should be pre-read before command execution
	PreReadAllArgs

	// PreReadFirstNArgs indicates the first N arguments should be pre-read
	PreReadFirstNArgs
)

// CommandInfo contains metadata about a command
type CommandInfo struct {
	Name         string            // Command name (uppercase)
	Summary      string            // Brief description
	Syntax       string            // Command syntax
	Categories   []CommandCategory // Command categories
	MinArgs      int               // Minimum number of arguments
	MaxArgs      int               // Maximum number of arguments (-1 for unlimited)
	ModifiesData bool              // Whether this command modifies data
	ArgReading   ArgReadingMode    // How arguments should be read (lazy, pre-read all, etc.)
	PreReadNArgs int               // Number of arguments to pre-read if ArgReading is PreReadFirstNArgs
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
	Engine        *Engine              // Engine instance
	Context       context.Context      // Request context
	Command       string               // Command name
	ReqReader     *resp.ReqReader      // RESP request reader
	RespWriter    *resp.RESPWriter     // RESP response writer
	Database      *storage.Database    // Current database
	TransportCtx  *transport.Context   // Transport context
}

// CommandRegistry manages command registration and lookup
type CommandRegistry struct {
	mu            sync.RWMutex
	commands      map[string]CommandHandler // command name -> handler
	commandHashes map[uint32]CommandHandler // command hash -> handler
	aliases       map[string]string         // alias -> canonical name
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands:      make(map[string]CommandHandler),
		commandHashes: make(map[uint32]CommandHandler),
		aliases:       make(map[string]string),
	}
}

// hashString computes a 32-bit FNV-1a hash of a string
func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// Register registers a command handler
func (r *CommandRegistry) Register(handler CommandHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := handler.GetInfo()
	cmdName := strings.ToUpper(info.Name)

	// Register by name
	r.commands[cmdName] = handler

	// Register by hash
	cmdHash := hashString(cmdName)
	r.commandHashes[cmdHash] = handler

	return nil
}

// RegisterAlias registers a command alias
func (r *CommandRegistry) RegisterAlias(alias, canonical string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	aliasUpper := strings.ToUpper(alias)
	canonicalUpper := strings.ToUpper(canonical)

	// Register alias by name
	r.aliases[aliasUpper] = canonicalUpper

	// Register alias by hash if the canonical command exists
	if handler, exists := r.commands[canonicalUpper]; exists {
		aliasHash := hashString(aliasUpper)
		r.commandHashes[aliasHash] = handler
	}
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

// GetCommandByHash retrieves a command handler by hash
func (r *CommandRegistry) GetCommandByHash(hash uint32) (CommandHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.commandHashes[hash]
	return handler, exists
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
