package file

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/Sarwarhridoy4/QuickShare/internal/utils"

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
	return loadIconFromPaths(
		"Icon.png",
		"icon.png",
		"assets/icons/windows.ico",
		"assets/icons/icon.png",
	)
}

func loadMacIcon() fyne.Resource {
	return loadIconFromPaths(
		"Icon.png",
		"icon.png",
		"assets/icons/mac.icns",
		"assets/icons/icon.png",
	)
}

func loadLinuxIcon() fyne.Resource {
	return loadIconFromPaths(
		"Icon.png",
		"icon.png",
		"assets/icons/linux.png",
		"assets/icons/icon.png",
	)
}

func loadMobileIcon() fyne.Resource {
	return loadIconFromPaths(
		"Icon.png",
		"icon.png",
		"assets/icons/mobile.png",
		"assets/icons/icon.png",
	)
}

func loadIconFromPaths(candidates ...string) fyne.Resource {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		icon := loadIconFile(candidate)
		if icon != nil {
			return icon
		}
	}

	utils.Log("No custom icon found, using default theme icon")
	return theme.FileIcon()
}

func loadIconFile(candidate string) fyne.Resource {
	searchPaths := []string{candidate}

	if wd, err := os.Getwd(); err == nil {
		searchPaths = append(searchPaths, filepath.Join(wd, candidate))
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		searchPaths = append(searchPaths,
			filepath.Join(exeDir, candidate),
			filepath.Join(exeDir, "..", candidate),
			filepath.Join(exeDir, "..", "..", candidate),
		)
	}

	for _, path := range searchPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		utils.Log("Loaded icon from " + path)
		return fyne.NewStaticResource(filepath.Base(candidate), data)
	}

	return nil
}

// GetTheme returns the appropriate theme based on platform
func GetTheme() fyne.Theme {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		// Mobile-optimized theme
		return theme.DefaultTheme()
	}
	return theme.DefaultTheme()
}
