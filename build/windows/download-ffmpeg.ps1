# Download ffmpeg/ffprobe 8.1.2 for Windows from GitHub Releases
param()

$owner = "QuickOrBeDead"
$repo = "graftik-video-player"
$tag = "ffmpeg-8.1.2"
$archive = "windows-amd64-${tag}.zip"
$expectedHash = "4a6340ba94659ad9a8397c514e7e5cfad31880df4c685433ec5463ff5f1e6d90"
$baseUrl = "https://github.com/${owner}/${repo}/releases/download/${tag}"

$outDir = Join-Path $PSScriptRoot "bin"
if (!(Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir -Force }

$marker = Join-Path $env:TEMP "ffmpeg-8.1.2-windows.verified"
$ffmpegPath = Join-Path $outDir "ffmpeg.exe"
if ((Test-Path $marker) -and (Test-Path $ffmpegPath)) {
    Write-Host "ffmpeg already present and verified at ${outDir}"
    return
}

$tmpFile = Join-Path $env:TEMP $archive

Write-Host "Downloading ${archive} ..."
Invoke-WebRequest -Uri "${baseUrl}/${archive}" -OutFile $tmpFile

Write-Host "Verifying integrity ..."
$hash = (Get-FileHash $tmpFile -Algorithm SHA256).Hash.ToLower()
if ($hash -ne $expectedHash) {
    Write-Error "SHA-256 mismatch: expected ${expectedHash}, got ${hash}"
    Remove-Item $tmpFile -Force
    exit 1
}

Write-Host "Extracting ffmpeg.exe and ffprobe.exe ..."
$tempDir = Join-Path $env:TEMP "ffmpeg_extract"
if (Test-Path $tempDir) { Remove-Item $tempDir -Recurse -Force }
Expand-Archive -Path $tmpFile -DestinationPath $tempDir

Copy-Item (Join-Path $tempDir "ffmpeg.exe") (Join-Path $outDir "ffmpeg.exe") -Force
Copy-Item (Join-Path $tempDir "ffprobe.exe") (Join-Path $outDir "ffprobe.exe") -Force

Remove-Item $tmpFile -Force
Remove-Item $tempDir -Recurse -Force
Set-Content -Path $marker -Value "verified" -Force

Write-Host "ffmpeg/ffprobe downloaded to: $outDir"
