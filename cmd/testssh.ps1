$key = "c:\Users\kbell\Downloads\bot-template-go\ssh-key-2026-10-06.key"
$tempKey = "$env:TEMP\vps_deploy.key"
Copy-Item -Path $key -Destination $tempKey -Force
icacls $tempKey /inheritance:r | Out-Null
icacls $tempKey /grant:r "${env:USERNAME}:(R)" | Out-Null

Write-Host "Testing ssh as opc@147.15.109.14..."
ssh -i $tempKey -o StrictHostKeyChecking=no -o ConnectTimeout=5 opc@147.15.109.14 "uname -a; uptime; free -m"

Write-Host "Testing ssh as root@147.15.109.14..."
ssh -i $tempKey -o StrictHostKeyChecking=no -o ConnectTimeout=5 root@147.15.109.14 "uname -a; uptime; free -m"
