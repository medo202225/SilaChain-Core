$node1 = "http://127.0.0.1:8090"
$node2 = "http://127.0.0.1:8091"

Write-Host "== node1 chain info =="
Invoke-WebRequest "$node1/chain/info" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== node1 sync status =="
Invoke-WebRequest "$node1/sync/status" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== node2 chain info =="
Invoke-WebRequest "$node2/chain/info" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== node2 sync status =="
Invoke-WebRequest "$node2/sync/status" -UseBasicParsing | Select-Object -Expand Content