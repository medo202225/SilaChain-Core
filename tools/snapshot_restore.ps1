param(
    [Parameter(Mandatory = $true)]
    [string]$SnapshotPath,

    [string]$TargetDir = ".\data\node",
    [string]$BackupRoot = ".\backups"
)

if (-not (Test-Path $SnapshotPath)) {
    Write-Error "Snapshot path not found: $SnapshotPath"
    exit 1
}

$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$backupDir = Join-Path $BackupRoot "node_backup_$timestamp"

New-Item -ItemType Directory -Force $BackupRoot | Out-Null

if (Test-Path $TargetDir) {
    Copy-Item $TargetDir $backupDir -Recurse -Force
    Remove-Item $TargetDir -Recurse -Force
}

Copy-Item $SnapshotPath $TargetDir -Recurse -Force

Write-Host "Backup created:"
Write-Host $backupDir
Write-Host ""
Write-Host "Snapshot restored to:"
Write-Host $TargetDir