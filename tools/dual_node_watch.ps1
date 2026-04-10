param(
    [int]$Loops = 8,
    [int]$SleepSeconds = 5
)

$node1 = "http://127.0.0.1:8090"
$node2 = "http://127.0.0.1:8091"

for ($i = 1; $i -le $Loops; $i++) {
    Write-Host ""
    Write-Host ("================ sample #{0} ================" -f $i)

    Write-Host "`n== node1 /network/status =="
    Invoke-WebRequest "$node1/network/status" -UseBasicParsing | Select-Object -Expand Content

    Write-Host "`n== node1 /consensus/rotation =="
    Invoke-WebRequest "$node1/consensus/rotation" -UseBasicParsing | Select-Object -Expand Content

    Write-Host "`n== node2 /network/status =="
    Invoke-WebRequest "$node2/network/status" -UseBasicParsing | Select-Object -Expand Content

    Write-Host "`n== node2 /consensus/rotation =="
    Invoke-WebRequest "$node2/consensus/rotation" -UseBasicParsing | Select-Object -Expand Content

    Start-Sleep -Seconds $SleepSeconds
}