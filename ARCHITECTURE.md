# Architecture Documentation

## System Overview

The File Transfer application is built with a modular architecture separating concerns into distinct layers: UI, Network, File Operations, and Utilities.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────┐
│                   User Interface                     │
│              (Fyne-based GUI)                        │
├─────────────────────────────────────────────────────┤
│                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐│
│  │  Discovery   │  │   Session    │  │ Transfer  ││
│  │    View      │  │     View     │  │   View    ││
│  └──────────────┘  └──────────────┘  └───────────┘│
│                                                      │
├─────────────────────────────────────────────────────┤
│              Application Layer                       │
│                                                      │
│  ┌──────────────────┐      ┌────────────────────┐  │
│  │ Discovery Service│      │ Transfer Manager   │  │
│  │  - UDP Broadcast │      │ - TCP Connections  │  │
│  │  - Device List   │      │ - Session Mgmt     │  │
│  └──────────────────┘      └────────────────────┘  │
│                                                      │
├─────────────────────────────────────────────────────┤
│              Network Layer                           │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ │
│  │ Session  │  │ Protocol │  │ Optimization     │ │
│  │ Manager  │  │ Handler  │  │ (TCP Settings)   │ │
│  └──────────┘  └──────────┘  └──────────────────┘ │
│                                                      │
├─────────────────────────────────────────────────────┤
│              File Layer                              │
│                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────┐ │
│  │ File I/O     │  │ Validation   │  │ Icons    │ │
│  └──────────────┘  └──────────────┘  └──────────┘ │
│                                                      │
├─────────────────────────────────────────────────────┤
│              Utility Layer                           │
│                                                      │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌────────┐│
│  │ Logging │  │ Config  │  │ Storage │  │Helpers ││
│  └─────────┘  └─────────┘  └─────────┘  └────────┘│
└─────────────────────────────────────────────────────┘
```

## Module Breakdown

### 1. UI Layer (`internal/ui/`)

#### `app.go`
**Responsibility**: Main application controller and UI orchestration

**Key Components**:
- `FileTransferApp`: Main application struct
- Three view screens: Discovery, Session, Transfer
- Event handlers for user interactions

**State Management**:
- Active session tracking
- Current screen state
- Discovered devices list
- Selected files queue

**Flow**:
```
Start → Discovery → Connect → Session → Transfer → Disconnect → Discovery
```

### 2. Network Layer (`internal/network/`)

#### `discovery.go`
**Responsibility**: Automatic device discovery using UDP broadcast

**Key Features**:
- Broadcasts presence every 2 seconds
- Listens for broadcasts from other devices
- Maintains active device list
- Auto-removes stale devices (10s timeout)

**Protocol**:
```json
{
  "name": "Device Name",
  "ip": "192.168.1.100",
  "port": 9999,
  "device_type": "desktop",
  "last_seen": "timestamp"
}
```

**Concurrency Model**:
- 3 goroutines per service:
  - Broadcast sender
  - Broadcast receiver  
  - Stale device cleanup

#### `session.go`
**Responsibility**: Manages active connections between devices

**Session Lifecycle**:
1. `Pending`: Connection requested
2. `Active`: Connection established
3. `Closed`: Connection terminated

**Message Types**:
```go
MsgConnectionRequest  // Initial connection
MsgConnectionAccept   // Accept connection
MsgConnectionReject   // Reject connection
MsgFileOffer         // Offer a file
MsgFileAccept        // Accept file offer
MsgFileReject        // Reject file offer
MsgFileData          // File data chunks
MsgFileComplete      // Transfer complete
MsgProgressUpdate    // Progress notification
MsgKeepAlive         // Connection health check
```

**Channels**:
- `messageChan`: Protocol messages
- `progressChan`: Transfer progress
- `stopChan`: Shutdown signal

#### `transfer.go`
**Responsibility**: High-speed file transfer implementation

**Optimization Techniques**:
1. **Large Buffers**: 4MB TCP buffers
2. **TCP_NODELAY**: Disable Nagle's algorithm
3. **Keepalive**: 30-second intervals
4. **Optimal Chunks**: 256KB chunk size
5. **Non-blocking I/O**: Async operations

**Transfer Flow**:
```
Offer File → Wait Accept → Send Metadata → Transfer Data → Complete
```

**Performance**:
- Theoretical max: ~118 MB/s (Gigabit Ethernet)
- Practical: 80-100 MB/s (overhead considered)
- Progress updates: Every 100ms

#### `connection.go`
**Responsibility**: Network utilities and helpers

**Functions**:
- Get local IP address
- Enumerate network interfaces
- Validate IP addresses
- Check port availability

### 3. File Layer (`internal/file/`)

#### `file_ops.go`
**Responsibility**: File operations and validation

**Key Functions**:
- `ValidateFilePath()`: Check file accessibility
- `GetFileSize()`: Get file size in bytes
- `CalculateChecksum()`: SHA256 checksum
- `SplitFile()`: Split large files (future)
- `MergeChunks()`: Merge file chunks (future)
- `FormatFileSize()`: Human-readable sizes

#### `icon.go`
**Responsibility**: Platform-specific icon loading

**Platforms Supported**:
- Windows: `.ico`
- macOS: `.icns`
- Linux: `.png`
- Android/iOS: `.png`

### 4. Utility Layer (`internal/utils/`)

#### `logger.go`
**Responsibility**: Centralized logging system

**Features**:
- File-based logging
- Console output (debug mode)
- Timestamped entries
- Error categorization

**Log Levels**:
- `Log()`: General information
- `LogError()`: Error messages
- `LogDebug()`: Debug information

#### `config.go`
**Responsibility**: Application configuration

**Configuration Schema**:
```json
{
  "default_port": 9999,
  "chunk_size": 262144,
  "max_concurrency": 8,
  "download_path": "./downloads",
  "auto_accept": false,
  "enable_encryption": false
}
```

**Functions**:
- `LoadConfig()`: Load from JSON
- `SaveConfig()`: Save to JSON
- `GetDefaultConfig()`: Default values

#### `storage.go`
**Responsibility**: Download path management

**Features**:
- Path validation
- Directory creation
- Write permission checks
- Default path detection

#### `helpers.go`
**Responsibility**: Miscellaneous utilities

**Functions**:
- Time formatting
- Speed calculation
- ETA estimation
- Filename sanitization
- Retry logic

## Communication Protocols

### Discovery Protocol (UDP)

**Port**: 9998  
**Type**: Broadcast  
**Interval**: 2 seconds

**Packet Structure**:
```
JSON payload with device information
```

### Transfer Protocol (TCP)

**Port**: 9999  
**Type**: Point-to-point  
**Connection**: Persistent

**Message Format**:
```
[1 byte: Message Type]
[4 bytes: Payload Length]
[N bytes: Payload Data]
```

**File Transfer Format**:
```
[4 bytes: Filename Length]
[N bytes: Filename]
[8 bytes: File Size]
[Data chunks...]
```

## Concurrency Model

### Goroutines per Component

**Discovery Service**:
- 1x Broadcast sender
- 1x Broadcast receiver
- 1x Cleanup routine

**Transfer Manager**:
- 1x Connection acceptor
- 1x per active session

**Session**:
- 1x Message receiver
- 1x Keepalive sender

**File Transfer**:
- 1x per file transfer
- 1x Progress monitor

**Total**: ~8-12 goroutines during active transfer

### Channel Usage

**Buffered Channels**:
- `progressChan`: Size 10 (prevent blocking)
- `messageChan`: Size 10 (message queue)

**Unbuffered Channels**:
- `stopChan`: Shutdown signal
- `errChan`: Error propagation
- `doneChan`: Completion signal

### Synchronization

**Mutexes**:
- `sessionMu`: Protect active session
- `session.mu`: Protect session state

**Thread Safety**:
- All public APIs are thread-safe
- Internal state protected by mutexes
- Channels for cross-goroutine communication

## Data Flow

### Connection Establishment

```
Device A                    Device B
   |                           |
   |------ Discovery -------->|
   |<----- Discovery ---------|
   |                           |
   |--- Connect Request ----->|
   |                           |
   |                    [User Approval]
   |                           |
   |<--- Accept Response -----|
   |                           |
   [Session Active]    [Session Active]
```

### File Transfer

```
Sender                      Receiver
   |                           |
   |------ File Offer ------->|
   |<----- Accept File -------|
   |                           |
   |------ Filename --------->|
   |------ File Size -------->|
   |                           |
   |====== Data Chunks ======>|
   |                           |
   [Progress Updates]  [Progress Updates]
   |                           |
   |------ Complete --------->|
   |<----- ACK ---------------|
```

## Error Handling Strategy

### Levels of Error Handling

1. **Network Level**: Connection failures, timeouts
2. **Protocol Level**: Invalid messages, malformed data
3. **File Level**: I/O errors, permission issues
4. **Application Level**: User-facing errors

### Error Recovery

**Transient Errors**:
- Automatic retry with exponential backoff
- Maximum 3 retry attempts
- User notification after final failure

**Permanent Errors**:
- Immediate failure
- Clear error message
- Session cleanup

**Partial Failures**:
- Continue operation where possible
- Log detailed error information
- Notify user of partial success

## Security Considerations

### Current Implementation

**Authentication**: None (local network trust)  
**Authorization**: User approval required  
**Encryption**: None (plaintext transfer)  
**Validation**: Filename sanitization

### Future Enhancements

- [ ] TLS encryption
- [ ] Device authentication
- [ ] File integrity verification
- [ ] Access control lists

## Performance Characteristics

### Memory Usage

**Per Session**: ~10-20 MB
- TCP buffers: 8 MB (4 MB read + 4 MB write)
- Transfer buffers: 256 KB
- Message queues: <1 MB
- UI components: ~5 MB

### CPU Usage

**Discovery**: <1% CPU
**Idle Session**: <1% CPU  
**Active Transfer**: 5-15% CPU (I/O bound)

### Network Bandwidth

**Maximum Theoretical**: 1000 Mbps (Gigabit)  
**Achievable**: 800-950 Mbps (80-120 MB/s)  
**Overhead**: ~5-10% (TCP/IP headers)

## Testing Strategy

### Unit Tests
- File validation
- Protocol message parsing
- Configuration management
- Utility functions

### Integration Tests
- Discovery service
- Session management
- File transfer
- Error handling

### Performance Tests
- Transfer speed benchmarks
- Memory usage profiling
- CPU usage profiling
- Concurrent transfer stress tests

## Deployment

### Build Process
1. Compile for target platform
2. Package with Fyne bundler
3. Include platform-specific icons
4. Generate installer/package

### Platform-Specific
- **Android**: APK via Fyne CLI
- **iOS**: App bundle via Fyne CLI
- **Desktop**: Native executable + bundled resources

## Future Architecture Improvements

### Scalability
- Support multiple simultaneous sessions
- Parallel file transfers
- Chunked parallel transfer for large files

### Reliability
- Resume interrupted transfers
- Checksum verification
- Automatic corruption detection

### Features
- Folder transfer
- Compression
- Encryption
- Transfer history database