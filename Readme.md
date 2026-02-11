# QuickShare

QuickShare is a Go + Fyne desktop/mobile app for transferring files over a local network.

## What It Does

- Discovers peers on the same LAN using UDP broadcast (`9998`)
- Creates a TCP session for transfer (`9999`)
- Requires receiver-side user approval before a session starts
- Sends files with real-time progress UI
- Saves received files to a configurable local folder
- Writes timestamped logs to `logs/` (now gitignored)

## Current Project Layout

```text
QuickShare/
├── main.go
├── internal/
│   ├── ui/
│   │   └── app.go
│   ├── network/
│   │   ├── connection.go
│   │   ├── discovery.go
│   │   ├── net_errors.go
│   │   ├── session.go
│   │   └── transfer.go
│   ├── file/
│   │   ├── file_ops.go
│   │   └── icon.go
│   └── utils/
│       ├── config.go
│       ├── helper.go
│       ├── logger.go
│       └── storage.go
├── config/
│   └── app_config.json
├── ARCHITECTURE.md
├── QUICKSTART.md
└── Readme.md
```

## Requirements

- Go `1.25+`
- A desktop environment for Fyne (Linux/macOS/Windows)
- Two devices on the same local network for peer discovery/transfer

## Run

```bash
go mod tidy
go run .
```

## Build

```bash
go build -o quickshare .
```

## How Transfer Works

1. App starts discovery service (`internal/network/discovery.go`)
2. App starts TCP listener (`internal/network/transfer.go`)
3. Initiator selects a discovered device
4. Receiver accepts/rejects connection request
5. Sender offers file metadata, receiver auto-accepts file offer
6. File bytes stream over active TCP connection

## Configuration

Config file: `config/app_config.json`

```json
{
  "default_port": 9999,
  "chunk_size": 65536,
  "max_concurrency": 4,
  "download_path": "./downloads",
  "auto_accept": false,
  "enable_encryption": false
}
```

Notes about current behavior:

- `download_path` is actively used by the UI receive flow
- `default_port`, `chunk_size`, and `max_concurrency` exist in config, but transfer currently uses constants in `internal/network/transfer.go`
- `auto_accept` and `enable_encryption` are present but not yet implemented

## Network/Performance Defaults (Code)

From `internal/network/transfer.go` and `internal/network/discovery.go`:

- Discovery port: `9998` (UDP broadcast)
- Transfer port: `9999` (TCP)
- Transfer chunk size: `256KB`
- TCP read/write buffer: `4MB`
- TCP keepalive period: `30s`
- Discovery broadcast interval: `2s`
- Stale device timeout: `10s`

## Logging

- Logs are created at startup in `logs/transfer_YYYY-MM-DD_HH-MM-SS.log`
- Console logging is enabled by debug mode in `internal/utils/logger.go`
- `logs/` is ignored by Git via `.gitignore`

## Shutdown Behavior

Expected listener closure during app shutdown is handled without noisy network-close errors:

- `internal/network/net_errors.go`
- `internal/network/discovery.go`
- `internal/network/transfer.go`

## Troubleshooting

- No peers found:
  - Verify both devices are on the same LAN
  - Allow UDP `9998` and TCP `9999` through firewall
  - Disable VPN for testing
- Connection fails:
  - Confirm receiver accepted connection dialog
  - Ensure remote app is still open
- File receive fails:
  - Verify write access to configured `download_path`

## Development

```bash
go test ./internal/network ./internal/file ./internal/utils
```

(Full `go test ./...` may require normal Go build cache access depending on environment sandboxing.)
