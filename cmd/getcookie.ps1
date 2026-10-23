Add-Type -AssemblyName System.Security

# 1. Master Key
$localStatePath = [System.IO.Path]::Combine($env:LOCALAPPDATA, 'BraveSoftware\Brave-Browser\User Data\Local State')
$localState = Get-Content $localStatePath -Raw | ConvertFrom-Json
$encryptedKey = [System.Convert]::FromBase64String($localState.os_crypt.encrypted_key)
$rawKey = $encryptedKey[5..($encryptedKey.Length - 1)]
$masterKey = [System.Security.Cryptography.ProtectedData]::Unprotect($rawKey, $null, [System.Security.Cryptography.DataProtectionScope]::CurrentUser)

# 2. Cookies DB
$cookiesPath = [System.IO.Path]::Combine($env:LOCALAPPDATA, 'BraveSoftware\Brave-Browser\User Data\Default\Network\Cookies')
$fs = New-Object System.IO.FileStream($cookiesPath, [System.IO.FileMode]::Open, [System.IO.FileAccess]::Read, [System.IO.FileShare]::ReadWrite)
$bytes = New-Object byte[] $fs.Length
$fs.Read($bytes, 0, $fs.Length) | Out-Null
$fs.Close()
$target = [System.Text.Encoding]::ASCII.GetBytes("arena-auth-prod-v1.0")

# Find target in bytes
$foundIdx = -1
for ($i = 0; $i -lt ($bytes.Length - $target.Length); $i++) {
    $match = $true
    for ($j = 0; $j -lt $target.Length; $j++) {
        if ($bytes[$i + $j] -ne $target[$j]) {
            $match = $false
            break
        }
    }
    if ($match) {
        $foundIdx = $i
        break
    }
}

if ($foundIdx -eq -1) {
    Write-Host "Target cookie not found in SQLite bytes"
    exit 1
}

Write-Host "Found cookie record at offset $foundIdx"

# Look for 'v10' within 200 bytes
$v10Idx = -1
for ($i = $foundIdx; $i -lt ($foundIdx + 200); $i++) {
    if ($bytes[$i] -eq 118 -and $bytes[$i+1] -eq 49 -and $bytes[$i+2] -eq 48) { # 'v10'
        $v10Idx = $i
        break
    }
}

if ($v10Idx -eq -1) {
    Write-Host "v10 prefix not found"
    exit 1
}

Write-Host "v10 at $v10Idx"
$encVal = $bytes[$v10Idx..($v10Idx + 4500)]
$nonce = $encVal[3..14] # 12 bytes nonce

# AES-GCM in .NET
Add-Type -TypeDefinition @"
using System;
using System.Security.Cryptography;
using System.Text;

public class Decryptor {
    public static string Decrypt(byte[] key, byte[] nonce, byte[] ciphertext, byte[] tag) {
        using (var aes = new AesGcm(key)) {
            byte[] plaintext = new byte[ciphertext.Length];
            aes.Decrypt(nonce, ciphertext, tag, plaintext);
            return Encoding.UTF8.GetString(plaintext);
        }
    }
}
"@

# Brute search length
for ($len = 1000; $len -le 4000; $len++) {
    $ct = $encVal[15..($len - 17)]
    $tag = $encVal[($len - 16)..($len - 1)]
    try {
        $res = [Decryptor]::Decrypt($masterKey, $nonce, $ct, $tag)
        if ($res.StartsWith("base64-")) {
            Write-Host "DECRYPTED COOKIE SUCCESSFULLY! Length = $($res.Length)"
            $outPath = [System.IO.Path]::Combine($env:USERPROFILE, ".arena_cookie")
            [System.IO.File]::WriteAllText($outPath, $res)
            Write-Host "Saved to $outPath"
            exit 0
        }
    } catch {}
}

Write-Host "Could not match tag"
