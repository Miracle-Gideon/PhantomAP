#!/data/data/com.termux/files/usr/bin/bash

echo ""
echo "  PhantomAP :: Build All"
echo "  Built Miracle Gideon"
echo ""

# Install Go if not present
if ! command -v go &> /dev/null; then
    echo "  Go not found. Installing via pkg ..."
    pkg install golang -y
    if [ $? -ne 0 ]; then
        echo "  ERR :: Go installation failed."
        exit 1
    fi
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# ── Build Windows receiver (.exe) via cross-compilation ──────────────────────
echo "  Building phantomAP_receiver.exe (Windows x64) ..."
cd "$SCRIPT_DIR/receiver"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
    go build -ldflags="-s -w" -o "$SCRIPT_DIR/phantomAP_receiver.exe" .
if [ $? -ne 0 ]; then
    echo "  ERR :: receiver build failed."
    exit 1
fi
echo "  BUILD OK :: phantomAP_receiver.exe"
echo ""

# ── Build Termux remote binary (Linux ARM64) ─────────────────────────────────
echo "  Building phantomAP_remote (Termux ARM64) ..."
cd "$SCRIPT_DIR/remote"
CGO_ENABLED=0 \
    go build -ldflags="-s -w" -o "$SCRIPT_DIR/phantomAP_remote" .
if [ $? -ne 0 ]; then
    echo "  ERR :: remote build failed."
    exit 1
fi
chmod +x "$SCRIPT_DIR/phantomAP_remote"
echo "  BUILD OK :: phantomAP_remote"
echo ""

# ── Summary ──────────────────────────────────────────────────────────────────
echo "  -----------------------------------------------"
echo "  OUTPUT FILES:"
echo ""
echo "  phantomAP_receiver.exe  →  copy to Windows PC"
echo "  phantomAP_remote        →  run on this device"
echo ""
echo "  RUN REMOTE : ./phantomAP_remote"
echo "  -----------------------------------------------"
echo ""
