param(
    [string]$SourceDir = ".\data\node",
    [string]$OutDir = ".\snapshots"
)

$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$target = Join-Path $OutDir "node_snapshot_$timestamp"

New-Item -ItemType Directory -Force $OutDir | Out-Null
Copy-Item $SourceDir $target -Recurse -Force

Write-Host "Snapshot created:"
Write-Host $target