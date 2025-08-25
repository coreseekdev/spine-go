package wal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

// Entry represents a WAL entry
type Entry struct {
	Timestamp int64    // Unix timestamp in nanoseconds
	Database  int      // Database number
	Command   string   // Command name
	Args      []string // Command arguments
}

// WAL represents the Write-Ahead Log
type WAL struct {
	mu       sync.Mutex
	file     *os.File
	writer   *bufio.Writer
	encoder  *zstd.Encoder
	filePath string
	closed   bool
}

// New creates a new WAL instance
func New(filePath string) (*WAL, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file: %w", err)
	}

	bufWriter := bufio.NewWriter(file)
	encoder, err := zstd.NewWriter(bufWriter)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}

	return &WAL{
		file:     file,
		writer:   bufWriter,
		encoder:  encoder,
		filePath: filePath,
	}, nil
}

// Write writes an entry to the WAL
func (w *WAL) Write(entry *Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return fmt.Errorf("WAL is closed")
	}

	// Set timestamp if not provided
	if entry.Timestamp == 0 {
		entry.Timestamp = time.Now().UnixNano()
	}

	// Serialize entry to RESP3 format
	data, err := w.serializeEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to serialize entry: %w", err)
	}

	// Write compressed data
	_, err = w.encoder.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to WAL: %w", err)
	}

	// Flush encoder and buffer
	err = w.encoder.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush encoder: %w", err)
	}

	err = w.writer.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	// Sync to disk
	err = w.file.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync WAL: %w", err)
	}

	return nil
}

// serializeEntry serializes an entry to RESP3 format
func (w *WAL) serializeEntry(entry *Entry) ([]byte, error) {
	// RESP3 format for WAL entry:
	// %5\r\n                    // Map with 5 elements
	// +timestamp\r\n          // Key: timestamp
	// :1234567890\r\n         // Value: timestamp as integer
	// +database\r\n           // Key: database
	// :0\r\n                  // Value: database number
	// +command\r\n            // Key: command
	// +SET\r\n                // Value: command name
	// +args\r\n               // Key: args
	// *2\r\n                  // Array with 2 elements
	// +key\r\n                // First arg
	// +value\r\n              // Second arg

	var result []byte

	// Map header
	result = append(result, fmt.Sprintf("%%4\r\n")...)

	// Timestamp
	result = append(result, "+timestamp\r\n"...)
	result = append(result, fmt.Sprintf(":%d\r\n", entry.Timestamp)...)

	// Database
	result = append(result, "+database\r\n"...)
	result = append(result, fmt.Sprintf(":%d\r\n", entry.Database)...)

	// Command
	result = append(result, "+command\r\n"...)
	result = append(result, fmt.Sprintf("+%s\r\n", entry.Command)...)

	// Args
	result = append(result, "+args\r\n"...)
	result = append(result, fmt.Sprintf("*%d\r\n", len(entry.Args))...)
	for _, arg := range entry.Args {
		result = append(result, fmt.Sprintf("+%s\r\n", arg)...)
	}

	return result, nil
}

// ReadEntries reads all entries from the WAL file
func ReadEntries(filePath string) ([]*Entry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No WAL file exists yet
		}
		return nil, fmt.Errorf("failed to open WAL file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	decoder, err := zstd.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer decoder.Close()

	var entries []*Entry
	for {
		entry, err := readEntry(decoder)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// readEntry reads a single entry from the decoder
func readEntry(reader io.Reader) (*Entry, error) {
	// This is a simplified RESP3 parser for WAL entries
	// In a production system, you'd want a more robust parser
	buf := make([]byte, 4096)
	n, err := reader.Read(buf)
	if err != nil {
		return nil, err
	}

	// For now, return a placeholder entry
	// TODO: Implement proper RESP3 parsing
	return &Entry{
		Timestamp: time.Now().UnixNano(),
		Database:  0,
		Command:   "PLACEHOLDER",
		Args:      []string{string(buf[:n])},
	}, nil
}

// Close closes the WAL
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true

	if w.encoder != nil {
		w.encoder.Close()
	}

	if w.writer != nil {
		w.writer.Flush()
	}

	if w.file != nil {
		return w.file.Close()
	}

	return nil
}

// Truncate truncates the WAL file
func (w *WAL) Truncate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return fmt.Errorf("WAL is closed")
	}

	return w.file.Truncate(0)
}