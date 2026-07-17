# install.ps1 — PowerShell installer for drift (Windows).
#
# Usage (from PowerShell):
#   irm https://raw.githubusercontent.com/SteelSprint/Drift/main/scripts/install.ps1 | iex
#
# Or pin a version:
#   $env:DRIFT_VERSION='v1.0.0'; irm https://raw.githubusercontent.com/SteelSprint/Drift/main/scripts/install.ps1 | iex
#
# Installs to $env:USERPROFILE\.local\bin\drift.exe (override with $env:DESTDIR).
# Prints a PATH hint if the install location is not on $env:PATH.
$ErrorActionPreference = "Stop"

$Repo = "SteelSprint/Drift"

# --- detect arch ---
# AMD64 → amd64, ARM64 → arm64, x86 → 386
switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { $GOARCH = "amd64" }
    "ARM64" { $GOARCH = "arm64" }
    "x86"   { $GOARCH = "386" }
    default { throw "install: error: unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

Write-Host "Detected: windows/$GOARCH"

# --- resolve version ---
$Tag = $env:DRIFT_VERSION
if (-not $Tag) {
    Write-Host "Fetching latest release..."
    $latest = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Tag = $latest.tag_name
}
if (-not $Tag) {
    throw "install: error: could not determine latest release tag from GitHub API"
}

# --- validate TAG: must be vMAJOR.MINOR.PATCH with optional pre-release/build suffix ---
if ($Tag -notmatch '^v\d+\.\d+\.\d+([+-][A-Za-z0-9._-]+)?$') {
    throw "install: error: invalid version tag '$Tag'; expected format like v1.0.0"
}

# strip leading 'v' for the archive version string
$Ver = $Tag.TrimStart('v')
$Archive = "drift_${Ver}_windows_${GOARCH}"
$Url = "https://github.com/$Repo/releases/download/$Tag/$Archive.zip"

Write-Host "Version: $Tag"
Write-Host "Downloading: $Url"

# --- resolve install location ---
if (-not $env:USERPROFILE -and -not $env:DESTDIR) {
    throw "install: error: USERPROFILE is unset and DESTDIR is not set; refusing to install. Set `$env:DESTDIR explicitly."
}
$DestDir = $env:DESTDIR
if (-not $DestDir) {
    $DestDir = Join-Path $env:USERPROFILE ".local\bin"
}

# --- create temp dir ---
$Tmp = Join-Path ([System.IO.Path]::GetTempPath()) "drift-install-$([System.Guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Path $Tmp -Force | Out-Null

try {
    # --- download archive ---
    Invoke-WebRequest -Uri $Url -OutFile (Join-Path $Tmp "$Archive.zip") -UseBasicParsing

    # --- verify checksum (strict: checksums.txt must be present AND list our archive) ---
    $ChecksumUrl = "https://github.com/$Repo/releases/download/$Tag/checksums.txt"
    try {
        Invoke-WebRequest -Uri $ChecksumUrl -OutFile (Join-Path $Tmp "checksums.txt") -UseBasicParsing
    } catch {
        throw "install: error: could not download checksums.txt. Refusing to install without checksum verification."
    }

    $checksumLine = Select-String -Path (Join-Path $Tmp "checksums.txt") -SimpleMatch "$Archive.zip" -ErrorAction SilentlyContinue
    if (-not $checksumLine) {
        throw "install: error: checksums.txt is available but does not list '$Archive.zip'. This is either a release packaging error or a tampering attempt — refusing to install."
    }
    $Expected = ($checksumLine[0].Line -split '\s+')[0]
    if (-not $Expected) {
        throw "install: error: could not parse expected checksum from checksums.txt"
    }

    $Actual = (Get-FileHash -Path (Join-Path $Tmp "$Archive.zip") -Algorithm SHA256).Hash.ToLower()
    if ($Actual -ne $Expected.ToLower()) {
        throw "install: error: checksum mismatch — the downloaded archive was tampered with or corrupted.`nexpected: $Expected`nactual:   $Actual"
    }
    Write-Host "Checksum verified."

    # --- extract only drift.exe from the zip ---
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [System.IO.Compression.ZipFile]::OpenRead((Join-Path $Tmp "$Archive.zip"))
    try {
        $entry = $zip.Entries | Where-Object { $_.FullName -eq "drift.exe" -or $_.FullName -eq "drift" }
        if (-not $entry) {
            throw "install: error: archive did not contain a 'drift.exe' binary at its root"
        }
        [System.IO.Compression.ZipFileExtensions]::ExtractToFile($entry, (Join-Path $Tmp "drift.exe"), $true)
    } finally {
        $zip.Dispose()
    }

    if (-not (Test-Path (Join-Path $Tmp "drift.exe"))) {
        throw "install: error: extraction succeeded but drift.exe is missing"
    }

    # --- install (atomic) ---
    New-Item -ItemType Directory -Force -Path $DestDir | Out-Null
    $InstallPath = Join-Path $DestDir "drift.exe"

    # if INSTALL_PATH is a directory, refuse
    if (Test-Path $InstallPath -PathType Container) {
        throw "install: error: $InstallPath is a directory, not a file; refusing to overwrite. Remove it first."
    }

    # backup copy of existing binary (copy, not move — original stays until atomic rename)
    if (Test-Path $InstallPath -PathType Leaf) {
        Copy-Item $InstallPath "$InstallPath.old" -Force -ErrorAction SilentlyContinue
    }

    # atomic install: Move-Item with -Force uses MoveFileEx (atomic on same volume)
    Move-Item (Join-Path $Tmp "drift.exe") $InstallPath -Force

    Write-Host ""
    Write-Host "Installed drift $Tag -> $InstallPath"
    if (Test-Path "$InstallPath.old") {
        Write-Host "Previous version backed up to $InstallPath.old"
    }

    # --- PATH hint ---
    $pathSep = ";"
    $pathDirs = $env:PATH -split [regex]::Escape($pathSep)
    if ($DestDir -notin $pathDirs) {
        Write-Host ""
        Write-Host "WARNING: $DestDir is not on your PATH."
        Write-Host "Add it to your PATH permanently:"
        Write-Host "  [Environment]::SetEnvironmentVariable('PATH', `"$DestDir;`$([Environment]::GetEnvironmentVariable('PATH', 'User'))`", 'User')"
    }

    # --- verify ---
    try {
        & $InstallPath version
        Write-Host ""
        Write-Host "Run 'drift help' to get started."
    } catch {
        Write-Host ""
        Write-Host "Installed, but 'drift version' failed. Run '$InstallPath help' to check."
    }
} finally {
    # cleanup temp dir
    if (Test-Path $Tmp) {
        Remove-Item $Tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}
