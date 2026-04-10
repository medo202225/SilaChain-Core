param(
    [string]$BaseUrl = "http://127.0.0.1:8090"
)

Write-Host "== validators =="
Invoke-WebRequest "$BaseUrl/validators" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== active validators =="
Invoke-WebRequest "$BaseUrl/validators/active" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== weighted validators =="
Invoke-WebRequest "$BaseUrl/validators/weighted" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== consensus proposer =="
Invoke-WebRequest "$BaseUrl/consensus/proposer" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== consensus rotation =="
Invoke-WebRequest "$BaseUrl/consensus/rotation" -UseBasicParsing | Select-Object -Expand Content