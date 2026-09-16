# VideoDownloaderUltra installer - Windows PowerShell
$Repo = "Saimonsanbr/VideoDownloaderUltra"
$Bin = "videodownloaderultra.exe"
$InstallDir = "$env:USERPROFILE\bin"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

Write-Host "Detectando último release..."
$LatestUrl = "https://github.com/$Repo/releases/latest"
try {
    $resp = Invoke-WebRequest -Uri $LatestUrl -UseBasicParsing -MaximumRedirection 0 -ErrorAction SilentlyContinue
    $tag = $resp.Headers.Location -replace ".*/tag/",""
} catch {
    $tag = $_.Exception.Response.Headers.Location -replace ".*/tag/",""
}
if (-not $tag) { $tag = "latest" }
Write-Host "Tag: $tag"

$File = "videodownloaderultra-windows-amd64.exe"
$Url = "https://github.com/$Repo/releases/download/$tag/$File"
if ($tag -eq "latest") { $Url = "https://github.com/$Repo/releases/latest/download/$File" }

$Tmp = Join-Path $env:TEMP "vdu_$([Guid]::NewGuid().ToString().Substring(0,8)).exe"
Write-Host "Baixando $Url ..."
Invoke-WebRequest -Uri $Url -OutFile $Tmp -UseBasicParsing

$Dest = Join-Path $InstallDir $Bin
Copy-Item $Tmp $Dest -Force
Remove-Item $Tmp -Force

# Adiciona ao PATH do usuário
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path += ";$InstallDir"
    Write-Host "Adicionado $InstallDir ao PATH (reinicie o terminal)"
}

# Atalho no Menu Iniciar
$StartMenu = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs"
$WshShell = New-Object -ComObject WScript.Shell
$Shortcut = $WshShell.CreateShortcut("$StartMenu\VideoDownloaderUltra.lnk")
$Shortcut.TargetPath = $Dest
$Shortcut.Arguments = "--gui"
$Shortcut.WorkingDirectory = $InstallDir
$Shortcut.Description = "VideoDownloaderUltra - Downloader teste API terceiros"
$Shortcut.Save()

Write-Host ""
Write-Host "✓ Instalado: $Dest" -ForegroundColor Green
Write-Host "Uso CLI: videodownloaderultra https://youtu.be/..."
Write-Host "Uso GUI: videodownloaderultra --gui  ou pesquise VideoDownloaderUltra no Menu Iniciar"
