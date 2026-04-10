param(
    [string]$SourceDir = ".\data\node",
    [string]$TargetDir = ".\data\node2"
)

if (Test-Path $TargetDir) {
    Remove-Item $TargetDir -Recurse -Force
}

Copy-Item $SourceDir $TargetDir -Recurse -Force

Write-Host "Node2 data prepared:"
Write-Host $TargetDir