param(
    [Parameter(Mandatory = $true)]
    [string]$PackagePath
)

$ErrorActionPreference = "Stop"

if (!(Test-Path $PackagePath)) {
    throw "Package path not found: $PackagePath"
}

$requiredFiles = @(
    "config\mainnet\node.json",
    "config\mainnet\validator.key",
    "config\mainnet\protocol.json",
    "config\mainnet\validators.json",
    "config\mainnet\bootnodes.json",
    "config\mainnet\peers.json",
    "validator-manifest.json",
    "README.txt"
)

$missing = @()

foreach ($file in $requiredFiles) {
    $fullPath = Join-Path $PackagePath $file
    if (!(Test-Path $fullPath)) {
        $missing += $file
    }
}

if ($missing.Count -gt 0) {
    Write-Host "Package validation failed. Missing files:"
    $missing | ForEach-Object { Write-Host "- $_" }
    exit 1
}

Write-Host "Package validation passed."
Write-Host "All required files are present."
