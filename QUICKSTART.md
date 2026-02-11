# QuickShare Quick Start

## 1. Run in 1 Minute

```bash
cd /home/sarwar/Desktop/QuickShare
go mod tidy
go run .
```

Start the app on two devices on the same LAN.

## 2. First Connection

1. Open QuickShare on both devices.
2. Wait for discovery list to show the other device.
3. Click/tap the target device on sender.
4. Accept connection request on receiver.
5. Both sides move to connected session view.

## 3. Send a File

1. Click `Send Files`.
2. Choose any file.
3. Watch transfer progress.
4. Receiver gets save confirmation on completion.

## 4. Change Download Location

1. In connected session screen, click `Change Download Location`.
2. Pick a folder.
3. App updates `config/app_config.json` (`download_path`).

## 5. Ports and Firewall

Allow these on local network:

- UDP `9998` (device discovery)
- TCP `9999` (file transfer)

## 6. Validate Installation

Expected startup logs include:

- `Starting QShare Application...`
- `Starting discovery service...`
- `Listening for connections on port 9999`

On app close, discovery/listener shutdown is clean and expected close events are handled.

## 7. Common Issues

### No devices discovered

- Ensure both devices are on the same subnet
- Temporarily disable VPN
- Check router/AP isolation settings
- Verify UDP `9998` is not blocked

### Connection request times out

- Confirm receiver app window is active
- Ensure receiver accepted the request within timeout
- Verify TCP `9999` is reachable

### Receive fails

- Confirm `download_path` exists and is writable
- Try changing download location from UI

## 8. Build Binary

```bash
go build -o quickshare .
./quickshare
```

## 9. Useful Commands

```bash
# Package-level tests used most often in this repo
go test ./internal/network ./internal/file ./internal/utils

# Tail latest log
ls -t logs/transfer_*.log | head -n 1 | xargs tail -n 50
```
