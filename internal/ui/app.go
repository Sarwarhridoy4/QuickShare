package ui

import (
	"github.com/Sarwarhridoy4/QuickShare/internal/file"
	"github.com/Sarwarhridoy4/QuickShare/internal/network"
	"github.com/Sarwarhridoy4/QuickShare/internal/utils"
	"fmt"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type FileTransferApp struct {
	app             fyne.App
	window          fyne.Window
	transferMgr     *network.TransferManager
	statusLabel     *widget.Label
	progressBar     *widget.ProgressBar
	filePathLabel   *widget.Label
	ipLabel         *widget.Label
	downloadPathLabel *widget.Label
	config          *utils.Config
}

func NewFileTransferApp() *FileTransferApp {
	a := app.NewWithID("com.filetransfer.app")
	w := a.NewWindow("File Transfer")
	
	// Load platform-specific icon
	icon := file.LoadIcon()
	if icon != nil {
		w.SetIcon(icon)
	}

	// Load configuration
	cfg, err := utils.LoadConfig()
	if err != nil {
		utils.LogError("Failed to load config, using defaults", err)
		cfg = &utils.Config{}
		*cfg = utils.GetDefaultConfig()
	}

	fta := &FileTransferApp{
		app:               a,
		window:            w,
		transferMgr:       network.NewTransferManager(),
		statusLabel:       widget.NewLabel("Ready"),
		progressBar:       widget.NewProgressBar(),
		filePathLabel:     widget.NewLabel("No file selected"),
		ipLabel:           widget.NewLabel("IP: Detecting..."),
		downloadPathLabel: widget.NewLabel(fmt.Sprintf("Download: %s", cfg.DownloadPath)),
		config:            cfg,
	}

	fta.setupUI()
	fta.detectIP()
	
	return fta
}

func (fta *FileTransferApp) setupUI() {
	// Detect if mobile
	isMobile := runtime.GOOS == "android" || runtime.GOOS == "ios"
	
	// Create UI elements
	sendBtn := widget.NewButton("Send File", fta.handleSendFile)
	receiveBtn := widget.NewButton("Receive File", fta.handleReceiveFile)
	selectFileBtn := widget.NewButton("Select File", fta.handleSelectFile)
	chooseDownloadBtn := widget.NewButton("Choose Download Location", fta.handleChooseDownloadLocation)
	
	// Apply mobile-specific styling if needed
	if isMobile {
		sendBtn.Importance = widget.HighImportance
		receiveBtn.Importance = widget.HighImportance
	}
	
	// Connection status section
	connectionBox := container.NewVBox(
		widget.NewLabelWithStyle("Connection Info", 
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		fta.ipLabel,
		fta.statusLabel,
	)
	
	// File selection section
	fileBox := container.NewVBox(
		widget.NewLabelWithStyle("File Selection", 
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		selectFileBtn,
		fta.filePathLabel,
	)
	
	// Download location section
	downloadBox := container.NewVBox(
		widget.NewLabelWithStyle("Download Settings", 
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		chooseDownloadBtn,
		fta.downloadPathLabel,
	)
	
	// Transfer controls
	controlBox := container.NewHBox(
		sendBtn,
		receiveBtn,
	)
	
	// Progress section
	progressBox := container.NewVBox(
		widget.NewLabelWithStyle("Transfer Progress", 
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		fta.progressBar,
	)
	
	// Main layout
	contentBox := container.NewVBox(
		connectionBox,
		widget.NewSeparator(),
		fileBox,
		widget.NewSeparator(),
		downloadBox,
		widget.NewSeparator(),
		controlBox,
		widget.NewSeparator(),
		progressBox,
	)
	
	var content fyne.CanvasObject
	// Wrap in scroll container for mobile
	if isMobile {
		content = container.NewScroll(contentBox)
	} else {
		content = contentBox
	}
	
	fta.window.SetContent(content)
	
	// Set appropriate window size
	if isMobile {
		fta.window.Resize(fyne.NewSize(360, 640))
	} else {
		fta.window.Resize(fyne.NewSize(600, 500))
	}
	
	fta.window.CenterOnScreen()
}

func (fta *FileTransferApp) detectIP() {
	go func() {
		ip := network.GetLocalIP()
		fta.ipLabel.SetText(fmt.Sprintf("IP: %s", ip))
		utils.Log(fmt.Sprintf("Local IP detected: %s", ip))
	}()
}

func (fta *FileTransferApp) handleSelectFile() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, fta.window)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()
		
		fta.filePathLabel.SetText(reader.URI().Path())
		utils.Log(fmt.Sprintf("File selected: %s", reader.URI().Path()))
	}, fta.window)
}

func (fta *FileTransferApp) handleSendFile() {
	filePath := fta.filePathLabel.Text
	if filePath == "No file selected" {
		dialog.ShowInformation("No File", "Please select a file first", fta.window)
		return
	}
	
	// Show input dialog for recipient IP
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Enter recipient IP address")
	
	dialog.ShowForm("Send File", "Send", "Cancel", []*widget.FormItem{
		{Text: "Recipient IP:", Widget: entry},
	}, func(confirmed bool) {
		if !confirmed || entry.Text == "" {
			return
		}
		
		fta.statusLabel.SetText("Sending...")
		fta.progressBar.SetValue(0)
		
		// Start transfer in goroutine
		go func() {
			progressChan := make(chan float64, 10)
			errChan := make(chan error, 1)
			
			go func() {
				err := fta.transferMgr.SendFile(filePath, entry.Text, progressChan)
				errChan <- err
			}()
			
			// Update progress
			for {
				select {
				case progress := <-progressChan:
					fta.progressBar.SetValue(progress)
				case err := <-errChan:
					if err != nil {
						fta.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
						dialog.ShowError(err, fta.window)
					} else {
						fta.statusLabel.SetText("Transfer complete!")
						fta.progressBar.SetValue(1.0)
						dialog.ShowInformation("Success", "File sent successfully", fta.window)
					}
					return
				}
			}
		}()
	}, fta.window)
}

func (fta *FileTransferApp) handleReceiveFile() {
	fta.statusLabel.SetText("Listening for incoming files...")
	fta.progressBar.SetValue(0)
	
	// Start receiving in goroutine
	go func() {
		progressChan := make(chan float64, 10)
		errChan := make(chan error, 1)
		filenameChan := make(chan string, 1)
		
		go func() {
			filename, err := fta.transferMgr.ReceiveFile(fta.config.DownloadPath, progressChan)
			filenameChan <- filename
			errChan <- err
		}()
		
		// Update progress
		for {
			select {
			case progress := <-progressChan:
				fta.progressBar.SetValue(progress)
			case err := <-errChan:
				filename := <-filenameChan
				if err != nil {
					fta.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
					dialog.ShowError(err, fta.window)
				} else {
					fta.statusLabel.SetText("Transfer complete!")
					fta.progressBar.SetValue(1.0)
					dialog.ShowInformation("Success", 
						fmt.Sprintf("File received: %s", filename), fta.window)
				}
				return
			}
		}
	}()
}

func (fta *FileTransferApp) handleChooseDownloadLocation() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, fta.window)
			return
		}
		if uri == nil {
			return
		}
		
		newPath := uri.Path()
		fta.config.DownloadPath = newPath
		fta.downloadPathLabel.SetText(fmt.Sprintf("Download: %s", newPath))
		
		// Save configuration
		if err := utils.SaveConfig(fta.config); err != nil {
			utils.LogError("Failed to save config", err)
			dialog.ShowError(fmt.Errorf("failed to save download location: %w", err), fta.window)
		} else {
			utils.Log(fmt.Sprintf("Download location changed to: %s", newPath))
			dialog.ShowInformation("Success", 
				fmt.Sprintf("Download location set to:\n%s", newPath), fta.window)
		}
	}, fta.window)
}

func (fta *FileTransferApp) Run() {
	fta.window.ShowAndRun()
}