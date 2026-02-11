# QuickShare Architecture

## Overview

QuickShare is a single-process Fyne application with modular packages for UI, networking, file operations, and utilities.

```text
UI (Fyne) -> Network Services (Discovery + Transfer) -> Session Protocol -> File I/O
```

## Runtime Components

### `main.go`

- Initializes logger (`utils.InitLogger`)
- Creates UI app (`ui.NewFileTransferApp`)
- Runs event loop (`app.Run`)

### `internal/ui/app.go`

Coordinates:

- Discovery screen (peer list)
- Session screen (connected state + actions)
- Transfer screen (progress)
- Incoming connection approval dialog
- File picker + folder picker workflows

Core state:

- `activeSession *network.Session`
- `discoveredDevices []*network.Device`
- `monitorRunning bool`

## Network Layer

### Discovery Service (`internal/network/discovery.go`)

- UDP listener bound to `0.0.0.0:9998`
- Broadcast device presence every `2s`
- Maintains in-memory map of active peers
- Removes stale peers after `10s`

Discovery service goroutines:

- `broadcastPresence`
- `listenForDevices`
- `cleanupStaleDevices`

### Transfer Manager (`internal/network/transfer.go`)

- TCP listener on `:9999`
- Accepts incoming sockets
- Creates/owns active `Session`
- Handles connect request/response and file send/receive flow

Performance-related constants:

- `ChunkSize = 256 * 1024`
- `TCPBufferSize = 4 * 1024 * 1024`
- `SetNoDelay(true)`
- `SetKeepAlive(true)` with `30s` period

### Session Protocol (`internal/network/session.go`)

Message frame format:

1. 1 byte: `MessageType`
2. 4 bytes: payload length (`int32`, little-endian)
3. N bytes: payload

Key message types:

- `MsgConnectionRequest`
- `MsgConnectionAccept`
- `MsgConnectionReject`
- `MsgFileOffer`
- `MsgFileAccept`
- `MsgKeepAlive`

Session goroutines:

- `receiveMessages`
- `sendKeepAlive` (every `10s`)

## File Transfer Flow

### Connection establishment

1. Initiator chooses discovered peer.
2. Initiator dials TCP `peerIP:9999`.
3. Initiator sends `MsgConnectionRequest`.
4. Receiver UI prompts user to accept/reject.
5. Receiver sends `MsgConnectionAccept` or `MsgConnectionReject`.
6. Both sides treat session as active.

### File send/receive

1. Sender sends `MsgFileOffer` (name + size JSON).
2. Receiver auto-sends `MsgFileAccept` in monitor loop.
3. Sender writes filename length + filename + file size.
4. Sender streams raw bytes in chunks.
5. Receiver writes bytes to `download_path`.
6. Both sides update progress UI.

## Error Handling and Shutdown

- Timeout reads are treated as expected polling behavior.
- Intentional listener/conn closures are recognized via `isExpectedNetCloseError` in `internal/network/net_errors.go`.
- Discovery `Stop()` is idempotent and safe for repeated calls.
- UI `Run()` cleanup calls:
  - `discoveryService.Stop()`
  - `transferMgr.Close()`

## Configuration Model

Source: `internal/utils/config.go` and `config/app_config.json`

Supported fields:

- `default_port`
- `chunk_size`
- `max_concurrency`
- `download_path`
- `auto_accept`
- `enable_encryption`

Current effective usage:

- `download_path` is used by receive flow and folder picker.
- Network and transfer behavior currently uses code constants (`9998`, `9999`, `256KB`, etc.), not config overrides.
- `auto_accept` and `enable_encryption` are placeholders for future implementation.

## Logging

`internal/utils/logger.go`:

- Creates `logs/transfer_<timestamp>.log`
- Writes info/error/debug events
- Debug mode prints logs to stdout (`isDebugMode = true`)

## Concurrency Summary

Typical active goroutines:

- UI event loop (Fyne)
- Discovery: 3 goroutines
- TCP accept loop: 1 goroutine
- Session: 2 goroutines per active session
- Per transfer: worker goroutine + progress monitor goroutine

## Security Posture (Current)

- Local-network trust model
- No transport encryption at present
- Receiver-side manual approval required for session establishment
- File-path and file-access checks in file/storage utility helpers

## Known Gaps

- Config fields for transfer tuning are not wired into transfer constants yet.
- `MaxConcurrency` constant exists but current transfer path is single-stream.
- No resume/checksum verification in transfer protocol path.
