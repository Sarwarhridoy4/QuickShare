# Cross-Platform File Transfer Application

A high-speed, cross-platform file transfer application built with Go and Fyne, supporting desktop (Windows, macOS, Linux) and mobile platforms (Android, iOS).

## Features

- 🚀 **High-Speed Transfer**: Optimized TCP transfer with 64KB chunks
- 📱 **Mobile-First Design**: Responsive UI that adapts to any screen size
- 🖥️ **Cross-Platform**: Works on Windows, macOS, Linux, Android, and iOS
- ⚡ **Non-Blocking Operations**: Multithreaded transfers using Go routines
- 📊 **Real-Time Progress**: Live transfer progress and speed indicators
- 🔒 **Reliable**: Robust error handling and automatic retries
- 📝 **Logging**: Comprehensive logging for debugging and monitoring

## Project Structure

```
filetransfer-app/
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── ui/
│   │   └── app.go              # Main UI logic
│   ├── network/
│   │   ├── transfer.go         # File transfer implementation
│   │   └── connection.go       # Network utilities
│   ├── file/
│   │   ├── file_ops.go         # File operations
│   │   └── icon.go             # Platform-specific icons
│   └── utils/
│       ├── logger.go           # Logging system
│       ├── config.go           # Configuration management
│       └── helpers.go          # Helper functions
├── assets/
│   └── icons/                  # Platform-specific icons
├── downloads/                  # Default download directory
├── logs/                       # Application logs
├── go.mod
└── README.md
```

## Prerequisites

- Go 1.21 or higher
- For desktop: Standard development tools for your platform
- For Android: Android SDK and NDK
- For iOS: Xcode and iOS SDK

## Installation

### 1. Clone or create the project

```bash
mkdir -p filetransfer-app
cd filetransfer-app
```

### 2. Initialize Go module

```bash
go mod init filetransfer
go mod tidy
```

### 3. Install dependencies

```bash
go get fyne.io/fyne/v2
```

## Building

### Desktop (All Platforms)

```bash
# Linux
go build -o filetransfer cmd/main.go

# Windows
go build -o filetransfer.exe cmd/main.go

# macOS
go build -o filetransfer cmd/main.go
```

### Using Fyne CLI (Recommended for packaging)

```bash
# Install Fyne CLI
go install fyne.io/fyne/v2/cmd/fyne@latest

# Build for current platform
fyne package -os [linux|windows|darwin] -icon assets/icons/icon.png

# Build for mobile
fyne package -os android -appID com.filetransfer.app
fyne package -os ios -appID com.filetransfer.app
```

## Usage

### Sending a File

1. Launch the application on both devices
2. Note the IP address displayed on the receiving device
3. On the sending device:
   - Click "Select File" and choose a file
   - Click "Send File"
   - Enter the recipient's IP address
   - Click "Send"

### Receiving a File

1. Launch the application
2. Note your IP address displayed in the app
3. Click "Receive File"
4. The app will wait for incoming transfers
5. Files are saved in the `downloads/` directory

## Configuration

The application creates a configuration file at `config/app_config.json` with the following options:

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

## Network Requirements

- Both devices must be on the same network or have direct connectivity
- Port 9999 must be open and not blocked by firewalls
- For best performance, use a wired connection or 5GHz Wi-Fi

## Performance

- Transfer speed depends on network bandwidth
- Optimized for LAN transfers (typically 50-100 MB/s on gigabit networks)
- Uses 64KB chunks for optimal throughput
- Non-blocking operations ensure UI remains responsive

## Troubleshooting

### Connection Issues

- Verify both devices are on the same network
- Check firewall settings (allow port 9999)
- Ensure IP address is entered correctly
- Try disabling VPN if active

### Transfer Failures

- Check available disk space on receiving device
- Verify file permissions
- Check logs in `logs/` directory
- Ensure network is stable

### Mobile-Specific Issues

- Grant necessary permissions (storage, network)
- Keep app in foreground during transfers
- Disable battery optimization for the app

## Development

### Running Tests

```bash
go test ./...
```

### Debugging

Enable debug mode in `utils/logger.go`:

```go
isDebugMode = true
```

Check logs in the `logs/` directory for detailed information.

## Future Enhancements

- [ ] End-to-end encryption
- [ ] Resume interrupted transfers
- [ ] QR code for easy IP sharing
- [ ] Peer discovery (mDNS/Bonjour)
- [ ] Transfer history
- [ ] Batch file transfers
- [ ] Compression support
- [ ] Dark/Light theme toggle

## License

MIT License - Feel free to use and modify

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## Support

For issues and questions, please check the logs directory and create an issue with relevant log files.