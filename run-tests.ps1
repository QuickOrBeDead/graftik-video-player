param(
    [switch]$Short
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

if ($Short) {
    Write-Host "=== Running Go tests (short mode) ==="
    & "go" @("test", "-short", "./...")
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    return
}

Write-Host "=== Downloading ffmpeg ==="
& .\build\windows\download-ffmpeg.ps1

Write-Host "`n=== Running Go tests ==="
& "go" @("test", "./...")
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
