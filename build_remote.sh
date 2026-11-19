#!/data/data/com.termux/files/usr/bin/bash

echo ""
echo "  PhantomAP :: Build Remote"
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

cd "$(dirname "$0")/remote"

CGO_ENABLED=0 go build -ldflags="-s -w" -o ../phantomAP_remote .
if [ $? -ne 0 ]; then
    echo "  ERR :: Build failed."
    exit 1
fi

cd ..
chmod +x phantomAP_remote

echo "  BUILD OK :: ./phantomAP_remote"
echo ""
echo "  Run with: ./phantomAP_remote"
echo ""
