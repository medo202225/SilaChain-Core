Set-Location "C:\Users\AHMED IBRAHIM\Desktop\SilaChain"

Write-Host "Starting SilaChain devnet..."

Start-Process powershell -ArgumentList '-NoExit', '-Command', 'Set-Location "C:\Users\AHMED IBRAHIM\Desktop\SilaChain"; go run ./cmd/sila-node --config .\config\node1.json'
Start-Sleep -Milliseconds 700

Start-Process powershell -ArgumentList '-NoExit', '-Command', 'Set-Location "C:\Users\AHMED IBRAHIM\Desktop\SilaChain"; go run ./cmd/sila-node --config .\config\node2.json'
Start-Sleep -Milliseconds 700

Start-Process powershell -ArgumentList '-NoExit', '-Command', 'Set-Location "C:\Users\AHMED IBRAHIM\Desktop\SilaChain"; go run ./cmd/sila-node --config .\config\node3.json'

Write-Host "Devnet start commands launched."
