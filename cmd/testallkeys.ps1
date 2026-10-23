$keys = @(
    "c:\Users\kbell\Downloads\bot-template-go\ssh-key-2026-10-06.key",
    "c:\Users\kbell\Downloads\ai-go-browser-colecao-completa\ssh-key-2026-10-09 sao paulo.key",
    "c:\Users\kbell\Downloads\ssh-key-2026-10-10.key",
    "c:\Users\kbell\Downloads\restream-go\ssh-key-2026-10-12.key"
)

$vpsIP = "147.15.109.14"

foreach ($k in $keys) {
    if (Test-Path $k) {
        $name = [System.IO.Path]::GetFileName($k)
        $dst = "$env:TEMP\key_$name.key"
        Copy-Item $k $dst -Force
        icacls $dst /inheritance:r | Out-Null
        icacls $dst /grant:r "${env:USERNAME}:(R)" | Out-Null
        
        Write-Host "Testing $name with $vpsIP..."
        $out = ssh -i $dst -o StrictHostKeyChecking=no -o ConnectTimeout=4 ubuntu@$vpsIP "echo SUCCESS_UBUNTU" 2>&1
        Write-Host "  ubuntu: $out"
        $out2 = ssh -i $dst -o StrictHostKeyChecking=no -o ConnectTimeout=4 opc@$vpsIP "echo SUCCESS_OPC" 2>&1
        Write-Host "  opc: $out2"
    }
}
