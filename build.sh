#!/usr/bin/env bash

set -euo pipefail

APP_NAME="QuickShare"
APP_ID="com.filetransfer.app"
MODULE_MAIN="."
BUILD_DIR="build"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

print_header() {
	echo "==================================="
	echo "${APP_NAME} - Build Script"
	echo "==================================="
}

usage() {
	cat <<'EOF'
Usage: ./build.sh [target] [platform]

Targets:
  desktop   Build desktop binary (default)
  mobile    Build mobile package with fyne (android|ios)
  package   Build desktop package with fyne (linux|windows|darwin)
  all       Build desktop binaries for linux/windows/darwin
  deps      Download dependencies only
  clean     Remove build artifacts
  help      Show this help

Platforms:
  linux | windows | darwin | android | ios
EOF
}

require_go() {
	if ! command -v go >/dev/null 2>&1; then
		echo -e "${RED}Error: Go is not installed${NC}"
		exit 1
	fi
	echo -e "${GREEN}Go version: $(go version)${NC}"
}

normalize_platform() {
	local raw="${1:-}"
	case "${raw}" in
		linux|windows|darwin|android|ios)
			echo "${raw}"
			;;
		*)
			case "$(uname -s)" in
				Linux) echo "linux" ;;
				Darwin) echo "darwin" ;;
				MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
				*)
					echo -e "${RED}Unsupported platform: ${raw:-$(uname -s)}${NC}"
					exit 1
					;;
			esac
			;;
	esac
}

download_deps() {
	echo -e "\n${YELLOW}Downloading dependencies...${NC}"
	go mod download
}

ensure_fyne() {
	if command -v fyne >/dev/null 2>&1; then
		FYNE_BIN="$(command -v fyne)"
		return
	fi

	echo -e "${YELLOW}Installing Fyne CLI...${NC}"
	go install fyne.io/fyne/v2/cmd/fyne@latest

	local gobin
	gobin="$(go env GOBIN)"
	if [[ -z "${gobin}" ]]; then
		gobin="$(go env GOPATH)/bin"
	fi

	FYNE_BIN="${gobin}/fyne"
	if [[ ! -x "${FYNE_BIN}" ]]; then
		echo -e "${RED}Error: fyne CLI was installed but not found at ${FYNE_BIN}${NC}"
		echo "Add ${gobin} to PATH or run '${FYNE_BIN}' directly."
		exit 1
	fi
}

build_desktop() {
	local platform="$1"
	mkdir -p "${BUILD_DIR}"
	echo -e "\n${YELLOW}Building desktop binary for ${platform}...${NC}"

	case "${platform}" in
		linux)
			go build -o "${BUILD_DIR}/quickshare-linux" "${MODULE_MAIN}"
			echo -e "${GREEN}Built: ${BUILD_DIR}/quickshare-linux${NC}"
			;;
		windows)
			GOOS=windows GOARCH=amd64 go build -o "${BUILD_DIR}/quickshare-windows-amd64.exe" "${MODULE_MAIN}"
			echo -e "${GREEN}Built: ${BUILD_DIR}/quickshare-windows-amd64.exe${NC}"
			;;
		darwin)
			GOOS=darwin GOARCH=amd64 go build -o "${BUILD_DIR}/quickshare-darwin-amd64" "${MODULE_MAIN}"
			GOOS=darwin GOARCH=arm64 go build -o "${BUILD_DIR}/quickshare-darwin-arm64" "${MODULE_MAIN}"
			echo -e "${GREEN}Built: ${BUILD_DIR}/quickshare-darwin-amd64${NC}"
			echo -e "${GREEN}Built: ${BUILD_DIR}/quickshare-darwin-arm64${NC}"
			;;
		*)
			echo -e "${RED}Unsupported desktop platform: ${platform}${NC}"
			exit 1
			;;
	esac
}

build_mobile() {
	local platform="$1"
	echo -e "\n${YELLOW}Building mobile package for ${platform}...${NC}"
	ensure_fyne

	case "${platform}" in
		android)
			"${FYNE_BIN}" package -os android -appID "${APP_ID}" -name "${APP_NAME}" -icon Icon.png
			echo -e "${GREEN}Built: ${APP_NAME}.apk${NC}"
			;;
		ios)
			"${FYNE_BIN}" package -os ios -appID "${APP_ID}" -name "${APP_NAME}" -icon Icon.png
			echo -e "${GREEN}Built: ${APP_NAME}.app${NC}"
			;;
		*)
			echo -e "${RED}Unsupported mobile platform: ${platform}${NC}"
			exit 1
			;;
	esac
}

build_package() {
	local platform="$1"
	echo -e "\n${YELLOW}Building packaged desktop app for ${platform}...${NC}"
	ensure_fyne

	case "${platform}" in
		linux|windows|darwin)
			"${FYNE_BIN}" package -os "${platform}" -name "${APP_NAME}" -appID "${APP_ID}" -icon Icon.png
			echo -e "${GREEN}Packaged for ${platform}${NC}"
			;;
		*)
			echo -e "${RED}Unsupported package platform: ${platform}${NC}"
			exit 1
			;;
	esac
}

clean_artifacts() {
	echo -e "\n${YELLOW}Cleaning build artifacts...${NC}"
	rm -rf "${BUILD_DIR}/"
	rm -f ./*.apk ./*.app ./*.exe
	echo -e "${GREEN}Clean completed${NC}"
}

main() {
	print_header
	require_go

	local target="${1:-desktop}"
	local platform
	platform="$(normalize_platform "${2:-}")"

	echo "Target: ${target}"
	echo "Platform: ${platform}"

	case "${target}" in
		desktop)
			download_deps
			build_desktop "${platform}"
			;;
		mobile)
			download_deps
			build_mobile "${platform}"
			;;
		package)
			download_deps
			build_package "${platform}"
			;;
		all)
			download_deps
			build_desktop linux
			build_desktop windows
			build_desktop darwin
			echo -e "\n${GREEN}All desktop builds completed${NC}"
			;;
		deps)
			download_deps
			;;
		clean)
			clean_artifacts
			;;
		help|-h|--help)
			usage
			;;
		*)
			echo -e "${RED}Unknown target: ${target}${NC}"
			usage
			exit 1
			;;
	esac

	echo -e "\n${GREEN}Build completed successfully${NC}"
}

main "$@"
