#!/bin/sh
set -e

REPO="ghchinoy/pirate-weather"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" = "x86_64" ]; then
  ARCH="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
  ARCH="arm64"
fi

echo "Detecting latest release for ${OS}_${ARCH}..."
LATEST_URL=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep "browser_download_url.*${OS}_${ARCH}\.tar\.gz" | cut -d '"' -f 4)

if [ -z "$LATEST_URL" ]; then
  echo "Could not find a release for ${OS}_${ARCH}."
  echo "You may need to build from source or check the releases page: https://github.com/${REPO}/releases"
  exit 1
fi

echo "Downloading $LATEST_URL..."
curl -sL "$LATEST_URL" | tar xz pirate-weather
echo ""
echo "Arrr! Downloaded pirate-weather successfully!"
echo "Run it with: ./pirate-weather"
