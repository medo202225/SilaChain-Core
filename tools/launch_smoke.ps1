param(
    [string]$BaseUrl = "http://127.0.0.1:8090"
)

Write-Host "== health =="
Invoke-WebRequest "$BaseUrl/health" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== chain info =="
Invoke-WebRequest "$BaseUrl/chain/info" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== explorer summary =="
Invoke-WebRequest "$BaseUrl/explorer/summary" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== validators =="
Invoke-WebRequest "$BaseUrl/validators" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== active validators =="
Invoke-WebRequest "$BaseUrl/validators/active" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== staking =="
Invoke-WebRequest "$BaseUrl/staking" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== delegations =="
Invoke-WebRequest "$BaseUrl/staking/delegations" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== undelegations =="
Invoke-WebRequest "$BaseUrl/staking/undelegations" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== rewards =="
Invoke-WebRequest "$BaseUrl/staking/rewards" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== delegator rewards =="
Invoke-WebRequest "$BaseUrl/staking/rewards/delegators" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== jails =="
Invoke-WebRequest "$BaseUrl/staking/jails" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== mempool count =="
Invoke-WebRequest "$BaseUrl/mempool/count" -UseBasicParsing | Select-Object -Expand Content