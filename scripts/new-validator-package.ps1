param(
    [Parameter(Mandatory = $true)]
    [string]$NodeName,

    [Parameter(Mandatory = $true)]
    [string]$ValidatorAddress,

    [Parameter(Mandatory = $true)]
    [string]$ValidatorKeySource,

    [Parameter(Mandatory = $false)]
    [string]$ListenAddress = "0.0.0.0:8090"
)

$ErrorActionPreference = "Stop"

$projectRoot = (Resolve-Path ".").Path
$mainnetPath = Join-Path $projectRoot "config\mainnet"
$templatePath = Join-Path $mainnetPath "templates\node.template.json"
$sharedPath = Join-Path $mainnetPath "shared"
$packageDir = Join-Path $projectRoot "dist\validator-$NodeName"
$packageConfigDir = Join-Path $packageDir "config\mainnet"

if (!(Test-Path $templatePath)) {
    throw "Template file not found: $templatePath"
}

if (!(Test-Path $ValidatorKeySource)) {
    throw "Validator key source not found: $ValidatorKeySource"
}

New-Item -ItemType Directory -Force -Path $packageConfigDir | Out-Null

$template = Get-Content $templatePath -Raw
$template = $template.Replace("{{NODE_NAME}}", $NodeName)
$template = $template.Replace("{{VALIDATOR_ADDRESS}}", $ValidatorAddress)
$template = $template.Replace("{{LISTEN_ADDRESS}}", $ListenAddress)

$nodeConfigPath = Join-Path $packageConfigDir "node.json"
Set-Content -Path $nodeConfigPath -Value $template -Encoding UTF8

$sharedFiles = @(
    "protocol.json",
    "validators.json",
    "bootnodes.json",
    "peers.json",
    "genesis.json"
)

foreach ($file in $sharedFiles) {
    $source = Join-Path $sharedPath $file
    if (Test-Path $source) {
        Copy-Item $source (Join-Path $packageConfigDir $file) -Force
    }
}

Copy-Item $ValidatorKeySource (Join-Path $packageConfigDir "validator.key") -Force

$manifest = @{
    network = "mainnet"
    node_name = $NodeName
    validator_address = $ValidatorAddress
    listen = $ListenAddress
    generated_at = (Get-Date).ToString("s")
    files = @(
        "config/mainnet/node.json",
        "config/mainnet/validator.key",
        "config/mainnet/protocol.json",
        "config/mainnet/validators.json",
        "config/mainnet/bootnodes.json",
        "config/mainnet/peers.json"
    )
} | ConvertTo-Json -Depth 5

Set-Content -Path (Join-Path $packageDir "validator-manifest.json") -Value $manifest -Encoding UTF8

$readme = @"
SilaChain Validator Package

Files included:
- config/mainnet/node.json
- config/mainnet/validator.key
- config/mainnet/protocol.json
- config/mainnet/validators.json
- config/mainnet/bootnodes.json
- config/mainnet/peers.json
- validator-manifest.json

Usage:
1. Copy this package into the SilaChain project root on the validator machine.
2. Ensure Go is installed.
3. Run from the project root:
   go run ./cmd/sila-node
"@

Set-Content -Path (Join-Path $packageDir "README.txt") -Value $readme -Encoding UTF8

Write-Host "Validator package created successfully:"
Write-Host $packageDir
