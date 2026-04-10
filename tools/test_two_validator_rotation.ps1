param(
    [string]$BaseUrl = "http://127.0.0.1:8090"
)

$w2 = Get-Content .\data\wallet\keystore\wallet2.json | ConvertFrom-Json

Write-Host "== before =="
Invoke-WebRequest "$BaseUrl/consensus/rotation" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== tx 1 =="
go run ./cmd/sila-wallet send .\data\wallet\keystore\wallet1.json $BaseUrl $w2.address 1 1 1001
Start-Sleep -Seconds 6

Write-Host "`n== after tx 1 =="
Invoke-WebRequest "$BaseUrl/chain/info" -UseBasicParsing | Select-Object -Expand Content
Invoke-WebRequest "$BaseUrl/consensus/proposer" -UseBasicParsing | Select-Object -Expand Content
Invoke-WebRequest "$BaseUrl/consensus/rotation" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== tx 2 =="
go run ./cmd/sila-wallet send .\data\wallet\keystore\wallet1.json $BaseUrl $w2.address 1 1 1001
Start-Sleep -Seconds 6

Write-Host "`n== after tx 2 =="
Invoke-WebRequest "$BaseUrl/chain/info" -UseBasicParsing | Select-Object -Expand Content
Invoke-WebRequest "$BaseUrl/consensus/proposer" -UseBasicParsing | Select-Object -Expand Content
Invoke-WebRequest "$BaseUrl/consensus/rotation" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== tx 3 =="
go run ./cmd/sila-wallet send .\data\wallet\keystore\wallet1.json $BaseUrl $w2.address 1 1 1001
Start-Sleep -Seconds 6

Write-Host "`n== after tx 3 =="
Invoke-WebRequest "$BaseUrl/chain/info" -UseBasicParsing | Select-Object -Expand Content
Invoke-WebRequest "$BaseUrl/consensus/proposer" -UseBasicParsing | Select-Object -Expand Content
Invoke-WebRequest "$BaseUrl/consensus/rotation" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== final weighted validators =="
Invoke-WebRequest "$BaseUrl/validators/weighted" -UseBasicParsing | Select-Object -Expand Content