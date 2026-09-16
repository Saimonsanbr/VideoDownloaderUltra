#!/bin/sh
set -e
# VideoDownloaderUltra installer - macOS / Linux (Debian 13)
REPO="Saimonsanbr/VideoDownloaderUltra"
BIN="videodownloaderultra"
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Arquitetura não suportada: $ARCH"; exit 1 ;;
esac
case "$OS" in
  darwin) OS="darwin" ;;
  linux) OS="linux" ;;
  *) echo "SO não suportado: $OS"; exit 1 ;;
esac

# pega latest release tag via redirect
echo "Detectando último release..."
LATEST_URL="https://github.com/$REPO/releases/latest"
TAG=$(curl -fsSL -o /dev/null -w "%{url_effective}" "$LATEST_URL" | sed 's|.*/tag/||')
if [ -z "$TAG" ]; then TAG="latest"; fi
echo "Tag: $TAG OS=$OS ARCH=$ARCH"

FILE="${BIN}-${OS}-${ARCH}"
if [ "$OS" = "linux" ] && [ "$ARCH" = "amd64" ]; then
  FILE="${BIN}-linux-amd64"
fi
URL="https://github.com/$REPO/releases/download/$TAG/$FILE"
if [ "$TAG" = "latest" ]; then
  URL="https://github.com/$REPO/releases/latest/download/$FILE"
fi
TMP=$(mktemp /tmp/vdu_XXXX.tar.gz 2>/dev/null || mktemp -t vdu)
echo "Baixando $URL ..."
if ! curl -fL "$URL" -o "$TMP"; then
  echo "Falha curl, tentando wget..."
  wget -O "$TMP" "$URL"
fi
# se for gz, extrai, se for binário direto copia
if file "$TMP" | grep -q "gzip"; then
  tar -xzf "$TMP" -C /tmp
  BIN_SRC=$(find /tmp -name "${BIN}*" -type f | head -n1)
else
  BIN_SRC="$TMP"
fi
if [ -z "$BIN_SRC" ] || [ ! -f "$BIN_SRC" ]; then
  # tenta como binário direto
  BIN_SRC="$TMP"
fi
echo "Instalando em $INSTALL_DIR/$BIN ..."
mkdir -p "$INSTALL_DIR"
cp "$BIN_SRC" "$INSTALL_DIR/$BIN"
chmod +x "$INSTALL_DIR/$BIN"
rm -f "$TMP"

# .desktop para Linux (menu)
if [ "$OS" = "linux" ]; then
  DESKTOP_DIR="$HOME/.local/share/applications"
  mkdir -p "$DESKTOP_DIR"
  cat > "$DESKTOP_DIR/videodownloaderultra.desktop" <<EOF
[Desktop Entry]
Name=VideoDownloaderUltra
Comment=Downloader de vídeo (teste API terceiros)
Exec=$INSTALL_DIR/$BIN --gui
Icon=video-display
Terminal=false
Type=Application
Categories=AudioVideo;Network;
EOF
  echo "Desktop entry criado em $DESKTOP_DIR/videodownloaderultra.desktop"
fi

echo ""
echo "✓ Instalado: $INSTALL_DIR/$BIN"
echo "  Versão: $($INSTALL_DIR/$BIN --help 2>&1 | head -n1)"
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
  echo "  Adicione ao PATH: export PATH=\"\$PATH:$INSTALL_DIR\""
  echo "  (adicione ao ~/.zshrc ou ~/.bashrc)"
fi
echo ""
echo "Uso CLI: $BIN https://youtu.be/..."
echo "Uso GUI: $BIN --gui  ou pesquise 'VideoDownloaderUltra' no menu"
echo ""
echo "macOS Gatekeeper: xattr -c $INSTALL_DIR/$BIN"
