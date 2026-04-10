param(
    [string]$BaseUrl = "http://127.0.0.1:8090",
    [int]$Loops = 12,
    [int]$SleepSeconds = 5
)

for ($i = 1; $i -le $Loops; $i++) {
    Write-Host ""
    Write-Host ("==== sample #{0} ====" -f $i)

    Write-Host "-- chain info --"
    Invoke-WebRequest "$BaseUrl/chain/info" -UseBasicParsing | Select-Object -Expand Content

    Write-Host "-- proposer --"
    Invoke-WebRequest "$BaseUrl/consensus/proposer" -UseBasicParsing | Select-Object -Expand Content

    Write-Host "-- rotation --"
    Invoke-WebRequest "$BaseUrl/consensus/rotation" -UseBasicParsing | Select-Object -Expand Content

    Write-Host "-- mempool count --"
    Invoke-WebRequest "$BaseUrl/mempool/count" -UseBasicParsing | Select-Object -Expand Content

    Start-Sleep -Seconds $SleepSeconds
}