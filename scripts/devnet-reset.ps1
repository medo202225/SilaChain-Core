Set-Location "C:\Users\AHMED IBRAHIM\Desktop\SilaChain"

Write-Host "Resetting devnet data..."

$paths = @(
    ".\data\node",
    ".\data\node2",
    ".\data\node3"
)

foreach ($item in $paths) {
    if (Test-Path $item) {
        Remove-Item $item -Recurse -Force -ErrorAction SilentlyContinue
        Write-Host "Removed $item"
    }
}

Write-Host "Devnet reset completed."
