# Redis Handler Test Framework

This directory contains an independent test framework specifically designed for testing the Redis handler implementation in spine-go.

## Overview

The Redis handler provides an in-memory Redis-compatible server that implements the Redis RESP (REdis Serialization Protocol) for communication. This test framework validates all core Redis operations.

## Architecture

### Components

1. **RedisTestClient** (`redis_client.go`)
   - Pure Go Redis client implementation
   - Supports RESP protocol parsing
   - Provides convenient methods for common Redis operations

2. **RedisTestSuite** (`redis_test.go`)
   - Test framework that sets up a minimal server environment
   - Handles connection management and test lifecycle
   - Provides comprehensive test coverage

### Supported Redis Commands

- `PING` - Server connectivity test
- `SET key value [EX seconds]` - Set key-value pairs with optional TTL
- `GET key` - Retrieve values by key
- `DEL key` - Delete keys
- `EXISTS key` - Check key existence
- `TTL key` - Get remaining time to live

## Test Coverage

### Basic Commands Test
- Tests fundamental Redis operations (PING, SET, GET, EXISTS, DEL)
- Validates proper RESP protocol responses
- Ensures data consistency

### TTL Commands Test
- Tests time-to-live functionality
- Validates automatic key expiration
- Tests TTL queries for existing and non-existing keys

### Multiple Keys Test
- Tests operations across multiple keys
- Validates independent key management
- Tests bulk operations

### Error Handling Test
- Tests error responses for invalid operations
- Validates proper error formatting
- Tests edge cases and boundary conditions

### Performance Benchmark
- Measures SET/GET operation performance
- Current performance: ~149,496 ns/op (≈6,688 ops/sec)

## Running Tests

```bash
# Run all tests
cd test/redis
go test -v

# Run specific test
go test -v -run TestBasicCommands

# Run benchmarks
go test -bench=BenchmarkSetGet -v

# Run with coverage
go test -cover -v
```

## Implementation Details

### In-Memory Storage
- Thread-safe map-based storage with RWMutex
- Automatic TTL expiration using goroutines
- Memory-efficient key-value storage

### RESP Protocol
- Full RESP protocol implementation
- Supports all Redis data types (strings, integers, arrays, errors, null)
- Compatible with standard Redis clients

### Connection Handling
- TCP-based connection management
- Persistent connections for multiple commands
- Graceful connection cleanup

## Integration

This test framework is completely independent from the chat e2e tests, allowing:
- Focused Redis functionality testing
- Independent development and maintenance
- Clear separation of concerns between different protocol handlers
