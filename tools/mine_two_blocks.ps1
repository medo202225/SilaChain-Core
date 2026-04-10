param(
    [string]$BaseUrl = "http://127.0.0.1:8090"
)

$w2 = Get-Content .\data\wallet\keystore\wallet2.json | ConvertFrom-Json

go run ./cmd/sila-wallet send .\data\wallet\keystore\wallet1.json $BaseUrl $w2.address 1 1 1001
Start-Sleep -Seconds 6

go run ./cmd/sila-wallet send .\data\wallet\keystore\wallet1.json $BaseUrl $w2.address 1 1 1001
Start-Sleep -Seconds 6

Write-Host "`n== chain info =="
Invoke-WebRequest "$BaseUrl/chain/info" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== proposer =="
Invoke-WebRequest "$BaseUrl/consensus/proposer" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== rotation =="
Invoke-WebRequest "$BaseUrl/consensus/rotation" -UseBasicParsing | Select-Object -Expand Content