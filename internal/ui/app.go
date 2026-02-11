package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/Sarwarhridoy4/QuickShare/internal/file"
	"github.com/Sarwarhridoy4/QuickShare/internal/network"
	"github.com/Sarwarhridoy4/QuickShare/internal/utils"
)

type FileTransferApp struct {
	app               fyne.App
	window            fyne.Window
	transferMgr       *network.TransferManager
	discoveryService  *network.DiscoveryService
	config            *utils.Config
	
	// UI Components - Discovery Screen
	discoveryView     *fyne.Container
	deviceList        *widget.List
	discoveredDevices []*network.Device
	refreshBtn        *widget.Button
	myDeviceLabel     *widget.Label
	
	// UI Components - Session Screen
	sessionView       *fyne.Container
	remoteDeviceLabel *widget.Label
	statusLabel       *widget.Label
	sendFileBtn       *widget.Button
	disconnectBtn     *widget.Button
	
	// UI Components - Transfer Screen
	transferView      *fyne.Container
	currentFileLabel  *widget.Label
	progressBar       *widget.ProgressBar
	speedLabel        *widget.Label
	downloadPathLabel *widget.Label
	
	// State
	activeSession     *network.Session
	selectedFiles     []string
	currentScreen     string
	monitorRunning    bool
}

func NewFileTransferApp() *FileTransferApp {
	a := app.NewWithID("com.filetransfer.app")
	w := a.NewWindow("File Transfer - Discovery")
	
	icon := file.LoadIcon()
	if icon != nil {
		a.SetIcon(icon)
		w.SetIcon(icon)
	}

	cfg, err := utils.LoadConfig()
	if err != nil {
		utils.LogError("Failed to load config", err)
		cfg = &utils.Config{}
		*cfg = utils.GetDefaultConfig()
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "Device"
	}

	fta := &FileTransferApp{
		app:               a,
		window:            w,
		transferMgr:       network.NewTransferManager(),
		config:            cfg,
		discoveredDevices: make([]*network.Device, 0),
		selectedFiles:     make([]string, 0),
		currentScreen:     "discovery",
	}

	// Initialize discovery service
	fta.discoveryService = network.NewDiscoveryService(hostname, fta.onDevicesUpdated)
	
	// Setup connection request handler
	fta.transferMgr.SetSessionRequestHandler(fta.handleConnectionRequest)
	
	fta.setupUI()
	fta.startServices()
	
	return fta
}

func (fta *FileTransferApp) setupUI() {
	fta.setupDiscoveryView()
	fta.setupSessionView()
	fta.setupTransferView()
	
	fta.showScreen("discovery")
	
	isMobile := runtime.GOOS == "android" || runtime.GOOS == "ios"
	if isMobile {
		fta.window.Resize(fyne.NewSize(360, 640))
	} else {
		fta.window.Resize(fyne.NewSize(500, 600))
	}
	
	fta.window.CenterOnScreen()
}

func (fta *FileTransferApp) setupDiscoveryView() {
	fta.myDeviceLabel = widget.NewLabel("Searching for devices...")
	fta.myDeviceLabel.Wrapping = fyne.TextWrapWord
	
	fta.deviceList = widget.NewList(
		func() int { return len(fta.discoveredDevices) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(nil),
				widget.NewLabel("Device Name"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(fta.discoveredDevices) {
				return
			}
			device := fta.discoveredDevices[id]
			box := obj.(*fyne.Container)
			label := box.Objects[1].(*widget.Label)
			label.SetText(fmt.Sprintf("%s\n%s", device.Name, device.IP))
		},
	)
	
	fta.deviceList.OnSelected = func(id widget.ListItemID) {
		if id >= len(fta.discoveredDevices) {
			return
		}
		fta.connectToDevice(fta.discoveredDevices[id])
	}
	
	fta.refreshBtn = widget.NewButton("Refresh", func() {
		fta.discoveredDevices = fta.discoveryService.GetActiveDevices()
		fta.deviceList.Refresh()
	})
	
	header := widget.NewLabelWithStyle("Available Devices", 
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	
	fta.discoveryView = container.NewBorder(
		container.NewVBox(
			header,
			fta.myDeviceLabel,
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			fta.refreshBtn,
		),
		nil, nil,
		fta.deviceList,
	)
}

func (fta *FileTransferApp) setupSessionView() {
	fta.remoteDeviceLabel = widget.NewLabel("Not connected")
	fta.statusLabel = widget.NewLabel("Ready")
	fta.downloadPathLabel = widget.NewLabel(fmt.Sprintf("Save to: %s", fta.config.DownloadPath))
	
	fta.sendFileBtn = widget.NewButton("Send Files", fta.handleSelectAndSendFiles)
	fta.sendFileBtn.Importance = widget.HighImportance
	
	chooseDownloadBtn := widget.NewButton("Change Download Location", fta.handleChooseDownloadLocation)
	
	fta.disconnectBtn = widget.NewButton("Disconnect", fta.handleDisconnect)
	fta.disconnectBtn.Importance = widget.DangerImportance
	
	header := widget.NewLabelWithStyle("Connected", 
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	
	fta.sessionView = container.NewVBox(
		header,
		widget.NewSeparator(),
		fta.remoteDeviceLabel,
		fta.statusLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("File Transfer", 
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		fta.sendFileBtn,
		widget.NewLabel("Files will be saved to:"),
		fta.downloadPathLabel,
		chooseDownloadBtn,
		widget.NewSeparator(),
		fta.disconnectBtn,
	)
}

func (fta *FileTransferApp) setupTransferView() {
	fta.currentFileLabel = widget.NewLabel("No active transfer")
	fta.progressBar = widget.NewProgressBar()
	fta.speedLabel = widget.NewLabel("Speed: 0 MB/s")
	
	header := widget.NewLabelWithStyle("File Transfer", 
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	
	fta.transferView = container.NewVBox(
		header,
		widget.NewSeparator(),
		fta.currentFileLabel,
		fta.progressBar,
		fta.speedLabel,
		widget.NewSeparator(),
		widget.NewButton("Cancel", func() {
			fta.showScreen("session")
		}),
	)
}

func (fta *FileTransferApp) showScreen(screen string) {
	fta.currentScreen = screen
	
	switch screen {
	case "discovery":
		fta.window.SetTitle("File Transfer - Discovery")
		fta.window.SetContent(fta.discoveryView)
	case "session":
		fta.window.SetTitle("File Transfer - Connected")
		fta.window.SetContent(fta.sessionView)
	case "transfer":
		fta.window.SetTitle("File Transfer - In Progress")
		fta.window.SetContent(fta.transferView)
	}
}

func (fta *FileTransferApp) startServices() {
	// Start discovery
	if err := fta.discoveryService.Start(); err != nil {
		utils.LogError("Failed to start discovery service", err)
		dialog.ShowError(err, fta.window)
	}
	
	// Start listening for connections
	if err := fta.transferMgr.StartListening(); err != nil {
		utils.LogError("Failed to start listening", err)
		dialog.ShowError(err, fta.window)
	}
	
	// Update device label
	localIP := network.GetLocalIP()
	hostname, _ := os.Hostname()
	fta.myDeviceLabel.SetText(fmt.Sprintf("Your Device: %s\nIP: %s\n\nWaiting for devices...", hostname, localIP))
}

func (fta *FileTransferApp) onDevicesUpdated(devices []*network.Device) {
	fta.discoveredDevices = devices
	fta.deviceList.Refresh()
	
	if len(devices) == 0 {
		fta.myDeviceLabel.SetText(fta.myDeviceLabel.Text + "\n\nNo devices found")
	}
}

func (fta *FileTransferApp) connectToDevice(device *network.Device) {
	utils.Log(fmt.Sprintf("User initiated connection to %s", device.Name))
	
	progressDialog := dialog.NewCustom("Connecting", "Cancel", 
		widget.NewLabel(fmt.Sprintf("Connecting to %s...\nPlease wait...", device.Name)), 
		fta.window)
	progressDialog.Show()
	
	go func() {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "Unknown Device"
		}
		
		session, err := fta.transferMgr.ConnectToDevice(device, hostname)
		
		// Close progress dialog
		progressDialog.Hide()
		
		if err != nil {
			utils.LogError("Connection failed", err)
			dialog.ShowError(fmt.Errorf("Connection failed: %v", err), fta.window)
			return
		}
		
		utils.Log("Connection successful, setting up UI")
		
		fta.activeSession = session
		fta.remoteDeviceLabel.SetText(fmt.Sprintf("Connected to: %s (%s)", device.Name, device.IP))
		fta.statusLabel.SetText("Connected - Ready to transfer files")
		
		dialog.ShowInformation("Connected", 
			fmt.Sprintf("Successfully connected to %s!\n\nYou can now send files.", device.Name), 
			fta.window)
		
		fta.showScreen("session")
		
		// Start monitoring for incoming files only if not already running
		if !fta.monitorRunning {
			fta.monitorRunning = true
			utils.Log("Starting session monitor (sender side)")
			go fta.monitorSession()
		} else {
			utils.Log("Session monitor already running")
		}
	}()
}

func (fta *FileTransferApp) handleConnectionRequest(session *network.Session, req *network.ConnectionRequest) bool {
	utils.Log(fmt.Sprintf("Handling connection request from %s", req.DeviceName))
	
	// Channel to communicate user's decision
	approved := make(chan bool, 1)
	
	// Show dialog on UI thread
	fta.window.Canvas().Content().Refresh()
	
	go func() {
		dialog.ShowConfirm("Connection Request", 
			fmt.Sprintf("%s (%s) wants to connect\n\nAccept connection?", 
				req.DeviceName, req.DeviceType),
			func(accept bool) {
				utils.Log(fmt.Sprintf("User decision: %v", accept))
				approved <- accept
			}, fta.window)
	}()
	
	// Wait for user decision with timeout
	select {
	case accepted := <-approved:
		if accepted {
			utils.Log("Connection accepted by user")
			fta.activeSession = session
			fta.remoteDeviceLabel.SetText(fmt.Sprintf("Connected to: %s", req.DeviceName))
			fta.statusLabel.SetText("Connected - Ready to transfer files")
			fta.showScreen("session")
			
			// Start monitoring session only if not already running
			if !fta.monitorRunning {
				fta.monitorRunning = true
				utils.Log("Starting session monitor (receiver side)")
				go fta.monitorSession()
			} else {
				utils.Log("Session monitor already running")
			}
			
			return true
		} else {
			utils.Log("Connection rejected by user")
			return false
		}
	case <-time.After(60 * time.Second):
		utils.Log("Connection request dialog timeout")
		return false
	}
}

func (fta *FileTransferApp) monitorSession() {
	if fta.activeSession == nil {
		utils.Log("Monitor called but no active session")
		fta.monitorRunning = false
		return
	}
	
	utils.Log("Session monitor started")
	msgChan := fta.activeSession.GetMessageChannel()
	
	defer func() {
		utils.Log("Session monitor stopped")
		fta.monitorRunning = false
	}()
	
	for msg := range msgChan {
		switch msg.Type {
		case network.MsgFileOffer:
			utils.Log("Received file offer")
			
			// Parse the offer
			var offer network.FileOffer
			if err := json.Unmarshal(msg.Payload, &offer); err != nil {
				utils.LogError("Failed to parse file offer", err)
				continue
			}
			
			utils.Log(fmt.Sprintf("File offer: %s (%.2f MB)", offer.FileName, float64(offer.FileSize)/1024/1024))
			
			// Send accept message
			if err := fta.activeSession.AcceptFile(); err != nil {
				utils.LogError("Failed to accept file", err)
				dialog.ShowError(fmt.Errorf("Failed to accept file: %v", err), fta.window)
				continue
			}
			
			utils.Log("File accepted, starting receive")
			// Start receiving
			go fta.receiveFile()
			
		case network.MsgKeepAlive:
			// Just a keepalive, ignore it
			utils.LogDebug("Received keepalive")
			
		case network.MsgConnectionReject:
			utils.Log("Connection rejected by remote")
			dialog.ShowError(fmt.Errorf("connection closed by remote device"), fta.window)
			fta.handleDisconnect()
			return
			
		default:
			utils.LogDebug(fmt.Sprintf("Received message type: %v", msg.Type))
		}
	}
}

func (fta *FileTransferApp) handleSelectAndSendFiles() {
	// Check if session is still active
	if fta.activeSession == nil {
		dialog.ShowError(fmt.Errorf("no active connection"), fta.window)
		return
	}
	
	session := fta.activeSession
	if session.State != network.SessionActive {
		dialog.ShowError(fmt.Errorf("session is not active"), fta.window)
		fta.handleDisconnect()
		return
	}
	
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, fta.window)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()
		
		filePath := reader.URI().Path()
		utils.Log(fmt.Sprintf("User selected file: %s", filePath))
		fta.sendFile(filePath)
	}, fta.window)
}

func (fta *FileTransferApp) sendFile(filePath string) {
	if fta.activeSession == nil {
		dialog.ShowError(fmt.Errorf("no active connection"), fta.window)
		return
	}
	
	fileName := filepath.Base(filePath)
	fta.currentFileLabel.SetText(fmt.Sprintf("Sending: %s", fileName))
	fta.progressBar.SetValue(0)
	fta.showScreen("transfer")
	
	go func() {
		progressChan := make(chan float64, 10)
		done := make(chan error, 1)
		
		startTime := utils.Now()
		
		go func() {
			err := fta.transferMgr.SendFile(filePath, progressChan)
			done <- err
		}()
		
		// Monitor progress
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		var lastProgress float64
		
		for {
			select {
			case progress := <-progressChan:
				lastProgress = progress
				fta.progressBar.SetValue(progress)
				
				// Calculate speed
				elapsed := utils.Since(startTime).Seconds()
				if elapsed > 0 {
					fileInfo, err := os.Stat(filePath)
					if err == nil && fileInfo != nil {
						bytesTransferred := int64(progress * float64(fileInfo.Size()))
						speed := float64(bytesTransferred) / elapsed / 1024 / 1024
						fta.speedLabel.SetText(fmt.Sprintf("Speed: %.2f MB/s", speed))
					}
				}
				
			case <-ticker.C:
				// Update UI periodically even without new progress
				if lastProgress > 0 && lastProgress < 1.0 {
					elapsed := utils.Since(startTime).Seconds()
					if elapsed > 0 {
						fileInfo, err := os.Stat(filePath)
						if err == nil && fileInfo != nil {
							bytesTransferred := int64(lastProgress * float64(fileInfo.Size()))
							speed := float64(bytesTransferred) / elapsed / 1024 / 1024
							fta.speedLabel.SetText(fmt.Sprintf("Speed: %.2f MB/s (%.0f%%)", speed, lastProgress*100))
						}
					}
				}
				
			case err := <-done:
				if err != nil {
					dialog.ShowError(fmt.Errorf("Transfer failed: %v", err), fta.window)
					fta.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
					utils.LogError("File send failed", err)
				} else {
					fta.progressBar.SetValue(1.0)
					dialog.ShowInformation("Success", "File sent successfully!", fta.window)
					fta.statusLabel.SetText("File sent successfully")
					utils.Log("File sent successfully")
				}
				fta.showScreen("session")
				return
			}
		}
	}()
}

func (fta *FileTransferApp) receiveFile() {
	utils.Log("Starting file receive process")
	
	fta.currentFileLabel.SetText("Receiving file...")
	fta.progressBar.SetValue(0)
	fta.speedLabel.SetText("Speed: 0 MB/s")
	fta.showScreen("transfer")
	
	go func() {
		progressChan := make(chan float64, 10)
		done := make(chan string, 1)
		errChan := make(chan error, 1)
		
		startTime := utils.Now()
		
		go func() {
			utils.Log("Calling ReceiveFile...")
			filename, err := fta.transferMgr.ReceiveFile(fta.config.DownloadPath, progressChan)
			if err != nil {
				utils.LogError("ReceiveFile error", err)
				errChan <- err
			} else {
				utils.Log(fmt.Sprintf("File received successfully: %s", filename))
				done <- filename
			}
		}()
		
		// Monitor progress
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case progress := <-progressChan:
				fta.progressBar.SetValue(progress)
				
				elapsed := utils.Since(startTime).Seconds()
				if elapsed > 0 {
					// Estimate speed based on progress
					speed := progress / elapsed * 100 // Rough estimate
					fta.speedLabel.SetText(fmt.Sprintf("Speed: %.2f MB/s (%.0f%%)", speed, progress*100))
				}
				
			case <-ticker.C:
				// Keep UI responsive
				
			case filename := <-done:
				fta.progressBar.SetValue(1.0)
				dialog.ShowInformation("Success", 
					fmt.Sprintf("File received:\n%s", filename), fta.window)
				fta.statusLabel.SetText("File received successfully")
				utils.Log("Receive complete, returning to session screen")
				fta.showScreen("session")
				return
				
			case err := <-errChan:
				dialog.ShowError(fmt.Errorf("Receive failed: %v", err), fta.window)
				fta.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				utils.LogError("Receive failed", err)
				fta.showScreen("session")
				return
			}
		}
	}()
}

func (fta *FileTransferApp) handleChooseDownloadLocation() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		
		newPath := uri.Path()
		fta.config.DownloadPath = newPath
		fta.downloadPathLabel.SetText(fmt.Sprintf("Save to: %s", newPath))
		
		if err := utils.SaveConfig(fta.config); err != nil {
			dialog.ShowError(err, fta.window)
		}
	}, fta.window)
}

func (fta *FileTransferApp) handleDisconnect() {
	utils.Log("User initiated disconnect")
	
	fta.monitorRunning = false
	
	if fta.activeSession != nil {
		fta.activeSession.Close()
		fta.activeSession = nil
		utils.Log("Session closed")
	}
	
	fta.transferMgr.CloseSession()
	fta.statusLabel.SetText("Disconnected")
	fta.remoteDeviceLabel.SetText("Not connected")
	
	// Return to discovery screen
	fta.showScreen("discovery")
	
	// Refresh device list
	fta.discoveredDevices = fta.discoveryService.GetActiveDevices()
	fta.deviceList.Refresh()
}

func (fta *FileTransferApp) Run() {
	fta.window.ShowAndRun()
	
	// Cleanup
	fta.discoveryService.Stop()
	fta.transferMgr.Close()
}
