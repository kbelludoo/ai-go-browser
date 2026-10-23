Add-Type -AssemblyName System.Security

$localStatePath = [System.IO.Path]::Combine($env:LOCALAPPDATA, 'BraveSoftware\Brave-Browser\User Data\Local State')
$localState = Get-Content $localStatePath -Raw | ConvertFrom-Json
$encryptedKey = [System.Convert]::FromBase64String($localState.os_crypt.encrypted_key)
$rawKey = $encryptedKey[5..($encryptedKey.Length - 1)]
$masterKey = [System.Security.Cryptography.ProtectedData]::Unprotect($rawKey, $null, [System.Security.Cryptography.DataProtectionScope]::CurrentUser)

Write-Host "Master key length: $($masterKey.Length)"
