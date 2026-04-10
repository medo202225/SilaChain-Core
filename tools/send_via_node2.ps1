$w2 = Get-Content .\data\wallet\keystore\wallet2.json | ConvertFrom-Json
go run ./cmd/sila-wallet send .\data\wallet\keystore\wallet1.json http://127.0.0.1:8091 $w2.address 1 1 1001