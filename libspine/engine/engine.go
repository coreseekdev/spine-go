package engine

import (
	"context"
	"sync"

	"spine-go/libspine/engine/pubsub"
	"spine-go/libspine/engine/resp"
	"spine-go/libspine/engine/storage"
	"spine-go/libspine/engine/wal"
	"spine-go/libspine/transport"
)

// Engine represents the main database engine
type Engine struct {
	mu        sync.RWMutex
	storages  map[int]*storage.Database // database instances indexed by db number
	wal       *wal.WAL                  // write-ahead log
	cmdReg    *CommandRegistry          // command registry
	pubsubMgr *pubsub.PubSubManager     // pub/sub manager
	currentDB int                       // current selected database
}

// NewEngine creates a new database engine
func NewEngine(walPath string) (*Engine, error) {
	// walInstance, err := wal.New(walPath)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create WAL: %w", err)
	// }

	engine := &Engine{
		storages:  make(map[int]*storage.Database),
		wal:       nil,
		cmdReg:    NewCommandRegistry(),
		pubsubMgr: pubsub.NewPubSubManager(),
		currentDB: 0,
	}

	// Initialize default database (db 0)
	engine.storages[0] = storage.NewDatabase(0)

	// Register built-in commands
	// engine.registerBuiltinCommands()

	return engine, nil
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

// GetPubSubManager returns the pub/sub manager
func (e *Engine) GetPubSubManager() *pubsub.PubSubManager {
	return e.pubsubMgr
}

// ExecuteCommand executes a Redis command using command hash for faster dispatch
func (e *Engine) ExecuteCommand(transportCtx *transport.Context, cmdHash uint32, cmdName string, reqReader *resp.ReqReader, respWriter *resp.RESPWriter) error {
	// Get command handler by hash
	handler, exists := e.cmdReg.GetCommandByHash(cmdHash)
	if !exists {
		return NewUnknownCommandError(cmdName, cmdHash)
	}

	// Get selected database from connection metadata
	selectedDB := 0 // Default to DB 0
	if transportCtx.ConnInfo != nil && transportCtx.ConnInfo.Metadata != nil {
		if dbVal, ok := transportCtx.ConnInfo.Metadata[transport.MetadataSelectedDB]; ok {
			if db, ok := dbVal.(int); ok {
				selectedDB = db
			}
		}
	}

	// Get the selected database
	db, exists := e.storages[selectedDB]
	if !exists {
		// Create database if it doesn't exist
		e.mu.Lock()
		db = storage.NewDatabase(selectedDB)
		e.storages[selectedDB] = db
		e.mu.Unlock()
	}

	// Create command context with background context
	cmdCtx := &CommandContext{
		Engine:       e,
		Context:      context.Background(),
		Command:      cmdName,
		ReqReader:    reqReader,
		RespWriter:   respWriter,
		Database:     db, // Use the selected database from metadata
		TransportCtx: transportCtx,
	}

	// Execute command
	err := handler.Execute(cmdCtx)
	if err != nil {
		return err
	}

	// Write to WAL if command modified data
	if handler.ModifiesData() {
		// TODO: 先暂时禁用 WAL 等待数据处理主流程通过。
		/*
			entry := &wal.Entry{
				Timestamp: time.Now().Unix(),
				Database:  e.currentDB,
				Command:   cmdName,
				// We don't have args anymore as they're read from reqReader
				Args: []string{}, // Empty args, as we now use ReqReader for argument reading
			}
			if err := e.wal.Write(entry); err != nil {
				return fmt.Errorf("failed to write to WAL: %w", err)
			}
		*/
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
