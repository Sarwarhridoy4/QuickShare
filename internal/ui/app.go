package ui

import (
	"github.com/Sarwarhridoy4/QuickShare/internal/file"
	"github.com/Sarwarhridoy4/QuickShare/internal/network"
	"github.com/Sarwarhridoy4/QuickShare/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
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
}

func NewFileTransferApp() *FileTransferApp {
	a := app.NewWithID("com.filetransfer.app")
	w := a.NewWindow("File Transfer - Discovery")
	
	icon := file.LoadIcon()
	if icon != nil {
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
	dialog.ShowInformation("Connecting", 
		fmt.Sprintf("Connecting to %s...", device.Name), fta.window)
	
	go func() {
		hostname, _ := os.Hostname()
		session, err := fta.transferMgr.ConnectToDevice(device, hostname)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Connection failed: %w", err), fta.window)
			return
		}
		
		fta.activeSession = session
		fta.remoteDeviceLabel.SetText(fmt.Sprintf("Connected to: %s (%s)", device.Name, device.IP))
		fta.statusLabel.SetText("Connected - Ready to transfer files")
		
		dialog.ShowInformation("Connected", 
			fmt.Sprintf("Successfully connected to %s", device.Name), fta.window)
		
		fta.showScreen("session")
		
		// Start monitoring for incoming files
		go fta.monitorSession()
	}()
}

func (fta *FileTransferApp) handleConnectionRequest(session *network.Session, req *network.ConnectionRequest) bool {
	approved := make(chan bool, 1)
	
	fta.window.Canvas().Content().Show()
	
	dialog.ShowConfirm("Connection Request", 
		fmt.Sprintf("%s wants to connect\n\nAccept connection?", req.DeviceName),
		func(accept bool) {
			approved <- accept
		}, fta.window)
	
	accepted := <-approved
	
	if accepted {
		fta.activeSession = session
		fta.remoteDeviceLabel.SetText(fmt.Sprintf("Connected to: %s", req.DeviceName))
		fta.statusLabel.SetText("Connected - Ready to transfer files")
		fta.showScreen("session")
		
		// Start monitoring session
		go fta.monitorSession()
	}
	
	return accepted
}

func (fta *FileTransferApp) monitorSession() {
	if fta.activeSession == nil {
		return
	}
	
	msgChan := fta.activeSession.GetMessageChannel()
	
	for msg := range msgChan {
		switch msg.Type {
		case network.MsgFileOffer:
			// Incoming file - automatically start receiving
			go fta.receiveFile()
		}
	}
}

func (fta *FileTransferApp) handleSelectAndSendFiles() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		
		filePath := reader.URI().Path()
		fta.sendFile(filePath)
	}, fta.window)
}

func (fta *FileTransferApp) sendFile(filePath string) {
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
		for {
			select {
			case progress := <-progressChan:
				fta.progressBar.SetValue(progress)
				
				// Calculate speed
				elapsed := utils.Since(startTime).Seconds()
				if elapsed > 0 {
					fileInfo, _ := os.Stat(filePath)
					if fileInfo != nil {
						bytesTransferred := int64(progress * float64(fileInfo.Size()))
						speed := float64(bytesTransferred) / elapsed / 1024 / 1024
						fta.speedLabel.SetText(fmt.Sprintf("Speed: %.2f MB/s", speed))
					}
				}
				
			case err := <-done:
				if err != nil {
					dialog.ShowError(err, fta.window)
					fta.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				} else {
					fta.progressBar.SetValue(1.0)
					dialog.ShowInformation("Success", "File sent successfully!", fta.window)
					fta.statusLabel.SetText("File sent successfully")
				}
				fta.showScreen("session")
				return
			}
		}
	}()
}

func (fta *FileTransferApp) receiveFile() {
	fta.currentFileLabel.SetText("Receiving file...")
	fta.progressBar.SetValue(0)
	fta.showScreen("transfer")
	
	go func() {
		progressChan := make(chan float64, 10)
		done := make(chan string, 1)
		errChan := make(chan error, 1)
		
		startTime := utils.Now()
		var fileSize int64
		
		go func() {
			filename, err := fta.transferMgr.ReceiveFile(fta.config.DownloadPath, progressChan)
			if err != nil {
				errChan <- err
			} else {
				done <- filename
			}
		}()
		
		for {
			select {
			case progress := <-progressChan:
				fta.progressBar.SetValue(progress)
				
				elapsed := utils.Since(startTime).Seconds()
				if elapsed > 0 && fileSize > 0 {
					bytesTransferred := int64(progress * float64(fileSize))
					speed := float64(bytesTransferred) / elapsed / 1024 / 1024
					fta.speedLabel.SetText(fmt.Sprintf("Speed: %.2f MB/s", speed))
				}
				
			case filename := <-done:
				fta.progressBar.SetValue(1.0)
				dialog.ShowInformation("Success", 
					fmt.Sprintf("File received:\n%s", filename), fta.window)
				fta.statusLabel.SetText("File received successfully")
				fta.showScreen("session")
				return
				
			case err := <-errChan:
				dialog.ShowError(err, fta.window)
				fta.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
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
	if fta.activeSession != nil {
		fta.activeSession.Close()
		fta.activeSession = nil
	}
	
	fta.transferMgr.CloseSession()
	fta.statusLabel.SetText("Disconnected")
	fta.showScreen("discovery")
}

func (fta *FileTransferApp) Run() {
	fta.window.ShowAndRun()
	
	// Cleanup
	fta.discoveryService.Stop()
	fta.transferMgr.Close()
}