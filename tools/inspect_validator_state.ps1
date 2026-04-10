param(
    [string]$BaseUrl = "http://127.0.0.1:8090"
)

Write-Host "== validators =="
Invoke-WebRequest "$BaseUrl/validators" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== active validators =="
Invoke-WebRequest "$BaseUrl/validators/active" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== weighted validators =="
Invoke-WebRequest "$BaseUrl/validators/weighted" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== staking =="
Invoke-WebRequest "$BaseUrl/staking" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== delegations =="
Invoke-WebRequest "$BaseUrl/staking/delegations" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== jails =="
Invoke-WebRequest "$BaseUrl/staking/jails" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== rotation =="
Invoke-WebRequest "$BaseUrl/consensus/rotation" -UseBasicParsing | Select-Object -Expand Content