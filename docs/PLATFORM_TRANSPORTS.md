# Platform-Specific Transport Support

spine-go now supports platform-specific transport layers that automatically adapt to the underlying operating system.

## Supported Transports by Platform

### All Platforms
- **TCP Transport** (`tcp`): Available on all platforms
- **WebSocket Transport** (`http`): Available on all platforms

### Unix/Linux/macOS Only
- **Unix Socket Transport** (`unix`): POSIX-compliant platforms only
  - Uses Unix domain sockets for high-performance local communication
  - Path-based addressing (e.g., `/tmp/spine.sock`)

### Windows Only
- **Named Pipe Transport** (`namedpipe`): Windows platforms only
  - Uses Windows Named Pipes for local inter-process communication
  - Pipe name addressing (e.g., `spine_pipe` → `\\.\pipe\spine_pipe`)

## Configuration

### Server Configuration

```go
// Unix/Linux configuration
config := &libspine.Config{
    ListenConfigs: []libspine.ListenConfig{
        {
            Schema: "unix",
            Path:   "/tmp/spine.sock",
        },
    },
    ServerMode: "redis", // or "chat"
}

// Windows configuration
config := &libspine.Config{
    ListenConfigs: []libspine.ListenConfig{
        {
            Schema: "namedpipe",
            Path:   "spine_pipe",
        },
    },
    ServerMode: "redis", // or "chat"
}
```

### Cross-Platform Configuration

```go
import "runtime"

func createPlatformConfig() *libspine.Config {
    var listenConfig libspine.ListenConfig
    
    if runtime.GOOS == "windows" {
        listenConfig = libspine.ListenConfig{
            Schema: "namedpipe",
            Path:   "spine_pipe",
        }
    } else {
        listenConfig = libspine.ListenConfig{
            Schema: "unix",
            Path:   "/tmp/spine.sock",
        }
    }
    
    return &libspine.Config{
        ListenConfigs: []libspine.ListenConfig{listenConfig},
        ServerMode:    "redis",
    }
}
```

## Implementation Details

### Build Constraints

The platform-specific implementations use Go build constraints:

- `//go:build windows` - Windows-only code
- `//go:build !windows` - Non-Windows (Unix/Linux/macOS) code

### Error Handling

When attempting to use an unsupported transport on a platform:

```go
// On Windows
_, err := transport.NewUnixSocketTransport("/tmp/test.sock")
// Returns: "Unix socket transport is not supported on Windows platform"

// On Unix/Linux
_, err := transport.NewNamedPipeTransport("test_pipe")
// Returns: "Named Pipe transport is not supported on Unix/Linux platforms, use Unix socket instead"
```

## Performance Characteristics

### Unix Socket Transport
- **Best for**: High-performance local communication on Unix systems
- **Advantages**: Lower overhead than TCP, filesystem permissions
- **Use cases**: Local microservices, high-throughput applications

### Named Pipe Transport
- **Best for**: Windows inter-process communication
- **Advantages**: Native Windows IPC, security integration
- **Use cases**: Windows services, desktop applications

### TCP Transport
- **Best for**: Network communication, cross-platform compatibility
- **Advantages**: Universal support, network-capable
- **Use cases**: Distributed systems, remote access

## Client Support

### Unix Socket Client (Go)
```go
conn, err := net.Dial("unix", "/tmp/spine.sock")
```

### Named Pipe Client (Windows Go)
```go
import "golang.org/x/sys/windows"

// Connect to named pipe
pipeName := `\\.\pipe\spine_pipe`
// Use Windows API or third-party libraries
```

### TCP Client (All Platforms)
```go
conn, err := net.Dial("tcp", "127.0.0.1:6379")
```

## Testing

Platform-specific transport tests automatically adapt to the current platform:

```bash
# Run platform-specific transport tests
go test -v ./test/ -run TestPlatformSpecificTransports

# Results on Linux:
# ✓ Unix socket correctly supported
# ✓ Named pipe correctly rejected

# Results on Windows:
# ✓ Unix socket correctly rejected  
# ✓ Named pipe correctly supported
```

## Migration Guide

### From Single Platform to Cross-Platform

1. **Identify current transport usage**
2. **Add platform detection logic**
3. **Update configuration based on platform**
4. **Test on target platforms**

### Example Migration

```go
// Before (Unix-only)
config := &libspine.Config{
    ListenConfigs: []libspine.ListenConfig{
        {Schema: "unix", Path: "/tmp/spine.sock"},
    },
}

// After (Cross-platform)
func createConfig() *libspine.Config {
    var transport libspine.ListenConfig
    
    switch runtime.GOOS {
    case "windows":
        transport = libspine.ListenConfig{
            Schema: "namedpipe",
            Path:   "spine_pipe",
        }
    default:
        transport = libspine.ListenConfig{
            Schema: "unix", 
            Path:   "/tmp/spine.sock",
        }
    }
    
    return &libspine.Config{
        ListenConfigs: []libspine.ListenConfig{transport},
        ServerMode:    "redis",
    }
}
```

## Troubleshooting

### Common Issues

1. **Permission denied on Unix socket**
   - Check filesystem permissions on socket path
   - Ensure directory exists and is writable

2. **Named pipe access denied on Windows**
   - Check Windows permissions
   - Run with appropriate privileges

3. **Platform mismatch errors**
   - Verify build constraints are working
   - Check runtime.GOOS detection

### Debugging

Enable transport-specific logging:
```go
log.SetLevel(log.DebugLevel)
// Transport startup and connection logs will be visible
```
