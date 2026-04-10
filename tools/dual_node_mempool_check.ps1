$node1 = "http://127.0.0.1:8090"
$node2 = "http://127.0.0.1:8091"

Write-Host "== node1 mempool count =="
Invoke-WebRequest "$node1/mempool/count" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== node1 sync mempool =="
Invoke-WebRequest "$node1/sync/mempool" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== node2 mempool count =="
Invoke-WebRequest "$node2/mempool/count" -UseBasicParsing | Select-Object -Expand Content

Write-Host "`n== node2 sync mempool =="
Invoke-WebRequest "$node2/sync/mempool" -UseBasicParsing | Select-Object -Expand Content