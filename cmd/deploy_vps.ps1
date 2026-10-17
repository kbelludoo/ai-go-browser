param (
    [string]$KeyPath = "c:\Users\kbell\Downloads\bot-template-go\ssh-key-2026-10-06.key",
    [string]$VPS = "ubuntu@147.15.109.14"
)

$ErrorActionPreference = "Stop"

Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host "🚀 DEPLOY DO AI GO BROWSER NA VPS ORACLE CLOUD" -ForegroundColor Cyan
Write-Host "   Chave SSH: $KeyPath" -ForegroundColor Cyan
Write-Host "   Destino:   $VPS" -ForegroundColor Cyan
Write-Host "================================================================================" -ForegroundColor Cyan

# 1. Configurar chave temporária com permissões estritas
$tempKey = "$env:TEMP\deploy_key_vps.key"
Copy-Item -Path $KeyPath -Destination $tempKey -Force
icacls $tempKey /inheritance:r | Out-Null
icacls $tempKey /grant:r "${env:USERNAME}:(R)" | Out-Null

# 2. Compilar binários Linux amd64
Write-Host "[1/5] Compilando binários Go para Linux x86_64..." -ForegroundColor Yellow
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

if (-not (Test-Path "bin-linux")) {
    New-Item -ItemType Directory -Path "bin-linux" | Out-Null
}

go build -ldflags="-s -w" -o bin-linux/arenaservice ./cmd/arenaservice
Write-Host "  ✅ arenaservice compilado com sucesso." -ForegroundColor Green

go build -ldflags="-s -w" -o bin-linux/mcp-server ./cmd/mcp-server
Write-Host "  ✅ mcp-server compilado com sucesso." -ForegroundColor Green

go build -ldflags="-s -w" -o bin-linux/aigobrowser ./cmd/aigobrowser
Write-Host "  ✅ aigobrowser compilado com sucesso." -ForegroundColor Green

# Restaurar ambiente Windows
$env:GOOS = "windows"
$env:GOARCH = "amd64"

# 3. Preparar diretórios remotos na VPS
Write-Host "[2/5] Garantindo diretórios na VPS..." -ForegroundColor Yellow
ssh -i $tempKey -o StrictHostKeyChecking=no $VPS "mkdir -p /home/ubuntu/ai-go-browser/bin /home/ubuntu/ai-go-browser/docs /home/ubuntu/.aigobrowser/sessions"

# 4. Parar serviço antigo temporariamente para substituir binários
Write-Host "[3/5] Parando serviço antigo na VPS..." -ForegroundColor Yellow
ssh -i $tempKey -o StrictHostKeyChecking=no $VPS "pkill -f arenaservice || true"

# 5. Fazer upload dos binários e arquivos estáticos via SCP
Write-Host "[4/5] Enviando binários e arquivos atualizados via SCP..." -ForegroundColor Yellow
scp -i $tempKey -o StrictHostKeyChecking=no bin-linux/arenaservice "${VPS}:/home/ubuntu/ai-go-browser/bin/arenaservice"
scp -i $tempKey -o StrictHostKeyChecking=no bin-linux/mcp-server "${VPS}:/home/ubuntu/ai-go-browser/bin/mcp-server"
scp -i $tempKey -o StrictHostKeyChecking=no bin-linux/aigobrowser "${VPS}:/home/ubuntu/ai-go-browser/bin/aigobrowser"
scp -i $tempKey -o StrictHostKeyChecking=no docs/index.html "${VPS}:/home/ubuntu/ai-go-browser/docs/index.html"

ssh -i $tempKey -o StrictHostKeyChecking=no $VPS "chmod +x /home/ubuntu/ai-go-browser/bin/*"

# 6. Criar ou atualizar serviço systemd para execução 24/7 sem navegador pesado
Write-Host "[5/5] Iniciando serviço arenaservice no modo zero-browser (porta :8080)..." -ForegroundColor Yellow

$serviceContent = @"
[Unit]
Description=AI Go Browser Headless Cloud Service
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/ai-go-browser
ExecStart=/home/ubuntu/ai-go-browser/bin/arenaservice -no-browser -addr :8080
Restart=always
RestartSec=5s
LimitNOFILE=65535
Environment=ARENA_NO_BROWSER=1
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
"@

# Salvar e ativar systemd na VPS
$remoteServiceScript = "echo '$serviceContent' | sudo tee /etc/systemd/system/arenaservice.service > /dev/null; sudo systemctl daemon-reload; sudo systemctl enable arenaservice; sudo systemctl restart arenaservice; sleep 2; sudo systemctl status arenaservice --no-pager"
ssh -i $tempKey -o StrictHostKeyChecking=no $VPS $remoteServiceScript

Write-Host "`n================================================================================" -ForegroundColor Cyan
Write-Host "✅ DEPLOY CONCLUÍDO COM SUCESSO!" -ForegroundColor Green
Write-Host "   Navegador e API ativos 24/7 na VPS:" -ForegroundColor White
Write-Host "   👉 http://147.15.109.14:8080/" -ForegroundColor Cyan
Write-Host "   👉 MCP Server pronto em: /home/ubuntu/ai-go-browser/bin/mcp-server" -ForegroundColor White
Write-Host "================================================================================" -ForegroundColor Cyan
