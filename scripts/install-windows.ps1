#!/usr/bin/env pwsh
<#
.SYNOPSIS
  Vectra Guard - Windows Installer (One-Line)

.DESCRIPTION
  Downloads and installs the latest Vectra Guard binary for Windows.

  Features:
    - Detects architecture (amd64 / arm64)
    - User-space install by default ($HOME\AppData\Local\VectraGuard)
    - SHA256 checksum verification
    - Upgrade detection with version comparison
    - Adds vg / vectraguard aliases to PowerShell profile
    - Updates User PATH via registry (persists across sessions)
    - Non-interactive mode for CI/automation

  One-line install:
    irm https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-windows.ps1 | iex

  Non-interactive (CI):
    $env:VECTRAGUARD_YES = '1'; irm ... | iex

.NOTES
  Works on Windows PowerShell 5.1+ and PowerShell 7+.
  No admin/elevation required (user-space install).
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# ============================================================================
# Configuration
# ============================================================================

$Repo           = 'xadnavyaai/vectra-guard'
$BinaryName     = 'vectra-guard'
$DefaultInstall = Join-Path $env:LOCALAPPDATA 'VectraGuard'
$InstallDir     = if ($env:VECTRAGUARD_INSTALL_DIR) { $env:VECTRAGUARD_INSTALL_DIR } else { $DefaultInstall }
$NonInteractive = [bool]($env:VECTRAGUARD_YES -or $env:CI -or (-not [Environment]::UserInteractive))

# ============================================================================
# Helpers
# ============================================================================

function Write-Banner {
    Write-Host ''
    Write-Host '  Vectra Guard - Windows Installer' -ForegroundColor Cyan
    Write-Host '  =================================' -ForegroundColor Cyan
    Write-Host ''
}

function Get-Arch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        'AMD64'   { return 'amd64'  }
        'x86'     { return 'amd64'  }   # x86 host running 64-bit Windows
        'ARM64'   { return 'arm64'  }
        default   {
            # Fallback: check via .NET
            if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq 'Arm64') {
                return 'arm64'
            }
            return 'amd64'
        }
    }
}

function Confirm-Prompt {
    param([string]$Message, [bool]$Default = $true)
    if ($NonInteractive) { return $Default }

    $suffix = if ($Default) { '[Y/n]' } else { '[y/N]' }
    Write-Host "  $Message $suffix " -NoNewline
    $response = Read-Host
    $response = $response.Trim().ToLower()
    if ($response -eq '') { return $Default }
    return ($response -eq 'y' -or $response -eq 'yes')
}

# ============================================================================
# Main
# ============================================================================

Write-Banner

# Platform check
if ($IsLinux -or $IsMacOS) {
    Write-Host '  This installer is for Windows. On macOS/Linux use:' -ForegroundColor Red
    Write-Host '  curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash' -ForegroundColor Yellow
    exit 1
}

$Arch = Get-Arch
Write-Host "  Platform:    Windows $Arch"
Write-Host "  Install dir: $InstallDir"
Write-Host ''

# ============================================================================
# Upgrade detection
# ============================================================================

$BinaryPath = Join-Path $InstallDir "$BinaryName.exe"
$IsUpgrade  = $false

if (Test-Path -LiteralPath $BinaryPath) {
    $IsUpgrade = $true
    try {
        $oldVersion = & $BinaryPath version 2>$null | Select-Object -First 1
    } catch {
        $oldVersion = 'unknown'
    }
    Write-Host "  Existing install detected: $oldVersion" -ForegroundColor Yellow

    if (-not (Confirm-Prompt -Message 'Upgrade to latest version?' -Default $true)) {
        Write-Host '  Installation cancelled.' -ForegroundColor Yellow
        exit 0
    }
    Write-Host ''
}

# ============================================================================
# Download
# ============================================================================

$assetName   = "$BinaryName-windows-${Arch}.exe"
$downloadUrl = "https://github.com/$Repo/releases/latest/download/$assetName"
$checksumUrl = "https://github.com/$Repo/releases/latest/download/checksums.txt"

Write-Host "  Downloading $assetName ..." -ForegroundColor Cyan

# Ensure install directory exists
if (-not (Test-Path -LiteralPath $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$tempFile = Join-Path ([System.IO.Path]::GetTempPath()) "$BinaryName-$([guid]::NewGuid()).exe"

try {
    # Use TLS 1.2+ (required by GitHub)
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 -bor [Net.SecurityProtocolType]::Tls13

    $ProgressPreference = 'SilentlyContinue'  # Speed up Invoke-WebRequest
    Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -UseBasicParsing
} catch {
    Write-Host "  Download failed: $_" -ForegroundColor Red
    Write-Host "  URL: $downloadUrl" -ForegroundColor Red
    Write-Host ''
    Write-Host '  Troubleshooting:' -ForegroundColor Yellow
    Write-Host '    1. Check your internet connection'
    Write-Host '    2. Verify the release exists: https://github.com/$Repo/releases/latest'
    Write-Host '    3. Build from source: go install github.com/xadnavyaai/vectra-guard@latest'
    if (Test-Path -LiteralPath $tempFile) { Remove-Item -LiteralPath $tempFile -Force }
    exit 1
}

Write-Host '  Download complete.' -ForegroundColor Green

# ============================================================================
# Checksum verification
# ============================================================================

$checksumOK = $false
try {
    $checksumFile = Join-Path ([System.IO.Path]::GetTempPath()) "vg-checksums-$([guid]::NewGuid()).txt"
    Invoke-WebRequest -Uri $checksumUrl -OutFile $checksumFile -UseBasicParsing

    $checksumContent = Get-Content -LiteralPath $checksumFile -Raw
    # Parse: "<sha256>  <filename>" lines
    $expectedLine = ($checksumContent -split "`n") | Where-Object { $_ -match $assetName } | Select-Object -First 1
    if ($expectedLine) {
        $expectedSha = ($expectedLine -split '\s+')[0]
        $actualSha   = (Get-FileHash -LiteralPath $tempFile -Algorithm SHA256).Hash.ToLower()
        if ($expectedSha.ToLower() -eq $actualSha) {
            Write-Host '  Checksum verified (SHA256).' -ForegroundColor Green
            $checksumOK = $true
        } else {
            Write-Host "  Checksum mismatch!" -ForegroundColor Red
            Write-Host "    Expected: $expectedSha"
            Write-Host "    Actual:   $actualSha"
            Remove-Item -LiteralPath $tempFile -Force
            if (Test-Path -LiteralPath $checksumFile) { Remove-Item -LiteralPath $checksumFile -Force }
            exit 1
        }
    }
    if (Test-Path -LiteralPath $checksumFile) { Remove-Item -LiteralPath $checksumFile -Force }
} catch {
    Write-Host '  Checksum file not available. Skipping verification.' -ForegroundColor Yellow
}

# ============================================================================
# Install binary
# ============================================================================

Write-Host "  Installing to $BinaryPath ..." -ForegroundColor Cyan

# If binary is running, we can't overwrite it. Copy with a temp name, swap.
if (Test-Path -LiteralPath $BinaryPath) {
    $bakPath = "$BinaryPath.old"
    try { Remove-Item -LiteralPath $bakPath -Force -ErrorAction SilentlyContinue } catch {}
    Rename-Item -LiteralPath $BinaryPath -NewName (Split-Path -Leaf $bakPath) -Force
}

Move-Item -LiteralPath $tempFile -Destination $BinaryPath -Force

Write-Host '  Binary installed.' -ForegroundColor Green
Write-Host ''

# ============================================================================
# Update User PATH (registry) — persists across sessions
# ============================================================================

$regPath = 'HKCU:\Environment'
try {
    $currentPath = (Get-ItemProperty -Path $regPath -Name Path -ErrorAction SilentlyContinue).Path
} catch {
    $currentPath = $null
}

$pathUpdated = $false
if (-not $currentPath -or $currentPath -notlike "*$InstallDir*") {
    $newPath = if ($currentPath) { "$currentPath;$InstallDir" } else { $InstallDir }
    Set-ItemProperty -Path $regPath -Name Path -Value $newPath
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    $pathUpdated = $true
}

# Update current session PATH
if ($env:PATH -notlike "*$InstallDir*") {
    $env:PATH = "$InstallDir;$env:PATH"
}

if ($pathUpdated) {
    Write-Host "  Added $InstallDir to User PATH." -ForegroundColor Green
}

# ============================================================================
# PowerShell profile: vg / vectraguard aliases
# ============================================================================

$profilePath = $PROFILE
$aliasBlock = @'

# Vectra Guard aliases (auto-generated)
Set-Alias -Name vg -Value vectra-guard -Scope Global -ErrorAction SilentlyContinue
Set-Alias -Name vectraguard -Value vectra-guard -Scope Global -ErrorAction SilentlyContinue
'@

$aliasAdded = $false
if (Test-Path -LiteralPath $profilePath) {
    $profileContent = Get-Content -LiteralPath $profilePath -Raw
    if ($profileContent -notmatch 'alias.*vg.*vectra-guard') {
        Add-Content -LiteralPath $profilePath -Value $aliasBlock
        $aliasAdded = $true
    }
} else {
    # Create profile directory if needed
    $profileDir = Split-Path -Parent $profilePath
    if (-not (Test-Path -LiteralPath $profileDir)) {
        New-Item -ItemType Directory -Path $profileDir -Force | Out-Null
    }
    Set-Content -LiteralPath $profilePath -Value $aliasBlock
    $aliasAdded = $true
}

# Set aliases in current session too
try {
    Set-Alias -Name vg -Value vectra-guard -Scope Global -ErrorAction SilentlyContinue
    Set-Alias -Name vectraguard -Value vectra-guard -Scope Global -ErrorAction SilentlyContinue
} catch {}

if ($aliasAdded) {
    Write-Host '  Added vg / vectraguard aliases to PowerShell profile.' -ForegroundColor Green
}

# ============================================================================
# Verify
# ============================================================================

Write-Host ''
Write-Host '  Verifying installation...' -ForegroundColor Cyan

try {
    $newVersion = & $BinaryPath version 2>$null | Select-Object -First 1
    if ($newVersion) {
        Write-Host "  vectra-guard installed successfully!" -ForegroundColor Green
        if ($IsUpgrade -and $oldVersion) {
            Write-Host "    Old: $oldVersion"
        }
        Write-Host "    Version: $newVersion"
    } else {
        Write-Host '  Installed, but could not read version.' -ForegroundColor Yellow
    }
} catch {
    Write-Host '  Installed, but verification failed. Restart PowerShell and try: vectra-guard version' -ForegroundColor Yellow
}

# ============================================================================
# Optional: PowerShell protection (shell tracker)
# ============================================================================

if (-not $NonInteractive) {
    Write-Host ''
    if (Confirm-Prompt -Message 'Enable PowerShell Protection (session tracking + command logging)?' -Default $false) {
        Write-Host ''
        try {
            $protectionUrl = "https://raw.githubusercontent.com/$Repo/main/scripts/install-powershell-protection.ps1"
            Invoke-Expression (Invoke-WebRequest -Uri $protectionUrl -UseBasicParsing).Content
        } catch {
            Write-Host "  Could not install PowerShell protection: $_" -ForegroundColor Yellow
            Write-Host "  You can install it later with:" -ForegroundColor Yellow
            Write-Host "    irm https://raw.githubusercontent.com/$Repo/main/scripts/install-powershell-protection.ps1 | iex"
        }
    }
}

# ============================================================================
# Quick start
# ============================================================================

Write-Host ''
Write-Host '  Quick Start:' -ForegroundColor Cyan
Write-Host ''
Write-Host '    1. Validate a script (safe - never executes):'
Write-Host '       vectra-guard validate my-script.ps1' -ForegroundColor Yellow
Write-Host ''
Write-Host '    2. Execute commands safely:'
Write-Host '       vg exec -- npm install' -ForegroundColor Yellow
Write-Host ''
Write-Host '    3. Initialize global config:'
Write-Host '       vectra-guard init    # creates ~\.config\vectra-guard\config.yaml' -ForegroundColor Yellow
Write-Host ''
Write-Host '    4. CVE scan dependencies:'
Write-Host '       vg cve sync --path . && vg cve scan --path .' -ForegroundColor Yellow
Write-Host ''
Write-Host "    Docs: https://github.com/$Repo" -ForegroundColor DarkGray
Write-Host "    Uninstall: Remove-Item -Recurse $InstallDir; # then remove from PATH" -ForegroundColor DarkGray
Write-Host ''
