# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Build all components using Makefile
make all
make build

# Build individual components
make spine      # Main server
make spine-cli  # CLI client
make spine-ws   # WebSocket client

# Alternative manual builds
go build -o bin/spine ./cmd/spine/
go build -o bin/spine-cli ./cmd/spine-cli/
go build -o bin/spine-ws ./cmd/spine-ws/

# Install dependencies
go mod tidy

# Clean build artifacts
make clean

# Run components
make run       # Main server
make run-cli   # CLI client
make run-ws    # WebSocket client
```

## Testing Commands

```bash
# Run unit tests
go test ./libspine/handler/ -v
go test ./test/redis/ -v
go test ./test/e2e/ -v

# Run integration tests
go test ./integration_test.go -v

# Run all tests
go test ./... -v

# Run tests with coverage
go test -cover ./...
```

## Architecture Overview

This is a modular chat application with a unified transport layer and pluggable handlers. The architecture follows a strict separation of concerns:

### Core Architecture Pattern
The application uses a **transport-handler pattern** where:
- **Transport layer** handles different protocols (TCP, Unix Socket, WebSocket)
- **Handler layer** contains business logic (chat, Redis operations)
- **Context objects** carry server/connection metadata
- **Reader/Writer interfaces** minimize data copying between layers

### Key Design Decisions

1. **Zero-Copy Data Flow**: The Reader/Writer interfaces allow handlers to process data without unnecessary copying between transport and application layers.

2. **Unified Transport Interface**: All transport protocols implement the same interface, making it easy to add new protocols without changing handler logic.

3. **Middleware Chain**: Handlers can be wrapped with middleware for logging, authentication, etc., using a chain-of-responsibility pattern.

4. **Protocol-Specific Implementations**: Each transport has its own request/response format optimized for its use case (HTTP-like for TCP, JSON for WebSocket, etc.).

### Component Relationships

- **libspine/**: Core library containing interfaces and implementations
  - `transport/`: Protocol implementations and common interfaces
  - `handler/`: Business logic handlers and middleware system
  - `server.go`: Main server orchestrating multiple transports
  - `client.go`: Client library for connecting to servers

- **cmd/spine/**: Server entry point that can run multiple transports simultaneously
- **cmd/spine-cli/**: Command-line client supporting both chat and Redis modes
- **cmd/spine-ws/**: Standalone WebSocket client with web interface

### Handler System

Handlers are registered by path and receive a Context containing:
- Server information (address, config)
- Connection information (ID, remote address, protocol)
- Reader for request data
- Writer for response data

The handler registry supports fallback to default handlers when specific paths aren't found.

### Transport Implementations

Each transport implements:
- `Accept()` for accepting new connections
- `NewHandlers()` for creating protocol-specific readers/writers
- `Close()` for cleanup

TCP and Unix Socket use HTTP-like protocols, while WebSocket uses JSON messages for better web compatibility.

## Project Structure

```
spine-go/
├── cmd/               # Executable entry points
│   ├── spine/         # Main server
│   ├── spine-cli/     # CLI client
│   └── spine-ws/      # WebSocket client
├── libspine/          # Core library
│   ├── transport/     # Transport layer (TCP, Unix, WebSocket)
│   ├── handler/       # Business logic handlers
│   ├── server.go      # Server orchestration
│   └── client.go      # Client library
├── test/              # Test suites
│   ├── e2e/          # End-to-end tests
│   └── redis/        # Redis compatibility tests
├── web/               # Web UI static files
├── go-redis/          # Redis implementation (vendored)
├── redcon/           # Redis protocol implementation
├── integration_test.go # Integration tests
└── Makefile          # Build automation
```

## Important Implementation Notes

- All ID generation functions must have unique names to avoid conflicts (generateServerID, generateClientID, etc.)
- Error handling should return appropriate HTTP status codes through the transport layer
- The WebSocket client (spine-ws) is separate from the main server and includes its own connection handling
- Redis handler supports basic operations but is designed for demonstration purposes
- Binary outputs are placed in `bin/` directory by the Makefile
- The server supports multiple simultaneous listeners via `-listen` flags