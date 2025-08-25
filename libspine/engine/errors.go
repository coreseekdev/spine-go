package engine

import (
	"fmt"
)

// ErrorType represents the type of error
type ErrorType int

const (
	// ErrorTypeUnknownCommand indicates the command is not found
	ErrorTypeUnknownCommand ErrorType = iota
	// ErrorTypeInvalidArgument indicates an argument is invalid
	ErrorTypeInvalidArgument
	// ErrorTypeInternalError indicates an internal error
	ErrorTypeInternalError
)

// EngineError represents an error from the engine
type EngineError struct {
	Type    ErrorType // Error type
	Message string    // Error message
	Command string    // Command that caused the error (if applicable)
	Hash    uint32    // Command hash (if applicable)
}

// Error implements the error interface
func (e *EngineError) Error() string {
	return e.Message
}

// IsUnknownCommandError checks if the error is an unknown command error
func IsUnknownCommandError(err error) bool {
	if engineErr, ok := err.(*EngineError); ok {
		return engineErr.Type == ErrorTypeUnknownCommand
	}
	return false
}

// NewUnknownCommandError creates a new unknown command error
func NewUnknownCommandError(command string, hash uint32) *EngineError {
	return &EngineError{
		Type:    ErrorTypeUnknownCommand,
		Message: fmt.Sprintf("unknown command: %s", command),
		Command: command,
		Hash:    hash,
	}
}
