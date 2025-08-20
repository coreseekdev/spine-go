# Spine-Go Chat Application

A modular chat application built with Go, featuring multiple transport protocols and handlers.

## Architecture

The application is divided into four main parts:

1. **libspine** - Core library with transport layer and handler interfaces
2. **spine** - Main server entry point supporting TCP, Unix Socket, and WebSocket
3. **spine-cli** - TCP/Unix socket client for command-line interaction
4. **spine-ws** - WebSocket server with web interface

## Features

### Transport Layer
- **TCP Transport** - For traditional TCP connections
- **Unix Socket Transport** - For local IPC communication
- **WebSocket Transport** - For web-based real-time communication

### Handler Layer
- **Chat Handler** - Real-time chat with room support
- **Redis Handler** - Basic Redis operations (GET, SET, DELETE, EXISTS, TTL)

### Key Design Principles
- Unified transport layer interface
- Modular handler system with middleware support
- Efficient data handling with readers and writers to minimize copying
- Context-based request processing

## Building

```bash
# Build all components
go build -o spine ./spine/
go build -o spine-cli ./spine-cli/
go build -o spine-ws ./spine-ws/
```

## Usage

### Starting the Server

```bash
# Start with default settings (TCP:8080, WebSocket:8081, Unix Socket:/tmp/spine.sock)
./spine

# Start with custom settings
./spine -tcp-port=8080 -ws-port=8081 -unix-socket=/tmp/spine.sock -redis-addr=localhost:6379
```

### Using the CLI Client

```bash
# Chat mode (default)
./spine-cli -server=localhost:8080 -protocol=tcp -mode=chat

# Redis mode
./spine-cli -server=localhost:8080 -protocol=tcp -mode=redis

# Unix socket mode
./spine-cli -socket=/tmp/spine.sock -protocol=unix -mode=chat
```

### Using the WebSocket Interface

```bash
# Start WebSocket server
./spine-ws -port=8081

# Access web interface
open http://localhost:8081
```

## Chat Commands

### CLI Chat Mode
- `/join <room>` - Join a chat room
- `/leave <room>` - Leave a chat room
- `/get <room>` - Get messages from a room
- `/quit` - Exit the client

### Redis Commands
- `SET <key> <value> [ttl]` - Set a key-value pair
- `GET <key>` - Get value by key
- `DELETE <key>` - Delete a key
- `EXISTS <key>` - Check if key exists
- `TTL <key>` - Get time-to-live for key

## Web Interface

The WebSocket server provides a web interface with:
- Real-time chat with multiple rooms
- Redis operations through web interface
- Tabbed interface for different services

## Configuration

### Server Options
- `-tcp-port` - TCP server port (default: 8080)
- `-ws-port` - WebSocket server port (default: 8081)
- `-unix-socket` - Unix socket path (default: /tmp/spine.sock)
- `-redis-addr` - Redis server address (default: localhost:6379)
- `-redis-pass` - Redis password (default: "")
- `-redis-db` - Redis database number (default: 0)
- `-enable-tcp` - Enable TCP server (default: true)
- `-enable-unix` - Enable Unix socket server (default: true)
- `-enable-ws` - Enable WebSocket server (default: true)

### Client Options
- `-server` - Server address (default: localhost:8080)
- `-protocol` - Protocol (tcp/unix) (default: tcp)
- `-socket` - Unix socket path (default: /tmp/spine.sock)
- `-mode` - Mode (chat/redis) (default: chat)

## Dependencies

- `github.com/gorilla/websocket` - WebSocket implementation
- `github.com/go-redis/redis/v8` - Redis client

## Project Structure

```
spine-go/
├── libspine/           # Core library
│   ├── transport/      # Transport layer implementations
│   ├── handler/        # Handler implementations
│   ├── server.go       # Server implementation
│   └── client.go       # Client implementation
├── spine/              # Main server entry point
├── spine-cli/          # CLI client
├── spine-ws/           # WebSocket server
└── go.mod              # Go module definition
```

## Examples

### Basic Chat Session

1. Start the server:
```bash
./spine
```

2. Start two CLI clients:
```bash
# Terminal 1
./spine-cli -mode=chat
# Enter username: alice
# /join general
# Hello everyone!

# Terminal 2
./spine-cli -mode=chat
# Enter username: bob
# /join general
# Hi alice!
```

### Redis Operations

1. Start the server with Redis:
```bash
./spine -redis-addr=localhost:6379
```

2. Use CLI client:
```bash
./spine-cli -mode=redis
redis> SET mykey hello
redis> GET mykey
redis> DELETE mykey
```

### Web Interface

1. Start WebSocket server:
```bash
./spine-ws
```

2. Open browser to `http://localhost:8081`
3. Use the web interface for chat and Redis operations