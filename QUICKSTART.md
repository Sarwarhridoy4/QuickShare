# Quick Start Guide

Get up and running with the File Transfer App in minutes!

## 🚀 Quick Setup (5 minutes)

### Step 1: Create Project Structure

```bash
# Create main directory
mkdir -p filetransfer-app
cd filetransfer-app

# Create all subdirectories
mkdir -p cmd internal/ui internal/network internal/file internal/utils
mkdir -p assets/icons downloads logs config
```

### Step 2: Copy Code Files

Copy each code file from the artifacts to the appropriate directory:

```
cmd/main.go
internal/ui/app.go
internal/network/transfer.go
internal/network/connection.go
internal/file/file_ops.go
internal/file/icon.go
internal/utils/logger.go
internal/utils/config.go
internal/utils/helpers.go
go.mod
```

### Step 3: Initialize and Build

```bash
# Initialize Go module
go mod tidy

# Build the application
go build -o filetransfer cmd/main.go

# Run it!
./filetransfer
```

## 📱 Platform-Specific Instructions

### Linux

```bash
# Install dependencies (Ubuntu/Debian)
sudo apt-get install -y golang gcc libgl1-mesa-dev xorg-dev

# Build
go build -o filetransfer cmd/main.go

# Run
./filetransfer
```

### Windows

```bash
# Install Go from https://golang.org/dl/
# Then build:
go build -o filetransfer.exe cmd/main.go

# Run
filetransfer.exe
```

### macOS

```bash
# Install Go from https://golang.org/dl/ or use Homebrew
brew install go

# Build
go build -o filetransfer cmd/main.go

# Run
./filetransfer
```

### Android

```bash
# Install Fyne CLI
go install fyne.io/fyne/v2/cmd/fyne@latest

# Install Android SDK and NDK
# Then build:
fyne package -os android -appID com.filetransfer.app

# Install the generated APK on your device
```

### iOS

```bash
# Requires macOS with Xcode installed
# Install Fyne CLI
go install fyne.io/fyne/v2/cmd/fyne@latest

# Build
fyne package -os ios -appID com.filetransfer.app
```

## 🎯 First Transfer Test

### On Device 1 (Receiver):

1. Start the app
2. (Optional) Click **"Choose Download Location"** to select where files will be saved
3. Note the IP address (e.g., 192.168.1.100)
4. Click **"Receive File"**
5. Wait for transfer

### On Device 2 (Sender):

1. Start the app
2. Click **"Select File"** and choose a file
3. Click **"Send File"**
4. Enter receiver's IP (192.168.1.100)
5. Click **"Send"**

The file will transfer with real-time progress!

## 🔧 Quick Troubleshooting

### "Cannot connect to receiver"
- Ensure both devices are on the same network
- Check firewall: allow port 9999
- Verify IP address is correct

### "Permission denied"
- On Linux: `chmod +x filetransfer`
- On mobile: Grant storage permissions in settings

### "Module not found"
- Run `go mod tidy`
- Ensure you're in the project root directory

### Slow transfer speeds
- Use wired connection if possible
- Switch to 5GHz Wi-Fi
- Close other network applications

## 📊 Testing Performance

Test with a large file (100MB+):

```bash
# Create test file
dd if=/dev/zero of=testfile.bin bs=1M count=100

# Then transfer using the app
# Monitor speed in the progress bar
```

Expected speeds:
- Gigabit Ethernet: 50-100 MB/s
- Wi-Fi 5GHz: 20-50 MB/s
- Wi-Fi 2.4GHz: 5-20 MB/s

## 🎨 Customization Quick Tips

### Change Download Location

**Via UI:**
```
Click "Choose Download Location" → Select folder → Done!
```

**Via Config:**
Edit `config/app_config.json`:
```json
{
  "download_path": "/path/to/your/downloads"
}
```

### Change Port

Edit `internal/network/transfer.go`:
```go
const Port = 8888  // Change from 9999
```

### Change Download Location

Edit `internal/utils/config.go`:
```go
DownloadPath: "./my-downloads",
```

### Increase Transfer Speed

Edit `internal/network/transfer.go`:
```go
const ChunkSize = 128 * 1024  // Increase from 64KB
```

## 📚 Next Steps

- Read the full README.md for detailed documentation
- Check logs/ directory for debugging
- Customize the UI in internal/ui/app.go
- Add encryption for secure transfers
- Implement auto-discovery for easier connections

## 🆘 Getting Help

1. Check the logs: `cat logs/transfer_*.log`
2. Enable debug mode in `utils/logger.go`
3. Create an issue with log output
4. Review the network configuration

---

**Congratulations!** You now have a working cross-platform file transfer application. Happy transferring! 🎉