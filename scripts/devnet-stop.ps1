Write-Host "Stopping SilaChain devnet processes..."

Get-Process | Where-Object {
    $_.ProcessName -match "go|sila-node"
} | ForEach-Object {
    try {
        Stop-Process -Id $_.Id -Force -ErrorAction Stop
        Write-Host "Stopped PID $($_.Id) ($($_.ProcessName))"
    } catch {
    }
}

Write-Host "Stop command completed."
