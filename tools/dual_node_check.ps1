$node1 = "http://127.0.0.1:8090"
$node2 = "http://127.0.0.1:8091"

Write-Host "== node1 health =="
Invoke-WebRequest "$node1/health" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== node1 chain info =="
Invoke-WebRequest "$node1/chain/info" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== node2 health =="
Invoke-WebRequest "$node2/health" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== node2 chain info =="
Invoke-WebRequest "$node2/chain/info" -UseBasicParsing | Select-Object -Expand Content