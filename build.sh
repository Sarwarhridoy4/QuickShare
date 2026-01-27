#!/bin/bash
# =====================================================================
# QuickShare Build & Package Script (Debian + AppImage)
# High-speed cross-platform file transfer application
# =====================================================================

set -e

APP_NAME="quickshare"
APP_ID="com.sarwar.quickshare"
ICON_NAME="quickshare"
AUTHOR="Sarwar Hossain"
EMAIL="sarwarhridoy4@gmail.com"

# ---------------------------------------------------------------------
# Version detection
# ---------------------------------------------------------------------
if git rev-parse --git-dir > /dev/null 2>&1; then
    DEFAULT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "1.0.0")
    GIT_HASH=$(git rev-parse --short HEAD)
else
    DEFAULT_VERSION="1.0.0"
    GIT_HASH="unknown"
fi

if [ -n "$1" ]; then
    VERSION="$1"
else
    read -p "Enter version [default: ${DEFAULT_VERSION}]: " VERSION
    VERSION=${VERSION:-"$DEFAULT_VERSION"}
fi

ARCH_RAW=$(uname -m)
case "$ARCH_RAW" in
  x86_64) ARCH="amd64"; APPIMAGE_ARCH="x86_64" ;;
  aarch64) ARCH="arm64"; APPIMAGE_ARCH="arm64" ;;
  armv7l) ARCH="armhf"; APPIMAGE_ARCH="armhf" ;;
  i386) ARCH="i386"; APPIMAGE_ARCH="i386" ;;
  *) ARCH="$ARCH_RAW"; APPIMAGE_ARCH="$ARCH_RAW" ;;
esac

echo "📦 Building QuickShare ${VERSION} (${GIT_HASH}) for ${ARCH}"

# ---------------------------------------------------------------------
# Dependency helper
# ---------------------------------------------------------------------
install_if_missing() {
    local cmd="$1"
    local pkg="$2"
    if ! command -v "$cmd" &> /dev/null; then
        echo "⬇️ Installing $pkg..."
        sudo apt-get update
        sudo apt-get install -y "$pkg"
    fi
}

install_if_missing go golang-go
install_if_missing git git
install_if_missing curl curl
install_if_missing unzip unzip
install_if_missing dpkg-deb dpkg-dev
install_if_missing magick imagemagick

# Install libfuse2 directly (required for AppImage)
if ! dpkg -l | grep -q libfuse2; then
    echo "⬇️ Installing libfuse2..."
    sudo apt-get update
    sudo apt-get install -y libfuse2
fi

# ---------------------------------------------------------------------
# AppImage tool
# ---------------------------------------------------------------------
if ! command -v appimagetool &> /dev/null; then
    echo "⬇️ Downloading appimagetool..."
    wget -q https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-${APPIMAGE_ARCH}.AppImage -O appimagetool
    chmod +x appimagetool
    sudo mv appimagetool /usr/local/bin/appimagetool
fi

# ImageMagick command
if command -v magick &> /dev/null; then
    IM_CMD="magick convert"
else
    IM_CMD="convert"
fi

# ---------------------------------------------------------------------
# Go build
# ---------------------------------------------------------------------
rm -rf ${APP_NAME}-deb QuickShare.AppDir *.deb *.AppImage

go mod tidy
echo "⚙️ Building Go binary..."
go build \
  -ldflags="-X 'main.AppID=${APP_ID}' -X 'main.GitCommit=${GIT_HASH}'" \
  -o ${APP_NAME}

# ---------------------------------------------------------------------
# Debian package
# ---------------------------------------------------------------------
echo "📦 Creating Debian package..."

ICON_SIZES=(16 22 24 32 48 64 128 256 512)

mkdir -p ${APP_NAME}-deb/DEBIAN
mkdir -p ${APP_NAME}-deb/usr/bin
mkdir -p ${APP_NAME}-deb/usr/share/applications
mkdir -p ${APP_NAME}-deb/usr/share/icons/hicolor
mkdir -p ${APP_NAME}-deb/usr/share/pixmaps

cp ${APP_NAME} ${APP_NAME}-deb/usr/bin/

for SIZE in "${ICON_SIZES[@]}"; do
    DIR=${APP_NAME}-deb/usr/share/icons/hicolor/${SIZE}x${SIZE}/apps
    mkdir -p ${DIR}
    $IM_CMD icon.png -resize ${SIZE}x${SIZE} ${DIR}/${ICON_NAME}.png
done

cp icon.png ${APP_NAME}-deb/usr/share/pixmaps/${ICON_NAME}.png

cat <<EOF > ${APP_NAME}-deb/usr/share/applications/${APP_NAME}.desktop
[Desktop Entry]
Type=Application
Name=QuickShare
GenericName=File Transfer
Comment=High-speed cross-platform file transfer
Exec=${APP_NAME}
Icon=${ICON_NAME}
Terminal=false
Categories=Utility;Network;
StartupNotify=true
StartupWMClass=QuickShare
X-GNOME-UsesNotifications=true
EOF

cat <<EOF > ${APP_NAME}-deb/DEBIAN/control
Package: ${APP_NAME}
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${ARCH}
Maintainer: ${AUTHOR} <${EMAIL}>
Depends: libgl1, libx11-6, libxcursor1, libxrandr2, libxinerama1, libxi6, libgtk-3-0
Description: QuickShare - High-speed cross-platform file transfer application
EOF

cat <<'EOF' > ${APP_NAME}-deb/DEBIAN/postinst
#!/bin/sh
set -e
command -v update-desktop-database >/dev/null && update-desktop-database -q
command -v gtk-update-icon-cache >/dev/null && gtk-update-icon-cache -q /usr/share/icons/hicolor
EOF
chmod 755 ${APP_NAME}-deb/DEBIAN/postinst

dpkg-deb --build ${APP_NAME}-deb
mv ${APP_NAME}-deb.deb ${APP_NAME}_${VERSION}_${ARCH}.deb

# ---------------------------------------------------------------------
# AppImage
# ---------------------------------------------------------------------
echo "📦 Creating AppImage..."

mkdir -p QuickShare.AppDir/usr/bin
mkdir -p QuickShare.AppDir/usr/share/applications
mkdir -p QuickShare.AppDir/usr/share/icons/hicolor

cp ${APP_NAME} QuickShare.AppDir/usr/bin/
chmod +x QuickShare.AppDir/usr/bin/${APP_NAME}
ln -sf usr/bin/${APP_NAME} QuickShare.AppDir/AppRun

for SIZE in "${ICON_SIZES[@]}"; do
    DIR=QuickShare.AppDir/usr/share/icons/hicolor/${SIZE}x${SIZE}/apps
    mkdir -p ${DIR}
    $IM_CMD icon.png -resize ${SIZE}x${SIZE} ${DIR}/${ICON_NAME}.png
done

cp icon.png QuickShare.AppDir/${ICON_NAME}.png

cat <<EOF > QuickShare.AppDir/${APP_NAME}.desktop
[Desktop Entry]
Type=Application
Name=QuickShare
GenericName=File Transfer
Comment=High-speed cross-platform file transfer
Exec=${APP_NAME}
Icon=${ICON_NAME}
Terminal=false
Categories=Utility;Network;
StartupNotify=true
StartupWMClass=QuickShare
X-GNOME-UsesNotifications=true
EOF

mkdir -p QuickShare.AppDir/usr/lib
ldd ${APP_NAME} | awk '/=> \// {print $3}' | xargs -I '{}' cp -v '{}' QuickShare.AppDir/usr/lib/ || true

appimagetool QuickShare.AppDir
mv QuickShare-${APPIMAGE_ARCH}.AppImage ${APP_NAME}_${VERSION}_${APPIMAGE_ARCH}.AppImage

# ---------------------------------------------------------------------
echo "🎉 Build complete"
echo " • Debian  : ${APP_NAME}_${VERSION}_${ARCH}.deb"
echo " • AppImage: ${APP_NAME}_${VERSION}_${APPIMAGE_ARCH}.AppImage"