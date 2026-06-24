# Features

## Video Playback
Core video player built on the HTML5 `<video>` element with a custom dark-themed controls overlay. Supports common video formats via system codecs.

- Play, pause, and seek through video files (mp4, mov, ogg, webm, 3gp)
- Progress bar with hover thumbnail preview and time tooltip
- Keyboard shortcuts (Space, Left/Right arrows, F, M, P, S, R, N)
- Playback speed control (0.5x, 1x, 1.25x, 1.5x, 2x)
- Volume control with mute toggle and slider
- Skip forward/backward by 10 seconds
- Picture-in-Picture mode
- Fullscreen mode
- Auto-play next video on end

## Playlist Management
Organize videos into named playlists stored in a local SQLite database. Playlists track per-video progress and remember the last-played item.

- Create, rename, and delete playlists
- Add videos via File > Add Video (supports multi-select)
- Switch between playlists via File > Choose Playlist
- Auto-creates a "default" playlist on first launch
- Right-click context menu: Play/Pause, Remove from Playlist, Open Containing Folder

## Playlist View
Sidebar panel with multiple view modes, sorting, filtering, and progress visualization. Resizable via drag handle with a minimum/maximum width constraint.

- Detailed and Simple view modes
- Drag-and-drop reordering of playlist items (via vuedraggable)
- Filter: "Unwatched" toggle
- Sort: Default, Name (A-Z/Z-A), Length (Shortest/Longest), Recently/Oldest Watched
- Total playlist duration display
- Per-video progress bar and watch time tracking
- Resizable sidebar (drag the resize handle)

## Thumbnails
Thumbnails are extracted from video files using FFmpeg at 10% seek position and cached to disk. Generated on-demand when a playlist loads, limited to 2 concurrent extractions.

- Automatic thumbnail generation using FFmpeg (seeks to 10% of video)
- Thumbnails cached per playlist/folder on disk
- File hash-based invalidation (detects file changes via path + size + mtime)

## Persistence
Playlists and playback state are stored in a local SQLite database using better-sqlite3 with Drizzle ORM. App preferences (current playlist selection) use electron-store. Database runs in WAL mode for read performance.

- Local SQLite database (via better-sqlite3 + Drizzle ORM)
- Playlist state persisted: current item, playback position, progress %, volume, shuffle/repeat mode
- Periodic save of playback state every 10 seconds
- State saved on window close
- WAL mode for database performance

## User Interface
Frameless dark-themed UI built with Bootstrap 5 and styled with CSS custom properties. Subwindows use a custom title bar with drag regions. Context menus and modals provide standard desktop interactions.

- Custom frameless window with custom title bar on subwindows
- Dark theme with accent blue highlights
- Bootstrap 5 + Bootstrap Icons for UI components
- Custom context menus (@imengyu/vue3-context-menu)
- Playlist modals for delete confirmation and rename

## Architecture
Electron app with strict process separation (main/renderer/preload) and context-isolated IPC. Built with electron-vite toolchain, Vue 3 + TypeScript on the frontend, and Drizzle ORM for database access.

- Electron main/renderer/preload process separation
- Context-isolated IPC communication
- Vue 3 + TypeScript + Vite frontend
- vue-router for subwindow routing (/, /playlists, /add-playlist)
- Cross-platform support (Windows, macOS, Linux builds via electron-builder)
- Auto-update support (electron-updater configured)

## Planned

### Keyboard Shortcuts
Mousetrap is a dependency but not yet wired. Planned bindings: Space (play/pause), Left/Right (skip 10s), F (fullscreen), M (mute), P (picture-in-picture), S (shuffle), R (repeat mode), N (next), Shift+N (previous).

### Playlist Search / Filter by Title
Text search input in the playlist header to filter videos by title, complementing the existing "Unwatched" toggle.

### OSD Notifications
Brief on-screen overlays when toggling volume, mute, shuffle, repeat, playback speed — providing visual feedback for actions taken via buttons or keyboard shortcuts.

### Session Restore
Automatically resume playback from the last position when reopening the app. Playback state (currentTime, isPlaying, currentItem) is already persisted every 10 seconds; the restore logic needs to be wired on startup.

### Auto-Updater
Wire up electron-updater to check for updates on startup, download in the background, and prompt the user to install. The electron-builder publish config is already in place (requires a real update server URL).

---

## Roadmap

### Build Size Optimization (Electron Quick Wins)
The current Windows build weighs ~884MB unpacked / ~200MB NSIS installer. Analysis of `dist/win-unpacked`:

| Component | Size | Issue |
|-----------|------|-------|
| Electron/Chromium (electron.exe + DLLs + PAK + locales) | ~339MB | Fixed — Chromium is inherent to Electron |
| ffprobe-static (6 platform binaries) | ~335MB | **~275MB wasted** — ships darwin/linux/win32-ia32 binaries on Windows |
| ffmpeg-static | ~79MB | Single Windows binary (~42MB) + install scripts |
| better-sqlite3 build artifacts | ~39MB | Compiled `.node` addon + full `build/` dir with `.obj`, `.tlog`, test extensions |
| App code + dependencies | ~92MB | Drizzle ORM, Bootstrap, Vue, etc. |

**Immediate fixes** (no architecture change):
- Switch `ffprobe-static` to `@ffprobe-installer/ffprobe` (platform-aware, single binary)
- Add `asarUnpack` filters for `better-sqlite3` to exclude build intermediates
- Remove unused SQLite test extensions
- **Estimated savings: ~300–350MB unpacked, ~60–80MB installer**

---

### Wails v2 Migration Plan

#### Electron vs Wails Output Size Comparison

| Metric | Electron (current) | Wails v2 (target) | Savings |
|--------|-------------------|--------------------|---------|
| Unpacked build | ~884 MB | ~15–25 MB | ~97% smaller |
| Installer (NSIS) | ~200 MB | ~15–25 MB | ~90% smaller |
| Runtime | Node.js + Chromium (~213 MB) | Go binary + WebView2 (system) | Eliminates Chromium |
| Memory usage (idle) | ~150–250 MB | ~30–60 MB | ~75% less |

Why the difference:

| Component | Electron | Wails |
|-----------|----------|-------|
| Browser engine | Bundled Chromium (~213 MB) | System WebView2 (already on Win10/11, ~500 KB bootstrapper) |
| Runtime | Node.js (~30 MB) | Go static binary (~10–15 MB) |
| Native modules | better-sqlite3 C++ addon (~39 MB with build artifacts) | Pure Go SQLite (`modernc.org/sqlite`, ~5 MB) |
| Cross-platform binaries | ffprobe for all 6 platforms bundled (~335 MB) | Single Win32 ffprobe (~60 MB, only if needed) |
| Installer overhead | NSIS + Electron meta (~50 MB) | Raw binary + optional NSIS wrapper (~5 MB) |

#### Migration Phases

**Phase 1 — Scaffold & Data Layer (Week 1)**
- `wails init -t vue` — scaffold Go + Vue 3 + Vite
- Replace better-sqlite3 + Drizzle ORM with `modernc.org/sqlite` (pure Go, no CGO)
- Port `PlayerRepository` (CRUD for playlists/items)
- Port `AppState` (current playlist, preferences via Go config)
- Port `ThumbnailDataStore` (disk cache with SHA256 hashing)

**Phase 2 — Backend API Bindings (Week 1–2)**
- Create `PlayerService` struct with exported Go methods that map 1:1 to current IPC handlers:
  - Playlist CRUD: `GetPlaylists`, `AddPlaylist`, `UpdatePlaylistName`, `DeletePlaylist`, `SelectPlaylist`
  - Playlist items: `AddPlaylistItems`, `UpdatePlaylistItem`, `DeletePlaylistItem`, `RebalancePlaylistOrder`
  - Metadata: `GetPlaylistItemVideoMetadata` (ffprobe probe + thumbnail extraction via ffmpeg exec)
  - Folder: `OpenContainingFolder` (exec `explorer /select,"path"`)
- Port file dialogs: `runtime.OpenFileDialog()` with video format filters
- Port app menu: Wails `menu.Menu()` with File > Add Video, New Playlist, Choose Playlist, Exit

**Phase 3 — Window & Event Management (Week 2)**
- Port subwindow creation: Wails `application.NewWindow()` for Playlists, NewPlaylist modals
- Port frameless window: Wails frameless config + custom title bar (frontend unchanged)
- Replace `mainWindow.webContents.send()` with `Events.Emit()` / `Events.On()`
- Implement periodic state save via Go ticker (replaces 10-second JS interval)

**Phase 4 — Frontend Adaptation (Week 2–3)**
- Replace all `window.electron.ipcRenderer.invoke()` calls with `window.runtime.Call()`
- Replace `window.electron.ipcRenderer.on()` with `EventsOn()` / `EventsOff()`
- Remove preload scripts — Wails handles context isolation natively
- Keep all Vue components, composables, styles exactly as-is

**Phase 5 — Polish & Distribution (Week 3)**
- Wire auto-updater: `github.com/rhysd/go-github-update` or Wails' built-in mechanism
- Configure `wails build -platform windows/amd64 -nsis` for installer
- Test all features: playback, playlists, thumbnails, drag-drop, context menus
- Bundle ffprobe as optional extra resource for metadata extraction

---

### FFmpeg Demuxing for Unsupported Formats (MKV, AVI, HEVC, AV1)

#### Problem
The HTML5 `<video>` element natively supports only: MP4 (H.264/AAC), WebM (VP8/VP9), OGG (Theora/Vorbis). MKV, AVI, FLV, MOV with exotic codecs, HEVC (x265), and AV1 are not playable.

#### Solution: HLS Transcoding + Local HTTP Server

**Architecture:**
```
User opens MKV file
        │
        ▼
Go backend runs ffprobe → detects unsupported format/codec
        │
        ▼
FFmpeg transcodes to HLS segments (m3u8 + .ts chunks)
        │
        ▼
Local Go HTTP server serves segments on http://localhost:<port>/
        │
        ▼
Frontend plays via hls.js <video> src = http://localhost:<port>/stream.m3u8
```

**Smart Transcoding Strategy:**

| Input Container | Input Codec | Action | Cost |
|---------------|-------------|--------|------|
| MKV | H.264 + AAC | Remux (`-c copy`) | Near-zero CPU |
| MKV | H.265/HEVC | Transcode to H.264 | Medium CPU |
| MKV | AV1 | Transcode to H.264 | High CPU |
| AVI | MPEG-4 + MP3 | Transcode to H.264 + AAC | Medium CPU |
| FLV | H.264 + AAC | Remux (`-c copy`) | Near-zero CPU |

**Hardware acceleration** (auto-detected):
- NVIDIA: `h264_nvenc`
- Intel QSV: `h264_qsv`
- AMD: `h264_amf`

**Go Backend API:**
```go
func (a *App) GetStreamURL(playlistItemID string) string {
    item := a.store.GetItem(playlistItemID)
    if isPlayable(item.Path) {
        return item.Path  // native, direct file:// URL
    }
    return a.hlsService.StartStream(item.Path)  // http://localhost:PORT/stream.m3u8
}
```

**Implementation Steps:**
1. Add `ffprobe` format/codec detection helper in Go
2. Implement `HLSService` — starts ffmpeg to produce segments to a temp dir
3. Implement local HTTP server (Go `net/http`) to serve segments
4. Add hls.js to the Vue frontend
5. Cleanup: remove temp segments when playlist changes or app closes
6. Optionally cache transcoded segments (hash of input file) for repeat viewing
