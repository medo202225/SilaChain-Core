Write-Host "== local node data dirs =="
Get-ChildItem .\data -Directory

Write-Host "`n== snapshots =="
if (Test-Path .\snapshots) {
    Get-ChildItem .\snapshots
} else {
    Write-Host "No snapshots directory found."
}

Write-Host "`n== backups =="
if (Test-Path .\backups) {
    Get-ChildItem .\backups
} else {
    Write-Host "No backups directory found."
}