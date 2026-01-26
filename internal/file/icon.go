package file

import (
	"github.com/Sarwarhridoy4/QuickShare/internal/utils"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// LoadIcon loads the appropriate icon based on the platform
func LoadIcon() fyne.Resource {
	utils.Log("Loading platform-specific icon")
	
	// Determine platform
	platform := runtime.GOOS
	
	switch platform {
	case "windows":
		return loadWindowsIcon()
	case "darwin":
		return loadMacIcon()
	case "linux":
		return loadLinuxIcon()
	case "android", "ios":
		return loadMobileIcon()
	default:
		return theme.FileIcon()
	}
}

func loadWindowsIcon() fyne.Resource {
	// Attempt to load from assets/icons/windows.ico
	// For now, return default icon
	// In production, you would use:
	// return fyne.NewStaticResource("icon", iconData)
	return theme.FileIcon()
}

func loadMacIcon() fyne.Resource {
	// Attempt to load from assets/icons/mac.icns
	return theme.FileIcon()
}

func loadLinuxIcon() fyne.Resource {
	// Attempt to load from assets/icons/linux.png
	return theme.FileIcon()
}

func loadMobileIcon() fyne.Resource {
	// Attempt to load from assets/icons/mobile.png
	return theme.FileIcon()
}

// GetTheme returns the appropriate theme based on platform
func GetTheme() fyne.Theme {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		// Mobile-optimized theme
		return theme.DefaultTheme()
	}
	return theme.DefaultTheme()
}