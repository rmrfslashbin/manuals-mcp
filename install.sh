#!/bin/sh
# Installation script for manuals-mcp
# Usage: wget -qO- https://raw.githubusercontent.com/rmrfslashbin/manuals-mcp/main/install.sh | sh
# Or: curl -fsSL https://raw.githubusercontent.com/rmrfslashbin/manuals-mcp/main/install.sh | sh

set -e

REPO="rmrfslashbin/manuals-mcp"
BINARY_NAME="manuals-mcp"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_info() {
    printf "${GREEN}==>${NC} %s\n" "$1"
}

print_error() {
    printf "${RED}Error:${NC} %s\n" "$1" >&2
}

print_warning() {
    printf "${YELLOW}Warning:${NC} %s\n" "$1"
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin*)
            echo "darwin"
            ;;
        Linux*)
            echo "linux"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            echo "windows"
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
}

# Get latest release tag from GitHub
get_latest_release() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
            grep '"tag_name":' |
            sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" |
            grep '"tag_name":' |
            sed -E 's/.*"([^"]+)".*/\1/'
    else
        print_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi
}

# Download file
download_file() {
    url="$1"
    output="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$output" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$output" "$url"
    else
        print_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi
}

main() {
    print_info "Installing manuals-mcp..."

    # Detect system
    OS=$(detect_os)
    ARCH=$(detect_arch)
    print_info "Detected platform: ${OS}-${ARCH}"

    # Get latest release
    print_info "Fetching latest release..."
    VERSION=$(get_latest_release)

    if [ -z "$VERSION" ]; then
        print_error "Failed to fetch latest release version"
        exit 1
    fi

    print_info "Latest version: ${VERSION}"

    # Construct binary name and download URL
    if [ "$OS" = "windows" ]; then
        BINARY_FILE="${BINARY_NAME}-${OS}-${ARCH}.exe"
        OUTPUT_NAME="${BINARY_NAME}.exe"
    else
        BINARY_FILE="${BINARY_NAME}-${OS}-${ARCH}"
        OUTPUT_NAME="${BINARY_NAME}"
    fi

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_FILE}"

    print_info "Downloading from ${DOWNLOAD_URL}..."

    # Download binary
    if ! download_file "$DOWNLOAD_URL" "$OUTPUT_NAME"; then
        print_error "Failed to download binary"
        exit 1
    fi

    # Make executable (not needed on Windows)
    if [ "$OS" != "windows" ]; then
        chmod +x "$OUTPUT_NAME"
    fi

    # Verify installation
    if [ -f "$OUTPUT_NAME" ]; then
        print_info "Successfully installed ${OUTPUT_NAME}"
        print_info "Binary location: $(pwd)/${OUTPUT_NAME}"

        # Try to get version
        if [ "$OS" != "windows" ]; then
            if ./"$OUTPUT_NAME" version >/dev/null 2>&1; then
                VERSION_OUTPUT=$(./"$OUTPUT_NAME" version)
                print_info "Installed version: ${VERSION_OUTPUT}"
            fi
        fi

        echo ""
        print_info "Installation complete!"
        echo ""
        print_warning "The binary is installed in the current directory: $(pwd)"
        print_warning "To use it from anywhere, move it to a directory in your PATH:"
        echo ""

        if [ "$OS" = "darwin" ] || [ "$OS" = "linux" ]; then
            echo "  sudo mv ${OUTPUT_NAME} /usr/local/bin/"
            echo "  # or"
            echo "  mv ${OUTPUT_NAME} ~/.local/bin/  # make sure ~/.local/bin is in your PATH"
        fi

        echo ""
        print_info "Get started:"
        echo "  ${OUTPUT_NAME} --help"
        echo "  ${OUTPUT_NAME} index --docs-path /path/to/manuals-data"
        echo "  ${OUTPUT_NAME} serve --db-path ./manuals.db"

    else
        print_error "Installation failed - binary not found"
        exit 1
    fi
}

main
