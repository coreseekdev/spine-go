package engine

import (
	"context"
	"fmt"
	"sync"

	"spine-go/libspine/engine/storage"
	"spine-go/libspine/engine/wal"
	"spine-go/libspine/transport"
)

// Engine represents the main database engine
type Engine struct {
	mu       sync.RWMutex
	storages map[int]*storage.Database // database instances indexed by db number
	wal      *wal.WAL                  // write-ahead log
	cmdReg   *CommandRegistry          // command registry
	currentDB int                      // current selected database
}

// NewEngine creates a new database engine
func NewEngine(walPath string) (*Engine, error) {
	walInstance, err := wal.New(walPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create WAL: %w", err)
	}

	engine := &Engine{
		storages:  make(map[int]*storage.Database),
		wal:      walInstance,
		cmdReg:   NewCommandRegistry(),
		currentDB: 0,
	}

	// Initialize default database (db 0)
	engine.storages[0] = storage.NewDatabase(0)

	// Register built-in commands
	engine.registerBuiltinCommands()

	return engine, nil
}

// SelectDatabase selects a database by number
func (e *Engine) SelectDatabase(dbNum int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if dbNum < 0 || dbNum > 15 { // Redis supports 16 databases by default
		return fmt.Errorf("invalid database number: %d", dbNum)
	}

	// Create database if it doesn't exist
	if _, exists := e.storages[dbNum]; !exists {
		e.storages[dbNum] = storage.NewDatabase(dbNum)
	}

	e.currentDB = dbNum
	return nil
}

// GetCurrentDatabase returns the current selected database
func (e *Engine) GetCurrentDatabase() *storage.Database {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.storages[e.currentDB]
}

// GetDatabase returns a specific database by number
func (e *Engine) GetDatabase(dbNum int) *storage.Database {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.storages[dbNum]
}

// GetCommandRegistry returns the command registry
func (e *Engine) GetCommandRegistry() *CommandRegistry {
	return e.cmdReg
}

// ExecuteCommand executes a Redis command
func (e *Engine) ExecuteCommand(ctx context.Context, cmd string, args []string, reader transport.Reader, writer transport.Writer) error {
	// Get command handler
	handler, exists := e.cmdReg.GetCommand(cmd)
	if !exists {
		return fmt.Errorf("unknown command: %s", cmd)
	}

	// Create command context
	cmdCtx := &CommandContext{
		Engine:   e,
		Context:  ctx,
		Command:  cmd,
		Args:     args,
		Reader:   reader,
		Writer:   writer,
		Database: e.GetCurrentDatabase(),
	}

	// Execute command
	err := handler.Execute(cmdCtx)
	if err != nil {
		return err
	}

	// Write to WAL if command modified data
	if handler.ModifiesData() {
		entry := &wal.Entry{
			Database: e.currentDB,
			Command:  cmd,
			Args:     args,
		}
		if walErr := e.wal.Write(entry); walErr != nil {
			// Log WAL error but don't fail the command
			fmt.Printf("WAL write error: %v\n", walErr)
		}
	}

	return nil
}

// Close closes the engine and all its resources
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.wal != nil {
		return e.wal.Close()
	}
	return nil
}

// registerBuiltinCommands registers all built-in Redis commands
func (e *Engine) registerBuiltinCommands() {
	// This will be implemented when we create the commands package
	// For now, we'll add a placeholder
}