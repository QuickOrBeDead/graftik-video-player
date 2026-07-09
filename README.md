# Graftik Video Player

A cross-platform desktop video player built with [Wails v2](https://wails.io/) (Go backend + WebView2), Vue 3, and TypeScript. Plays native formats (MP4, WebM, Ogg) and transcodes unsupported formats (MKV, HEVC, AV1, AVI, FLV) to HLS via FFmpeg.

## Features

- HTML5 `<video>` with custom dark overlay controls
- HLS transcoding via FFmpeg for unsupported formats (remux or transcode)
- Thumbnail preview on progress bar hover
- Playlist management with SQLite persistence
- Plugins system (Lua 5.1)
- Auto-update via GitHub Releases

## Prerequisites

- Go 1.26+
- Node.js 24+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)
- Linux: `libgtk-3-dev`, `libwebkit2gtk-4.1-dev`
- Windows: [WebView2](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) (preinstalled on Win10+)

## Development

```bash
wails dev
```

This runs the Vite dev server with hot-reload for the Vue frontend and rebuilds the Go backend on changes.

## Build

```bash
# Linux
wails build -platform linux/amd64 -clean

# Windows
wails build -platform windows/amd64 -clean -nsis
```

The version is injected at build time from the git tag via `-ldflags "-X main.appVersion=<tag>"`.

## Project Structure

```
.
├── main.go              # Wails app entry point
├── app.go               # App struct, menu, startup/shutdown, update logic
├── version.go           # Version variable, update checker/downloader
├── videoserver.go       # Local HTTP server for HLS segments
├── wails.json           # Wails project config
├── internal/
│   ├── data/            # SQLite store (modernc.org/sqlite), thumbnail cache, config
│   ├── hls/             # FFmpeg HLS engine
│   ├── media/           # FFprobe probe + codec classification
│   ├── plugin/          # Lua plugin manager
│   ├── logger/          # Structured logger
│   └── player_service.go  # Wails-bound service methods
├── frontend/
│   ├── src/             # Vue 3 + TypeScript app
│   └── package.json
└── build/               # Build scripts, FFmpeg bundling, nfpm config
```
