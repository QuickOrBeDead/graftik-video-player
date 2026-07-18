# Test Matrix — Container/Codec Combinations

Based on the classification logic in `src/internal/media/probe.go:136-190`.

## Native (Direct Playback)

Played directly by the HTML5 `<video>` element — no processing.

| Container | Video Codec | Audio Codec | Download Source |
|-----------|-------------|-------------|----------------|
| MP4/MOV | H.264 | AAC | [test-videos.co.uk](https://test-videos.co.uk/) — Big Buck Bunny H.264/MP4 |
| MP4/MOV | H.264 | MP3 | [samplelib.com](https://samplelib.com/sample-mp4.html) or generate with FFmpeg |
| MP4/MOV | HEVC/H.265 | AAC | [samplelib.com](https://samplelib.com/sample-mp4.html) — "H.265/HEVC (hvc1, 720p)" |
| MP4/MOV | MPEG-4 | AAC | [filesamples.com](https://filesamples.com/formats/mp4) — MP4 sample |
| MP4/MOV | VP9 | Opus | [samplelib.com](https://samplelib.com/sample-mp4.html) — "VP9 inside MP4 (720p)" |
| MP4/MOV | VP8 | Vorbis | [blender.org](https://peach.blender.org/download/) — Big Buck Bunny MP4 |
| WebM | VP8/VP9 | Vorbis/Opus | [test-videos.co.uk](https://test-videos.co.uk/) — Big Buck Bunny VP9/WebM |
| OGG | Theora | Vorbis | [file-examples.com](https://file-examples.com/index.php/sample-video-files/sample-ogg-files-download/) |
| 3GP/3G2 | H.263/H.264 | AAC/AMR | [filesamples.com](https://filesamples.com/formats/3gp) |

## Remux (`-c copy`)

Stream-copied into HLS segments — no re-encode, should be very fast.

| Container | Video Codec | Audio Codec | Download Source |
|-----------|-------------|-------------|----------------|
| MKV | H.264 | AAC | [test-videos.co.uk](https://test-videos.co.uk/) — Big Buck Bunny H.264/MKV |
| MKV | H.264 | MP3 | [filesamples.com](https://filesamples.com/formats/mkv) — MKV sample |
| MKV | H.264 | FLAC | Generate with FFmpeg: `ffmpeg -i input -c:v copy -c:a flac test.mkv` |
| AVI | H.264 | AAC/MP3 | [filesamples.com](https://filesamples.com/formats/avi) or [file-examples.com](https://file-examples.com/index.php/sample-video-files/sample-avi-files-download/) |
| FLV | H.264 | AAC/MP3 | [filesamples.com](https://filesamples.com/formats/flv) |
| TS/MTS/M2TS | H.264 | AAC/AC3 | [filesamples.com](https://filesamples.com/formats/ts) / [MTS](https://filesamples.com/formats/mts) |
| VOB | H.264/MPEG-2 | AC3 | [filesamples.com](https://filesamples.com/formats/vob) |

## Transcode (SW / HW)

Re-encoded to H.264 + AAC via libx264 (SW) or NVENC/QSV/AMF (HW).

| Container | Video Codec | Audio Codec | Expected Action | Download Source |
|-----------|-------------|-------------|-----------------|----------------|
| MKV | HEVC/H.265 | AAC/FLAC/Opus/DTS | SW/HW Transcode | [hitokageproduction.com](https://hitokageproduction.com/article/83) — 4K H.265 (CC0); or [filesamples.com](https://filesamples.com/formats/hevc) |
| MKV | AV1 | Opus | SW/HW Transcode | [hitokageproduction.com](https://hitokageproduction.com/article/83) — 4K AV1 (CC0) |
| MKV | VP9 | Opus/Vorbis | SW/HW Transcode | [hitokageproduction.com](https://hitokageproduction.com/article/83) — 4K VP9 (CC0) |
| AVI | MPEG-4 | MP3 | SW/HW Transcode | [filesamples.com](https://filesamples.com/formats/avi) |
| AVI | HEVC | AAC | SW/HW Transcode | Generate: `ffmpeg -i input -c:v libx265 test.avi` |
| FLV | H.263 / Sorenson Spark | MP3 | SW/HW Transcode | [filesamples.com](https://filesamples.com/formats/flv) |
| FLV | H.264 | AAC | Remux | [filesamples.com](https://filesamples.com/formats/flv) |
| MP4 | AV1 | Opus | SW/HW Transcode | [test-videos.co.uk](https://test-videos.co.uk/) — Big Buck Bunny AV1/MP4 |
| MP4 | ProRes | PCM | SW/HW Transcode | Generate with FFmpeg |
| MOV | ProRes | PCM | SW/HW Transcode | [blender.org](https://peach.blender.org/download/) — Big Buck Bunny MOV |
| MOV | DNxHD | PCM | SW/HW Transcode | Generate with FFmpeg |
| WMV | WMV3/WVC1 | WMA | SW/HW Transcode | [filesamples.com](https://filesamples.com/formats/wmv) or [file-examples.com](https://file-examples.com/index.php/sample-video-files/sample-wmv-files-download/) |
| RM/RMVB | RealVideo | RealAudio | SW/HW Transcode | [filesamples.com](https://filesamples.com/formats/rm) |

## Edge Cases

| Scenario | Expected Behaviour |
|----------|--------------------|
| MP4 with H.264 + AC3 audio | Native container → plays directly (codec display falls back to raw "AC3") |
| MKV with H.264 + multiple audio tracks (e.g. AC3 + AAC) | Remux — first audio track processed |
| MKV with H.264 and no audio | Remux (video-only stream-copy) |
| MKV with no video stream (audio-only) | Classification may misbehave; should handle gracefully |
| TS with MPEG-2 video | Native codec in non-native container → Remux |
| VOB with MPEG-2 video | Remux |
| Large files (>4GB) | Ensure FFmpeg/HLS handling works |
| Files with non-ASCII filenames | Ensure paths are handled correctly |
| Corrupted / truncated files | FFmpeg error should be surfaced gracefully |

## HW Encoder Detection

| Encoder | FFmpeg Name | Expected On |
|---------|-------------|-------------|
| NVIDIA NVENC | `h264_nvenc` | NVIDIA GPU present |
| Intel QSV | `h264_qsv` | Intel iGPU present |
| AMD AMF | `h264_amf` | AMD GPU present |

Behaviour when no HW encoder is available: falls back to `libx264` software encoding.

## Native File Dialog Extensions

From `src/app.go:534` (18 extensions):

```
*.mp4 *.mov *.ogg *.webm *.3gp
*.mkv *.avi *.flv *.ts  *.mts *.m2ts
*.wmv *.rm  *.rmvb *.vob  *.mpg *.mpeg *.m4v
```

## Natively-Playable Extensions

From `src/probe.go:212-219` / `Player.vue:43` (8 extensions):

```
.mp4 .mov .m4v
.3gp .3g2
.webm
.ogg .ogv
```

## Remux Candidates (non-native container, native codec)

Files matching this pattern bypass re-encode entirely:

| Extension | Typical Containers |
|-----------|-------------------|
| `.mkv` | Matroska with H.264/AAC |
| `.avi` | AVI with H.264 |
| `.flv` | FLV with H.264/AAC |
| `.ts`/`.mts`/`.m2ts` | MPEG-TS with H.264 |
| `.vob` | VOB with MPEG-2 / H.264 |
| `.wmv` | ASF with H.264 |
