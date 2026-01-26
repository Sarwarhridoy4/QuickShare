package main

import (
	"github.com/Sarwarhridoy4/QuickShare/internal/ui"
	"github.com/Sarwarhridoy4/QuickShare/internal/utils"
)

func main() {
	// Initialize logger
	utils.InitLogger()
	utils.Log("Starting File Transfer Application")

	// Create and run the application
	app := ui.NewFileTransferApp()
	app.Run()
}