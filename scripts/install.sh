#!/bin/bash
# Poros Installation Script
# Usage: curl -sSL https://raw.githubusercontent.com/KilimcininKorOglu/poros/master/scripts/install.sh | bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="KilimcininKorOglu/poros"
BINARY_NAME="poros"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "$ARCH" in
        x86_64)  ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        arm64)   ARCH="arm64" ;;
        *)       echo -e "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
    esac
    
    case "$OS" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)      echo -e "${RED}Unsupported OS: $OS${NC}"; exit 1 ;;
    esac
    
    echo "$OS/$ARCH"
}

# Get latest release version
get_latest_version() {
    curl -sS "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install() {
    echo -e "${GREEN}Poros Network Path Tracer - Installer${NC}"
    echo ""
    
    PLATFORM=$(detect_platform)
    echo -e "Detected platform: ${YELLOW}$PLATFORM${NC}"
    
    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        VERSION="latest"
        echo -e "Using: ${YELLOW}$VERSION${NC}"
    else
        echo -e "Latest version: ${YELLOW}$VERSION${NC}"
    fi
    
    OS=${PLATFORM%/*}
    ARCH=${PLATFORM#*/}
    
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY_NAME}-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
    fi
    
    echo -e "Downloading from: ${YELLOW}$DOWNLOAD_URL${NC}"
    
    TMP_DIR=$(mktemp -d)
    TMP_FILE="$TMP_DIR/$BINARY_NAME"
    
    if ! curl -sSL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
        echo -e "${RED}Failed to download binary${NC}"
        rm -rf "$TMP_DIR"
        exit 1
    fi
    
    chmod +x "$TMP_FILE"
    
    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    else
        echo -e "${YELLOW}Installing to $INSTALL_DIR requires sudo${NC}"
        sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    rm -rf "$TMP_DIR"
    
    echo ""
    echo -e "${GREEN}✓ Poros installed successfully!${NC}"
    echo ""
    echo "Run 'poros --help' to get started."
    echo ""
    
    # Verify installation
    if command -v poros &> /dev/null; then
        echo -e "Installed version: ${YELLOW}$(poros version 2>/dev/null | head -n1)${NC}"
    fi
}

# Build from source
build_from_source() {
    echo -e "${GREEN}Building Poros from source...${NC}"
    
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Go is not installed. Please install Go first.${NC}"
        exit 1
    fi
    
    go install github.com/$REPO/cmd/poros@latest
    
    echo -e "${GREEN}✓ Poros built and installed!${NC}"
}

# Main
case "${1:-install}" in
    install)
        install
        ;;
    source)
        build_from_source
        ;;
    *)
        echo "Usage: $0 [install|source]"
        exit 1
        ;;
esac
