#!/bin/sh
set -e

# Beaverish Installer script
# Usage:
#   curl -sSL https://raw.githubusercontent.com/nathfavour/beaverish/main/install.sh | sh

INSTALL_DIR="${HOME}/.local/bin"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/beaverish"
BINARY_NAME="beaverish"

echo "==> Installing Beaverish..."

mkdir -p "${INSTALL_DIR}"
mkdir -p "${CONFIG_DIR}"

if command -v go >/dev/null 2>&1; then
    echo "==> Go compiler detected. Building latest version from source..."
    TMP_SRC=$(mktemp -d)
    git clone --depth 1 https://github.com/nathfavour/beaverish.git "${TMP_SRC}" 2>/dev/null || true
    if [ -d "${TMP_SRC}/cmd/agent" ]; then
        cd "${TMP_SRC}"
        CGO_ENABLED=0 go build -ldflags="-s -w" -o "${INSTALL_DIR}/${BINARY_NAME}" ./cmd/agent
        rm -rf "${TMP_SRC}"
    else
        echo "==> Installing direct via go install..."
        GOBIN="${INSTALL_DIR}" go install github.com/nathfavour/beaverish/cmd/agent@latest
        if [ -f "${INSTALL_DIR}/agent" ]; then
            mv "${INSTALL_DIR}/agent" "${INSTALL_DIR}/${BINARY_NAME}"
        fi
    fi
else
    echo "==> Downloading precompiled binary..."
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    DOWNLOAD_URL="https://github.com/nathfavour/beaverish/releases/latest/download/beaverish-${OS}-${ARCH}"
    curl -sSL -f "${DOWNLOAD_URL}" -o "${INSTALL_DIR}/${BINARY_NAME}" || {
        echo "Precompiled release binary not found, checking local build..."
        if [ -f "./bin/beaverish" ]; then
            cp ./bin/beaverish "${INSTALL_DIR}/${BINARY_NAME}"
        else
            echo "Failed to download binary. Please install Go and run 'make install'."
            exit 1
        fi
    }
fi

chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

# Scaffold default configuration if not present
CONFIG_FILE="${CONFIG_DIR}/config.json"
if [ ! -f "${CONFIG_FILE}" ]; then
    cat << 'CONFIG_EOF' > "${CONFIG_FILE}"
{
  "network": {
    "driver": "somnia",
    "rpc_url": "https://dream-rpc.somnia.network",
    "ws_url": "wss://dream-rpc.somnia.network/ws",
    "chain_id": 50312,
    "contracts": {
      "clob_router": "0x0000000000000000000000000000000000000000",
      "collateral_token": "0x0000000000000000000000000000000000000000"
    }
  },
  "wallet": {
    "private_key": "",
    "auto_approve": true
  },
  "risk": {
    "max_bet_size_units": 5.0,
    "max_drawdown_limit_units": 50.0,
    "min_edge_threshold": 0.04,
    "expiry_cutoff_seconds": 30
  },
  "runtime": {
    "poll_interval_ms": 500,
    "dry_run": false,
    "log_format": "text"
  }
}
CONFIG_EOF
    echo "==> Created default configuration at ${CONFIG_FILE}"
fi

echo "==> Beaverish successfully installed to ${INSTALL_DIR}/${BINARY_NAME}"
echo ""
echo "Ensure ${INSTALL_DIR} is in your PATH:"
echo "    export PATH=\"\$PATH:${INSTALL_DIR}\""
echo ""
echo "Test installation:"
echo "    beaverish --version"
echo "    beaverish --mcp"
