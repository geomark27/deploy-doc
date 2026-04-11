# ================================================================
#  gtt - Instalador para Windows
#  Uso: iwr -useb https://raw.githubusercontent.com/geomark27/deploy-doc/main/scripts/install.ps1 | iex
# ================================================================

$ErrorActionPreference = "Stop"

$repo    = "geomark27/deploy-doc"
$asset   = "gtt-windows-amd64.exe"
$installDir = "$env:LOCALAPPDATA\Programs\deploy-doc"
$dest    = "$installDir\gtt.exe"

Write-Host ""
Write-Host "  gtt - Instalador" -ForegroundColor Cyan
Write-Host "  ────────────────────────────────────" -ForegroundColor Cyan
Write-Host ""

# 1. Obtener última versión desde GitHub API
Write-Host "[1/4] Buscando ultima version..." -ForegroundColor Yellow
$release = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
$version = $release.tag_name
Write-Host "      version: $version" -ForegroundColor Green

# 2. Crear directorio si no existe
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

# 3. Descargar binario
Write-Host "[2/4] Descargando $asset..." -ForegroundColor Yellow
$url = "https://github.com/$repo/releases/download/$version/$asset"
Invoke-WebRequest -Uri $url -OutFile $dest
Write-Host "      descargado en $dest" -ForegroundColor Green

# 4. Limpiar archivos legacy de deploy-doc (si existen)
Write-Host "[3/4] Limpiando archivos anteriores..." -ForegroundColor Yellow
$removed = $false
if (Test-Path "$installDir\deploy-doc.exe") {
    Remove-Item "$installDir\deploy-doc.exe" -Force
    $removed = $true
}
if (Test-Path "$installDir\deploy-doc.exe.old") {
    Remove-Item "$installDir\deploy-doc.exe.old" -Force
    $removed = $true
}
if ($removed) {
    Write-Host "      deploy-doc eliminado" -ForegroundColor Green
} else {
    Write-Host "      nada que limpiar" -ForegroundColor Green
}

# 5. Agregar al PATH si no esta
Write-Host "[4/4] Verificando PATH..." -ForegroundColor Yellow
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$userPath;$installDir", "User")
    Write-Host "      agregado al PATH" -ForegroundColor Green
} else {
    Write-Host "      ya estaba en PATH" -ForegroundColor Green
}

# Resultado final
Write-Host ""
Write-Host "  gtt $version instalado correctamente" -ForegroundColor Green
Write-Host ""
Write-Host "  Proximos pasos:" -ForegroundColor Cyan
Write-Host "    1. Reinicia tu terminal"
Write-Host "    2. Ejecuta: gtt init"
Write-Host "    3. Usa:     gtt g -i APP-1999 -b <hash>"
Write-Host ""
