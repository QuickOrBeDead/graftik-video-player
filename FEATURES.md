# Features

## Video Playback
Core video player built on the HTML5 `<video>` element with a custom dark-themed controls overlay. Supports common video formats via system codecs and unsupported formats via HLS transcoding.

- Play, pause, and seek through video files (mp4, webm, ogg, mov, 3gp)
- Progress bar with hover thumbnail preview and time tooltip
- Keyboard shortcuts (Space, Left/Right arrows, F, M, I, S, R, N, P, T, +, -)
- Playback speed control (0.5x, 1x, 1.25x, 1.5x, 2x)
- Volume control with mute toggle and slider
- Skip forward/backward by 10 seconds
- Picture-in-Picture mode
- Fullscreen mode
- Auto-play next video on end
- Repeat modes: Off, All, One
- Shuffle mode with auto-generated deck

### HLS Transcoding for Unsupported Formats
When a file is not natively playable (MKV, HEVC, AV1, AVI, FLV), the Go backend runs FFprobe to classify the file and then either remuxes (`-c copy`, near-zero CPU) or transcodes (software or hardware-accelerated) to HLS segments served by a local HTTP server.

```
User opens MKV file
        │
        ▼
Go backend runs ffprobe → classifies as native/remux/sw_transcode/hw_transcode
        │
        ▼
FFmpeg produces .ts segments + .m3u8 playlist
        │
        ▼
Local Go HTTP server serves segments on http://localhost:<port>/
        │
        ▼
Frontend plays via hls.js <video> src = http://localhost:<port>/stream.m3u8
```

| Input Container | Input Codec | Action |
|---------------|-------------|--------|
| MKV | H.264 + AAC | Remux (`-c copy`) |
| MKV | H.265/HEVC | Transcode to H.264 |
| MKV | AV1 | Transcode to H.264 |
| AVI | MPEG-4 + MP3 | Transcode to H.264 + AAC |
| FLV | H.264 + AAC | Remux (`-c copy`) |

Hardware acceleration auto-detection: NVIDIA NVENC, Intel QSV, AMD AMF.

## Playlist Management
Organize videos into named playlists stored in a local SQLite database. Playlists track per-video progress and remember the last-played item.

- Create, rename, and delete playlists
- Add videos via File > Add Video (multi-select system file dialog)
- Switch between playlists via File > Choose Playlist
- Auto-creates a "default" playlist on first launch
- Right-click context menu: Play/Pause, Remove from Playlist, Open Containing Folder

## Playlist View
Sidebar panel with multiple view modes, sorting, filtering, and progress visualization. Resizable via drag handle with a minimum/maximum width constraint.

- Detailed and Simple view modes
- Drag-and-drop reordering of playlist items (via vuedraggable)
- Filter: "Unwatched" toggle (progress < 5%)
- Sort: Default, Name (A-Z/Z-A), Length (Shortest/Longest), Recently/Oldest Watched
- Total playlist duration display
- Per-video progress bar and watch time tracking
- Resizable sidebar (drag handle, 230–600px range)

## Thumbnails
Thumbnails are extracted from video files using FFmpeg at 10% seek position and cached to disk. Generated on-demand when a playlist loads, limited to 2 concurrent extractions.

- Automatic thumbnail generation using FFmpeg (seeks to 10% of video)
- Thumbnails cached per playlist/folder on disk (JPEG, 180px wide)
- File hash-based invalidation (detects file changes via path + size + mtime)

## Persistence
Playlists and playback state are stored in a local SQLite database using `modernc.org/sqlite` (pure Go, no CGO). App preferences (volume, playback rate, sidebar width, window geometry, shuffle/repeat mode, last played item) use a JSON config file.

- Local SQLite database (via `modernc.org/sqlite`)
- Playlist state persisted: current item, playback position, progress %, volume, shuffle/repeat mode
- Periodic save of playback state every 10 seconds (Go ticker)
- State saved on window close
- Preferences saved with 500ms debounce via frontend watchers
- WAL mode for database performance

## Plugin System (Lua)
Extend the player with Lua 5.1 scripts. Plugins are discovered from `~/.config/graftik-video-player/plugins/<id>/` and can register menu entries, actions, and custom UI.

- Plugin structure: `plugin.json` + `main.lua`
- Host API: `host.exec()`, `host.emit()`, `host.addToPlaylist()`
- Install from ZIP (URL or local file picker)
- Custom plugin UI (HTML/JS loaded in modal)
- Plugin management panel with status and action buttons

## Auto-Update
Checks GitHub Releases API on startup (ETag-based caching). Downloads and installs updates with a progress bar. Supports semver comparison.

- Checks GitHub API on startup with ETag caching
- Semver comparison via `github.com/Masterminds/semver/v3`
- Download with progress bar
- Linux install via `pkexec dpkg -i`, Windows via silent installer
- Update dialog with release notes display

## User Interface
Frameless dark-themed UI built with Bootstrap 5 and styled with CSS custom properties. Application menu, modals, and context menus provide standard desktop interactions.

- Custom frameless window (#0f0f0f background)
- Dark theme with accent blue highlights
- Bootstrap 5 + Bootstrap Icons for UI components
- Custom context menus (@imengyu/vue3-context-menu)
- Playlist modals for delete confirmation and rename
- Application menu: File (Add Video, New Playlist, Choose Playlist, Plugins, Exit), Help (Check for Updates, About)

## Architecture
Wails v2 application with Go backend and Vue 3 + TypeScript frontend. The Go binary embeds the built frontend assets and serves them via WebView2. No Electron or Node.js runtime is required.

- Go backend (single static binary, ~15 MB)
- Vue 3 + TypeScript + Vite frontend (served via `embed.FS`)
- Wails v2 bindings for Go ↔ frontend communication (no IPC boilerplate)
- Cross-platform: Windows amd64, Linux amd64
- FFmpeg/FFprobe bundled alongside the binary for media processing
- CI/CD: GitHub Actions — typecheck + `go vet` on push, tag-triggered builds with `.deb` and `.exe` output

## Planned

### Playlist Search / Filter by Title
Text search input in the playlist header to filter videos by title, complementing the existing "Unwatched" toggle.

### OSD Notifications
Brief on-screen overlays when toggling volume, mute, shuffle, repeat, playback speed — providing visual feedback for actions taken via buttons or keyboard shortcuts.
