#!/usr/bin/env bash

set -euo pipefail

APP_NAME="QuickShare"
APP_ID="com.filetransfer.app"
MODULE_MAIN="."
BUILD_DIR="build"
APP_VERSION="1.0.0"
APP_BUILD="1"
APP_SLUG="quickshare"

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
  debian    Build Debian package (.deb) for linux
  appimage  Build AppImage package (.AppImage) for linux
  linuxpkg  Build both Debian and AppImage packages for linux
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

load_app_metadata() {
	if [[ -f "FyneApp.toml" ]]; then
		local fyne_name fyne_version fyne_build

		fyne_name="$(sed -n 's/^Name = "\(.*\)"/\1/p' FyneApp.toml | head -n 1)"
		fyne_version="$(sed -n 's/^Version = "\(.*\)"/\1/p' FyneApp.toml | head -n 1)"
		fyne_build="$(sed -n 's/^Build = \([0-9][0-9]*\)$/\1/p' FyneApp.toml | head -n 1)"

		if [[ -n "${fyne_name}" ]]; then
			APP_NAME="${fyne_name}"
		fi
		if [[ -n "${fyne_version}" ]]; then
			APP_VERSION="${fyne_version}"
		fi
		if [[ -n "${fyne_build}" ]]; then
			APP_BUILD="${fyne_build}"
		fi
	fi

	APP_SLUG="$(echo "${APP_NAME}" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/--*/-/g; s/^-//; s/-$//')"
	if [[ -z "${APP_SLUG}" ]]; then
		APP_SLUG="quickshare"
	fi
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

require_cmd() {
	local command_name="$1"
	local install_hint="$2"
	if ! command -v "${command_name}" >/dev/null 2>&1; then
		echo -e "${RED}Error: '${command_name}' is required.${NC}"
		echo "${install_hint}"
		exit 1
	fi
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

build_debian_package() {
	local platform="$1"
	if [[ "${platform}" != "linux" ]]; then
		echo -e "${RED}Debian packaging is supported on linux only${NC}"
		exit 1
	fi

	require_cmd "dpkg-deb" "Install it with: sudo apt-get install -y dpkg-dev"

	local deb_arch
	deb_arch="$(dpkg --print-architecture 2>/dev/null || true)"
	if [[ -z "${deb_arch}" ]]; then
		case "$(uname -m)" in
			x86_64) deb_arch="amd64" ;;
			aarch64|arm64) deb_arch="arm64" ;;
			armv7l) deb_arch="armhf" ;;
			i386|i686) deb_arch="i386" ;;
			*) deb_arch="amd64" ;;
		esac
	fi

	local deb_version
	deb_version="${APP_VERSION}-${APP_BUILD}"

	local stage_root pkg_root control_dir bin_dir lib_dir app_dir icon_dir desktop_file wrapper_file launcher_file binary_file
	stage_root="${BUILD_DIR}/debian"
	pkg_root="${stage_root}/${APP_SLUG}_${deb_version}_${deb_arch}"
	control_dir="${pkg_root}/DEBIAN"
	bin_dir="${pkg_root}/usr/bin"
	lib_dir="${pkg_root}/usr/lib/${APP_SLUG}"
	app_dir="${pkg_root}/usr/share/applications"
	icon_dir="${pkg_root}/usr/share/icons/hicolor/256x256/apps"
	desktop_file="${app_dir}/${APP_SLUG}.desktop"
	wrapper_file="${bin_dir}/${APP_SLUG}"
	launcher_file="${lib_dir}/${APP_SLUG}-bin"
	binary_file="${BUILD_DIR}/${APP_SLUG}-linux"

	echo -e "\n${YELLOW}Building Debian package...${NC}"
	rm -rf "${stage_root}"
	mkdir -p "${control_dir}" "${bin_dir}" "${lib_dir}" "${app_dir}" "${icon_dir}"

	go build -o "${binary_file}" "${MODULE_MAIN}"
	cp "${binary_file}" "${launcher_file}"
	chmod 0755 "${launcher_file}"

	cat > "${wrapper_file}" <<EOF
#!/bin/sh
set -eu
WORKDIR="\${XDG_DATA_HOME:-\$HOME/.local/share}/${APP_SLUG}"
mkdir -p "\${WORKDIR}"
cd "\${WORKDIR}"
exec "/usr/lib/${APP_SLUG}/${APP_SLUG}-bin" "\$@"
EOF
	chmod 0755 "${wrapper_file}"

	cat > "${desktop_file}" <<EOF
[Desktop Entry]
Type=Application
Name=${APP_NAME}
Exec=${APP_SLUG}
Icon=${APP_SLUG}
Terminal=false
Categories=Utility;Network;
StartupNotify=true
EOF

	cp Icon.png "${icon_dir}/${APP_SLUG}.png"
	chmod 0644 "${icon_dir}/${APP_SLUG}.png"

	cat > "${control_dir}/control" <<EOF
Package: ${APP_SLUG}
Version: ${deb_version}
Section: utils
Priority: optional
Architecture: ${deb_arch}
Maintainer: QuickShare Team <noreply@localhost>
Depends: libc6
Description: ${APP_NAME} LAN file transfer application
 ${APP_NAME} is a Go + Fyne desktop app for local network file transfers.
EOF
	chmod 0644 "${control_dir}/control"

	local output_deb
	output_deb="${BUILD_DIR}/${APP_SLUG}_${deb_version}_${deb_arch}.deb"
	dpkg-deb --build --root-owner-group "${pkg_root}" "${output_deb}" >/dev/null

	echo -e "${GREEN}Built: ${output_deb}${NC}"
}

build_appimage_package() {
	local platform="$1"
	if [[ "${platform}" != "linux" ]]; then
		echo -e "${RED}AppImage packaging is supported on linux only${NC}"
		exit 1
	fi

	require_cmd "appimagetool" "Install appimagetool from AppImageKit: https://github.com/AppImage/AppImageKit/releases"

	local app_arch
	case "$(uname -m)" in
		x86_64) app_arch="x86_64" ;;
		aarch64|arm64) app_arch="aarch64" ;;
		armv7l) app_arch="armhf" ;;
		i386|i686) app_arch="i386" ;;
		*) app_arch="$(uname -m)" ;;
	esac

	local stage_root appdir usr_bin usr_share_app usr_share_icons launcher_bin wrapper_bin desktop_file apprun_file output_file
	stage_root="${BUILD_DIR}/appimage"
	appdir="${stage_root}/AppDir"
	usr_bin="${appdir}/usr/bin"
	usr_share_app="${appdir}/usr/share/applications"
	usr_share_icons="${appdir}/usr/share/icons/hicolor/256x256/apps"
	launcher_bin="${usr_bin}/${APP_SLUG}-bin"
	wrapper_bin="${usr_bin}/${APP_SLUG}"
	desktop_file="${appdir}/${APP_SLUG}.desktop"
	apprun_file="${appdir}/AppRun"
	output_file="${BUILD_DIR}/${APP_SLUG}-${APP_VERSION}-${APP_BUILD}-${app_arch}.AppImage"

	echo -e "\n${YELLOW}Building AppImage package...${NC}"
	rm -rf "${stage_root}"
	mkdir -p "${usr_bin}" "${usr_share_app}" "${usr_share_icons}"

	go build -o "${launcher_bin}" "${MODULE_MAIN}"
	chmod 0755 "${launcher_bin}"

	cat > "${wrapper_bin}" <<EOF
#!/bin/sh
set -eu
WORKDIR="\${XDG_DATA_HOME:-\$HOME/.local/share}/${APP_SLUG}"
mkdir -p "\${WORKDIR}"
cd "\${WORKDIR}"
exec "\$(dirname "\$0")/${APP_SLUG}-bin" "\$@"
EOF
	chmod 0755 "${wrapper_bin}"

	cat > "${desktop_file}" <<EOF
[Desktop Entry]
Type=Application
Name=${APP_NAME}
Exec=${APP_SLUG}
Icon=${APP_SLUG}
Terminal=false
Categories=Utility;Network;
StartupNotify=true
EOF
	cp "${desktop_file}" "${usr_share_app}/${APP_SLUG}.desktop"

	cp Icon.png "${usr_share_icons}/${APP_SLUG}.png"
	cp Icon.png "${appdir}/${APP_SLUG}.png"

	cat > "${apprun_file}" <<EOF
#!/bin/sh
set -eu
HERE="\$(dirname "\$(readlink -f "\$0")")"
export PATH="\${HERE}/usr/bin:\${PATH}"
exec "\${HERE}/usr/bin/${APP_SLUG}" "\$@"
EOF
	chmod 0755 "${apprun_file}"

	ARCH="${app_arch}" APPIMAGE_EXTRACT_AND_RUN=1 appimagetool "${appdir}" "${output_file}" >/dev/null
	echo -e "${GREEN}Built: ${output_file}${NC}"
}

clean_artifacts() {
	echo -e "\n${YELLOW}Cleaning build artifacts...${NC}"
	rm -rf "${BUILD_DIR}/"
	rm -f ./*.apk ./*.app ./*.exe ./*.deb ./*.AppImage
	echo -e "${GREEN}Clean completed${NC}"
}

main() {
	print_header
	require_go
	load_app_metadata

	local target="${1:-desktop}"
	local platform
	platform="$(normalize_platform "${2:-}")"

	echo "Target: ${target}"
	echo "Platform: ${platform}"
	echo "Version: ${APP_VERSION} (build ${APP_BUILD})"

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
		debian)
			download_deps
			build_debian_package "${platform}"
			;;
		appimage)
			download_deps
			build_appimage_package "${platform}"
			;;
		linuxpkg)
			download_deps
			build_debian_package linux
			build_appimage_package linux
			echo -e "\n${GREEN}Linux packages completed${NC}"
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
