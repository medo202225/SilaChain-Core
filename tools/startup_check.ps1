param(
    [string]$BaseUrl = "http://127.0.0.1:8090"
)

Write-Host "== health =="
Invoke-WebRequest "$BaseUrl/health" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== network status =="
Invoke-WebRequest "$BaseUrl/network/status" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== explorer summary =="
Invoke-WebRequest "$BaseUrl/explorer/summary" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== validators active =="
Invoke-WebRequest "$BaseUrl/validators/active" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== mempool count =="
Invoke-WebRequest "$BaseUrl/mempool/count" -UseBasicParsing | Select-Object -Expand Content