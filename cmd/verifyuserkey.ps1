$key = "c:\Users\kbell\Downloads\bot-template-go\ssh-key-2026-10-06.key"
$tempKey = "$env:TEMP\user_requested_key.key"
Copy-Item $key $tempKey -Force
icacls $tempKey /inheritance:r | Out-Null
icacls $tempKey /grant:r "${env:USERNAME}:(R)" | Out-Null

Write-Host "Testing connection with user requested key: $key..."
ssh -i $tempKey -o StrictHostKeyChecking=no ubuntu@147.15.109.14 "uname -a; uptime; free -h; ps aux | grep arenaservice | grep -v grep"
