param(
    [switch]$Short
)

$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot ".." "src")

$exclude = "github.com/QuickOrBeDead/graftik-video-player/frontend"

if ($Short) {
    Write-Host "=== Running Go tests (short mode) ==="
    $pkgs = go list ./... | Where-Object { $_ -notmatch $exclude }
    & "go" @("test", "-short") + $pkgs
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    return
}

Write-Host "=== Downloading ffmpeg ==="
& .\build\windows\download-ffmpeg.ps1

Write-Host "`n=== Running Go tests ==="
$pkgs = go list ./... | Where-Object { $_ -notmatch $exclude }
& "go" @("test") + $pkgs
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
