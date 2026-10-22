$ErrorActionPreference = "Stop"

Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host "⚡ AI GO BROWSER: REPRODUÇÃO DA AUDITORIA EMPÍRICA INDEPENDENTE" -ForegroundColor Cyan
Write-Host "   Ambiente: Windows | Go: $(go version)" -ForegroundColor Cyan
Write-Host "================================================================================" -ForegroundColor Cyan

New-Item -ItemType Directory -Force -Path bin | Out-Null

Write-Host "`n📦 [1/4] Compilando binários em Go puro..." -ForegroundColor Yellow
go build -o bin/aigobrowser.exe ./cmd/aigobrowser
go build -o bin/wasmbench.exe ./cmd/wasmbench
go build -o bin/external_benchmarks.exe ./cmd/external_benchmarks

Write-Host "`n🧪 [2/4] Executando Autoteste Integrado (-demo)..." -ForegroundColor Yellow
.\bin\aigobrowser.exe -demo

Write-Host "`n🔒 [3/4] Executando Auditoria Forense TLS (tls.peet.ws) & Microbenchmark..." -ForegroundColor Yellow
.\bin\wasmbench.exe

Write-Host "`n🏛️ [4/4] Executando Tarefas do WebArena & Mind2Web..." -ForegroundColor Yellow
.\bin\external_benchmarks.exe

Write-Host "`n================================================================================" -ForegroundColor Green
Write-Host "✅ AUDITORIA CONCLUÍDA COM SUCESSO! Relatórios salvos em benchmark-results/" -ForegroundColor Green
Write-Host "================================================================================" -ForegroundColor Green
