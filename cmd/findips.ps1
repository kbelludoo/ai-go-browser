$files = Get-ChildItem -Path "c:\Users\kbell\Downloads" -Recurse -Include *.bat, *.ps1, *.sh, *.json, *.txt, *.md -ErrorAction SilentlyContinue
foreach ($f in $files) {
    if ($f.FullName -like "*node_modules*" -or $f.FullName -like "*\.git\*") { continue }
    $content = Get-Content $f.FullName -Raw -ErrorAction SilentlyContinue
    if ($content) {
        $ips = [regex]::Matches($content, '\b(?:1\d\d|2[0-4]\d|25[0-5]|[1-9]\d|\d)\.(?:1\d\d|2[0-4]\d|25[0-5]|[1-9]\d|\d)\.(?:1\d\d|2[0-4]\d|25[0-5]|[1-9]\d|\d)\.(?:1\d\d|2[0-4]\d|25[0-5]|[1-9]\d|\d)\b')
        foreach ($ip in $ips) {
            $val = $ip.Value
            if ($val -ne "127.0.0.1" -and $val -ne "0.0.0.0" -and !$val.StartsWith("255.")) {
                Write-Host "$($f.FullName) => $val"
            }
        }
    }
}
