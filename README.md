# Spine-Go Chat Application

A modular chat application built with Go, featuring multiple transport protocols and handlers.

## Architecture

The application is divided into four main parts:

1. **libspine** - Core library with transport layer and handler interfaces
2. **spine** - Main server supporting TCP, Unix Socket, and WebSocket transports
3. **spine-cli** - TCP/Unix socket client for command-line interaction
4. **spine-ws** - WebSocket client application with web interface

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
# Build all components using make
make all

# Or build individual components
make spine
make spine-cli
make spine-ws

# Alternatively, build manually
go build -o bin/spine ./cmd/spine/
go build -o bin/spine-cli ./cmd/spine-cli/
go build -o bin/spine-ws ./cmd/spine-ws/
```

## Usage

### Starting the Server

```bash
# Start with default settings
./bin/spine

# Start with custom listen addresses
./bin/spine -listen=tcp://:8080 -listen=ws://:8081 -listen=unix:///tmp/spine.sock

# Start with static file serving for web UI
./bin/spine -static=./web
```

### Using the CLI Client

```bash
# Chat mode (default)
./bin/spine-cli -server=localhost:8080 -protocol=tcp -mode=chat

# Redis mode
./bin/spine-cli -server=localhost:8080 -protocol=tcp -mode=redis

# Unix socket mode
./bin/spine-cli -socket=/tmp/spine.sock -protocol=unix -mode=chat
```

### Using the WebSocket Interface

```bash
# Start WebSocket client
./bin/spine-ws -port=8081

# Or use the convenience script
./start-web-chat.sh

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
- `-listen` - Listen address (format: schema://host:port, e.g., tcp://:8080, ws://:8081, unix:///tmp/spine.sock). Can be specified multiple times.
- `-mode` - Server mode (chat/redis) (default: chat)
- `-static` - Static files path for chat webui

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
├── bin/               # Compiled executables
├── cmd/               # Source code for executables
│   ├── spine/         # Main server source
│   ├── spine-cli/     # CLI client source
│   └── spine-ws/      # WebSocket client source
├── libspine/          # Core library
│   ├── transport/     # Transport layer implementations
│   ├── handler/       # Handler implementations
│   ├── server.go      # Server implementation
│   └── client.go      # Client implementation
├── web/               # Web UI static files
│   ├── index.html     # Main HTML page
│   ├── chat.js        # JavaScript for chat functionality
│   └── style.css      # CSS styling
└── go.mod             # Go module definition
```

## Examples

### Basic Chat Session

1. Start the server:
```bash
./bin/spine
```

2. Start two CLI clients:
```bash
# Terminal 1
./bin/spine-cli -mode=chat
# Enter username: alice
# /join general
# Hello everyone!

# Terminal 2
./bin/spine-cli -mode=chat
# Enter username: bob
# /join general
# Hi alice!
```

### Redis Operations

1. Start the server with Redis mode:
```bash
./bin/spine -mode=redis
```

2. Use CLI client:
```bash
./bin/spine-cli -mode=redis
redis> SET mykey hello
redis> GET mykey
redis> DELETE mykey
```

### Web Interface

1. Start WebSocket client:
```bash
./bin/spine-ws
```

2. Open browser to `http://localhost:8081`
3. Use the web interface for chat and Redis operations