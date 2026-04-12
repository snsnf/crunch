$ErrorActionPreference = "Stop"

$repo = "snsnf/crunch"
$installDir = "$env:LOCALAPPDATA\Crunch"

Write-Host "Installing Crunch CLI..." -ForegroundColor Cyan

# Get latest release tag
$release = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
$tag = $release.tag_name
Write-Host "Latest version: $tag"

# Find CLI asset
$asset = $release.assets | Where-Object { $_.name -like "crunch-cli-windows-amd64.zip" }
if (-not $asset) {
    Write-Host "Error: Could not find Windows CLI release." -ForegroundColor Red
    exit 1
}

# Download
$zipPath = "$env:TEMP\crunch-cli.zip"
Write-Host "Downloading..."
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zipPath

# Verify checksum if available
$checksumAsset = $release.assets | Where-Object { $_.name -eq "checksums.txt" }
if ($checksumAsset) {
    $checksumFile = "$env:TEMP\crunch-checksums.txt"
    Invoke-WebRequest -Uri $checksumAsset.browser_download_url -OutFile $checksumFile
    $expectedLine = Get-Content $checksumFile | Where-Object { $_ -like "*crunch-cli-windows-amd64.zip*" }
    if ($expectedLine) {
        $expectedHash = ($expectedLine -split '\s+')[0]
        $actualHash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToLower()
        if ($actualHash -ne $expectedHash) {
            Write-Host "Error: Checksum verification failed!" -ForegroundColor Red
            Write-Host "  Expected: $expectedHash"
            Write-Host "  Got:      $actualHash"
            Remove-Item $zipPath, $checksumFile -ErrorAction SilentlyContinue
            exit 1
        }
        Write-Host "Checksum verified." -ForegroundColor Green
    }
    Remove-Item $checksumFile -ErrorAction SilentlyContinue
}

# Extract
if (Test-Path $installDir) { Remove-Item $installDir -Recurse -Force }
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Expand-Archive -Path $zipPath -DestinationPath $installDir -Force
Remove-Item $zipPath

# Verify
if (-not (Test-Path "$installDir\crunch.exe")) {
    Write-Host "Error: Installation failed." -ForegroundColor Red
    exit 1
}

# Add to PATH if not already there
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Added to PATH." -ForegroundColor Green
}

# Update current session
$env:Path = "$env:Path;$installDir"

Write-Host ""
Write-Host "Crunch $tag installed!" -ForegroundColor Green
Write-Host "Restart your terminal, then run: crunch --help" -ForegroundColor Yellow
