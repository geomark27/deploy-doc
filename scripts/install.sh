#!/usr/bin/env bash
# ================================================================
#  gtt - Instalador para Linux / Mac
#  Uso: curl -fsSL https://raw.githubusercontent.com/geomark27/deploy-doc/main/scripts/install.sh | bash
# ================================================================

set -e

REPO="geomark27/deploy-doc"
INSTALL_DIR="$HOME/.local/bin"

# Detectar OS y arquitectura
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  ASSET="gtt-linux-amd64"  ;;
  darwin) ASSET="gtt-darwin-amd64" ;;
  *)
    echo "Sistema operativo no soportado: $OS"
    exit 1
    ;;
esac

echo ""
echo "  gtt - Instalador"
echo "  ────────────────────────────────────"
echo ""

# 1. Obtener última versión
echo "[1/4] Buscando ultima version..."
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' \
  | cut -d'"' -f4)
echo "      version: $VERSION"

# 2. Crear directorio si no existe
mkdir -p "$INSTALL_DIR"

# 3. Descargar binario
echo "[2/4] Descargando $ASSET..."
URL="https://github.com/$REPO/releases/download/$VERSION/$ASSET"
curl -fsSL "$URL" -o "$INSTALL_DIR/gtt"
chmod +x "$INSTALL_DIR/gtt"
echo "      descargado en $INSTALL_DIR/gtt"

# 4. Limpiar binario legacy de deploy-doc (si existe)
echo "[3/4] Limpiando archivos anteriores..."
if [ -f "$INSTALL_DIR/deploy-doc" ]; then
  rm -f "$INSTALL_DIR/deploy-doc"
  echo "      deploy-doc eliminado"
else
  echo "      nada que limpiar"
fi

# 5. Verificar PATH
echo "[4/4] Verificando PATH..."
if echo "$PATH" | grep -q "$INSTALL_DIR"; then
  echo "      ya estaba en PATH"
else
  echo "      agregando $INSTALL_DIR al PATH..."
  SHELL_RC=""
  if [ -f "$HOME/.zshrc" ]; then
    SHELL_RC="$HOME/.zshrc"
  elif [ -f "$HOME/.bashrc" ]; then
    SHELL_RC="$HOME/.bashrc"
  fi

  if [ -n "$SHELL_RC" ]; then
    echo "" >> "$SHELL_RC"
    echo "# gtt" >> "$SHELL_RC"
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_RC"
    echo "      agregado a $SHELL_RC"
  else
    echo "      agrega manualmente a tu shell: export PATH=\"\$PATH:$INSTALL_DIR\""
  fi
fi

# Resultado final
echo ""
echo "  gtt $VERSION instalado correctamente"
echo ""
echo "  Proximos pasos:"
echo "    1. Ejecuta: source ~/.zshrc  (o abre una nueva terminal)"
echo "    2. Ejecuta: gtt init"
echo "    3. Usa:     gtt g -i APP-1999 -b <hash>"
echo ""
