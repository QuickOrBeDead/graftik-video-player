# Download minimal ffmpeg/ffprobe for Windows from gyan.dev
$baseUrl = "https://www.gyan.dev/ffmpeg/builds"
$archive = "ffmpeg-release-essentials.7z"
$outDir = Join-Path $PSScriptRoot "bin"

if (!(Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir -Force }

Write-Host "Downloading $archive ..."
Invoke-WebRequest -Uri "$baseUrl/$archive" -OutFile "$env:TEMP\$archive"

Write-Host "Extracting ffmpeg.exe and ffprobe.exe ..."
& 7z x "$env:TEMP\$archive" -o"$env:TEMP\ffmpeg" -y "ffmpeg-*-essentials_build\bin\ffmpeg.exe" "ffmpeg-*-essentials_build\bin\ffprobe.exe" | Out-Null

$extracted = Get-ChildItem "$env:TEMP\ffmpeg\ffmpeg-*-essentials_build\bin" -Recurse
foreach ($file in $extracted) {
    Copy-Item $file.FullName (Join-Path $outDir $file.Name) -Force
}

Remove-Item "$env:TEMP\$archive" -Force
Remove-Item "$env:TEMP\ffmpeg" -Recurse -Force

Write-Host "ffmpeg/ffprobe downloaded to: $outDir"
