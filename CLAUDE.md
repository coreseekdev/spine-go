# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Build all components
go build -o spine ./spine/
go build -o spine-cli ./spine-cli/
go build -o spine-ws ./spine-ws/

# Build individual components
go build -o spine ./spine/main.go
go build -o spine-cli ./spine-cli/main.go
go build -o spine-ws ./spine-ws/main.go

# Install dependencies
go mod tidy
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

- **spine/**: Server entry point that can run multiple transports simultaneously
- **spine-cli/**: Command-line client supporting both chat and Redis modes
- **spine-ws/**: Standalone WebSocket server with web interface

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

## Important Implementation Notes

- All ID generation functions must have unique names to avoid conflicts (generateServerID, generateClientID, etc.)
- Error handling should return appropriate HTTP status codes through the transport layer
- The WebSocket server (spine-ws) is separate from the main server and includes its own web interface
- Redis handler supports basic operations but is designed for demonstration purposes