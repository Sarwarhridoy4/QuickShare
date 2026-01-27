#!/bin/bash

# Build script for File Transfer Application
# Supports Linux, Windows, macOS, Android, and iOS

set -e

echo "==================================="
echo "Quick Share - Build Script"
echo "==================================="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}Go version: $(go version)${NC}"

# Create necessary directories
mkdir -p build
mkdir -p logs
mkdir -p downloads
mkdir -p assets/icons

# Parse command line arguments
TARGET=${1:-"desktop"}
PLATFORM=${2:-$(uname | tr '[:upper:]' '[:lower:]')}

echo "Target: $TARGET"
echo "Platform: $PLATFORM"

# Install dependencies
echo -e "\n${YELLOW}Installing dependencies...${NC}"
go mod download
go mod tidy

# Function to build for desktop
build_desktop() {
    echo -e "\n${YELLOW}Building for desktop ($PLATFORM)...${NC}"
    
    case $PLATFORM in
        linux)
            go build -o build/filetransfer-linux cmd/main.go
            echo -e "${GREEN}Built: build/filetransfer-linux${NC}"
            ;;
        windows)
            GOOS=windows GOARCH=amd64 go build -o build/filetransfer-windows.exe cmd/main.go
            echo -e "${GREEN}Built: build/filetransfer-windows.exe${NC}"
            ;;
        darwin)
            go build -o build/filetransfer-macos cmd/main.go
            echo -e "${GREEN}Built: build/filetransfer-macos${NC}"
            ;;
        *)
            go build -o build/filetransfer cmd/main.go
            echo -e "${GREEN}Built: build/filetransfer${NC}"
            ;;
    esac
}

# Function to build for mobile
build_mobile() {
    echo -e "\n${YELLOW}Building for mobile ($PLATFORM)...${NC}"
    
    # Check if fyne CLI is installed
    if ! command -v fyne &> /dev/null; then
        echo -e "${YELLOW}Installing Fyne CLI...${NC}"
        go install fyne.io/fyne/v2/cmd/fyne@latest
    fi
    
    case $PLATFORM in
        android)
            echo -e "${YELLOW}Building Android APK...${NC}"
            fyne package -os android -appID com.filetransfer.app -name FileTransfer
            echo -e "${GREEN}Built: FileTransfer.apk${NC}"
            ;;
        ios)
            echo -e "${YELLOW}Building iOS app...${NC}"
            fyne package -os ios -appID com.filetransfer.app -name FileTransfer
            echo -e "${GREEN}Built: FileTransfer.app${NC}"
            ;;
        *)
            echo -e "${RED}Unsupported mobile platform: $PLATFORM${NC}"
            exit 1
            ;;
    esac
}

# Function to package with Fyne
build_package() {
    echo -e "\n${YELLOW}Building packaged application...${NC}"
    
    if ! command -v fyne &> /dev/null; then
        echo -e "${YELLOW}Installing Fyne CLI...${NC}"
        go install fyne.io/fyne/v2/cmd/fyne@latest
    fi
    
    case $PLATFORM in
        linux)
            fyne package -os linux -name FileTransfer
            echo -e "${GREEN}Packaged for Linux${NC}"
            ;;
        windows)
            fyne package -os windows -name FileTransfer
            echo -e "${GREEN}Packaged for Windows${NC}"
            ;;
        darwin)
            fyne package -os darwin -name FileTransfer
            echo -e "${GREEN}Packaged for macOS${NC}"
            ;;
    esac
}

# Main build logic
case $TARGET in
    desktop)
        build_desktop
        ;;
    mobile)
        build_mobile
        ;;
    package)
        build_package
        ;;
    all)
        echo -e "${YELLOW}Building for all platforms...${NC}"
        
        # Desktop builds
        PLATFORM=linux build_desktop
        PLATFORM=windows build_desktop
        PLATFORM=darwin build_desktop
        
        echo -e "\n${GREEN}All builds completed!${NC}"
        ;;
    clean)
        echo -e "${YELLOW}Cleaning build artifacts...${NC}"
        rm -rf build/
        rm -f *.apk *.app
        echo -e "${GREEN}Clean completed${NC}"
        ;;
    *)
        echo -e "${RED}Unknown target: $TARGET${NC}"
        echo "Usage: $0 [desktop|mobile|package|all|clean] [platform]"
        echo "Platforms: linux, windows, darwin, android, ios"
        exit 1
        ;;
esac

echo -e "\n${GREEN}Build completed successfully!${NC}"