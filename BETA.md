# Graftik Video Player — Beta Release Plan

**Current version:** `0.0.0`  
**Target first beta:** `v0.1.0-beta.1`  
**Target stable:** `v0.1.0`

---

## Current Features

### Video Playback Core
- HTML5 `<video>` with custom dark overlay controls
- Play/Pause (click or toolbar button)
- Seek via progress bar (click/drag) with thumbnail preview + time tooltip
- Skip forward/backward by 10 seconds
- Playback speed control (0.5x, 1x, 1.25x, 1.5x, 2x)
- Volume control with mute toggle and slider
- Picture-in-Picture mode
- Fullscreen mode
- Auto-play next video on end
- Repeat modes: Off, All, One
- Shuffle mode with auto-generated deck

### HLS Transcoding for Unsupported Formats
- FFprobe-based media detection — classifies as `native`, `remux`, `sw_transcode`, or `hw_transcode`
- Automatic remuxing (MKV/H.264+AAC → HLS with `-c copy`, near-zero CPU)
- Software transcoding (HEVC, AV1, AVI, FLV → H.264)
- Hardware acceleration auto-detection: NVIDIA NVENC, Intel QSV, AMD AMF
- Local HTTP video server with range request support
- FFmpeg-based HLS engine produces .ts segments + .m3u8 playlist
- hls.js for HLS stream playback
- Cleanup of stream files on shutdown

### Thumbnail Generation
- FFmpeg-based extraction at 10% seek position
- Resized to 180px wide, cached as JPEG to disk
- File hash-based cache invalidation (path + size + mtime)
- Concurrency-limited to 2 parallel extractions (via p-limit)
- Cache organized per playlist folder

### Playlist Management
- SQLite-backed storage with WAL mode (playlists + playlist_items tables)
- Create, rename, and delete playlists
- Add videos via File > Add Video (multi-select system file dialog)
- Auto-creates "default" playlist on first launch
- Per-video progress tracking (elapsed time, duration, progress %, last watched)
- Drag-and-drop reordering via vuedraggable
- Detailed and Simple view modes
- Sort: Default, Name (A-Z/Z-A), Length (Shortest/Longest), Recently/Oldest Watched
- Filter: "Unwatched" toggle (progress < 5%)
- Total playlist duration display
- Resizable sidebar (drag handle, 230–600px range)
- Right-click context menu: Play/Pause, Remove, Open Containing Folder

### State Persistence
- SQLite database + JSON config file
- Saved preferences: volume, playback rate, sidebar visibility/width, window size, shuffle/repeat mode, last played item
- Periodic save every 10 seconds (Go ticker)
- Window size saved on shutdown
- Preferences saved with 500ms debounce via frontend watchers

### Plugin System (Lua)
- Plugin discovery from `~/.config/graftik-video-player/plugins/<id>/`
- Lua 5.1 runtime via gopher-lua
- Plugin structure: `plugin.json` + `main.lua` (returns table with `menuEntries`, `actions`, `ui`)
- Host API exposed to Lua: `host.exec()`, `host.emit()`, `host.addToPlaylist()`
- Install from ZIP (URL or local file picker)
- Custom plugin UI (HTML/JS loaded in modal)
- Plugin management panel with status and action buttons

### Auto-Update System
- Checks GitHub releases API on startup (ETag-based caching)
- Semver comparison
- Downloads update binary (.deb for Linux, .exe for Windows) with progress bar
- Linux install via `pkexec dpkg -i`, Windows via silent installer
- Update dialog with release notes display

### User Interface & UX
- Frameless dark-themed window (#0f0f0f)
- Bootstrap 5 + Bootstrap Icons
- Custom dark scrollbar, hover effects, accent blue highlights
- Progress bar with hover thumbnail preview and time tooltip
- Modal dialogs for playlists, plugins, updates, delete confirmations
- Application menu: File (Add Video, New Playlist, Choose Playlist, Plugins, Exit), Help (Check for Updates, About)

### CI/CD Pipeline
- GitHub Actions workflows:
  - **CI** — typecheck + frontend build + go vet on push/PR to main
  - **Build** — tag-triggered (v\*), produces Windows .exe + Linux .deb
  - **Release** — manual workflow to create and push tags
- Cross-platform: Windows amd64, Linux amd64
- Bundled ffmpeg/ffprobe binaries (Windows in `bundled/`)

---

## Beta Release Requirements

### Must-Have (for v0.1.0-beta.1)

#### 1. Keyboard Shortcuts
Mousetrap `^1.6.5` is already a dependency but **not wired**. The toolbar tooltips reference keyboard shortcuts that don't work.

**Required bindings:**
| Key | Action |
|---|---|
| Space | Play/Pause |
| Left Arrow | Skip backward 10s |
| Right Arrow | Skip forward 10s |
| F | Fullscreen toggle |
| M | Mute toggle |
| P | Picture-in-Picture toggle |
| S | Shuffle toggle |
| R | Repeat mode cycle |
| N | Next video |
| Shift+N | Previous video |

**Implementation:** Add a `keydown` event listener in `Player.vue` (or `usePlayer` composable) that maps keys to existing composable actions.

#### 2. Session Restore (Playback Position)
The app saves `elapsedTime` per item and `lastPlayedItem` in preferences, but **does not restore playback position on startup**. `shouldAutoplay` is already wired.

**Implementation:** In `Main.vue` after `loadPlaylist()`, compare the current playlist's items with the saved `lastPlayedItem` — if matched, seek the video to that item's `elapsedTime` via `handleProgressBarHover` or direct `currentTime` set.

#### 3. Documentation Cleanup
- **`README.md`** — Still says "An Electron application with Vue and TypeScript" and references nonexistent npm scripts. Rewrite to describe the Wails-based app.
- **`FEATURES.md`** — Describes Electron-era architecture (better-sqlite3, Drizzle ORM, electron-store, electron-updater, vue-router subwindows). Update to match current Wails implementation.

#### 4. Version Bump
- Update `appVersion` in `version.go:19` from `"0.0.0"` to `"0.1.0-beta.1"`
- Ensure `wails.json` version aligns

#### 5. Manual QA Pass
Verify all features before tagging:

| Area | Test Cases |
|---|---|
| Playback | Open native MP4/WebM — play, pause, seek, speed, volume, PiP, fullscreen, auto-next |
| Transcoding | Open MKV (H.264 + AAC) — verify remux path. Open HEVC/AV1 file — verify transcode path |
| Thumbnails | Load playlist — thumbnails generate and display |
| Playlists | Create, rename, delete. Add/remove/reorder items. Switch playlists |
| Persistence | Close and reopen — verify settings, last-watched item, playback position |
| Plugins | Install a plugin, execute an action, verify custom UI rendering |
| Updates | Check for updates, download, install |
| UI | Resize sidebar, context menus, modals, app menu items |

### Nice-to-Have (can defer to v0.1.0 stable)

- **Playlist Search / Filter by Title** — text input to filter videos by name
- **OSD Notifications** — brief on-screen overlays for volume, mute, shuffle, repeat, speed changes

---

## Test Strategy

### Current State
- **No automated tests exist** — zero Go `*_test.go` files, zero frontend test specs
- CI runs only `vue-tsc --noEmit` (TypeScript type-check) and `go vet` (Go static analysis)
- All testing is manual
- No test frameworks are installed

### Test Pyramid

```
          /\
         /E2E\         ~5–10 critical user journeys
        /------\
       /Integra\       ~20–30 service/database/integration tests
      /----------\
     /   Unit     \    ~50+ small, fast unit tests (Go + TypeScript)
    /--------------\
```

### Go Backend Tests

#### Framework
- Use Go's built-in `testing` package (no third-party framework needed)
- Use `testing/slogtest` or custom helpers for structured log assertions
- Use `net/http/httptest` for HTTP handler tests (video server)

#### What to Test

| Package | Test | What It Verifies |
|---|---|---|
| `internal/data/player_repository.go` | `TestCreatePlaylist`, `TestAddItem`, `TestDeleteItem`, `TestUpdateProgress` | SQLite CRUD correctness with in-memory DB (WAL mode edge cases) |
| `internal/data/thumbnail_data_store.go` | `TestGetCachedPath`, `TestCacheHit`, `TestCacheMiss`, `TestInvalidation` | Thumbnail caching logic, hash-based invalidation on file change |
| `internal/data/config.go` | `TestSaveLoad`, `TestRoundTrip`, `TestDefaults` | JSON config read/write, missing file fallback |
| `internal/data/migrations.go` | `TestMigrateFresh`, `TestMigrateIdempotent` | Schema creation and re-application safety |
| `internal/hls/engine.go` | `TestStartStream`, `TestStopStream`, `TestCleanup`, `TestRemuxVsTranscode` | HLS engine start/stop lifecycle, temp dir cleanup |
| `internal/media/probe.go` | `TestProbeNative`, `TestProbeRemux`, `TestProbeTranscode`, `TestHWEncoderDetection` | FFprobe output parsing, codec/container classification, HW encoder availability |
| `internal/plugin/manager.go` | `TestDiscoverPlugins`, `TestInstallFromZip`, `TestRemovePlugin` | Plugin discovery, ZIP extraction, manifest parsing, removal |
| `internal/plugin/lua.go` | `TestExecAction`, `TestHostAPI`, `TestBadScript` | Lua VM execution, host API (exec, emit, addToPlaylist), error handling |
| `app.go` | `TestGetStreamURLNative`, `TestGetStreamURLTranscode`, `TestStopHLSStream` | Stream URL routing, stream lifecycle binding |
| `videoserver.go` | `TestServeVideoRange`, `TestServeHLS`, `Test404`, `TestMethodNotAllowed` | HTTP range requests, HLS directory serving, error codes |

#### Test Fixtures
- Small `.mp4` file (~100KB) for probe/stream tests (committed to `internal/testdata/`)
- Sample `plugin.json` + `main.lua` for plugin tests
- Mock FFprobe/FFmpeg binaries for CI environments without FFmpeg installed

#### Running
```bash
go test ./...                    # all tests
go test ./internal/data/...      # specific package
go test -v -run TestProbe        # specific test
go test -coverprofile=coverage.out ./...
```

### Frontend Tests

#### Framework
- **Vitest** — install with `npm install -D vitest` (native Vite integration)
- **@vue/test-utils** — for component mounting and interaction
- **jsdom** — DOM environment for tests

#### What to Test

| File | Test | What It Verifies |
|---|---|---|
| `composables/usePlayer.ts` | `TogglePlay`, `SetVolume`, `SkipTime`, `Seek`, `ToggleShuffle`, `ToggleRepeatMode`, `ToggleMute` | State transitions and edge cases (negative time, beyond duration) |
| `composables/usePlaylist.ts` | `FilterUnwatched`, `SortByName`, `SortByDuration`, `GenerateShuffleDeck`, `DragReorder` | Playlist filtering, sorting, shuffle deck generation, reorder logic |
| `utils.ts` | `FormatTime`, `FormatTimeZero`, `FormatTimeLarge` | Time formatting edge cases |
| `Player.vue` | `RendersControls`, `TogglePlayClick`, `ProgressBarSeek`, `VolumeSliderChange`, `KeyboardShortcuts`, `HlsPlayback`, `VideoEndAutoNext` | Component rendering, user interaction, keyboard bindings, HLS integration |
| `Playlist.vue` | `RendersItems`, `DragReorder`, `ContextMenu`, `FilterToggle`, `SortChange`, `ThumbnailLoading` | Playlist rendering, drag-drop, context menu, filter/sort controls |
| `Main.vue` | `LoadsPreferences`, `LoadsPlaylist`, `SavesPreferencesDebounced`, `EventListeners` | Startup flow, preference persistence, event wiring |
| `PluginPanel.vue` | `ListsPlugins`, `ExecutesAction`, `InstallsPlugin` | Plugin UI rendering and interaction |
| `UpdateDialog.vue` | `ShowsReleaseNotes`, `DownloadProgress`, `InstallsUpdate` | Update dialog states |

#### Configuration
Add `vitest.config.ts` (or extend `vite.config.ts`):
```ts
/// <reference types="vitest/config" />
export default defineConfig({
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    coverage: { provider: 'v8', reporter: ['text', 'html'] }
  }
})
```

#### Running
```bash
npx vitest                       # watch mode
npx vitest run                   # single run (CI)
npx vitest run --coverage        # with coverage report
```

### End-to-End Tests

#### Approach
E2E testing a Wails desktop app is complex (WebView2 has no standard automation API). For the beta, rely on the **Manual QA Pass** checklist. Consider these options for stable:

- **Playwright** — can automate the frontend in a headless browser context if the Vite dev server is running, but cannot test Wails Go bindings
- **Cypress** — similar limitations
- **Go integration tests** — more practical: write Go tests that start the app, exercise the service layer, and verify database state

#### Auto-Update Tests
- Mock GitHub API with `httptest.NewServer`
- Test version comparison (`semverCompare`)
- Test download with progress tracking
- Test install command construction (do not execute)

### CI Integration

Add a test step to `.github/workflows/ci.yml`:

```yaml
- name: Go tests
  run: go test ./... -v -count=1 -coverprofile=coverage.out

- name: Frontend tests
  run: npx vitest run --coverage
  working-directory: ./frontend
```

For Go tests in CI, mock FFmpeg/FFprobe to avoid requiring them as system dependencies. Create a small `testdata/` directory with minimal media files.

### Manual QA Checklist (Beta Gate)

Before tagging `v0.1.0-beta.1`, run through this checklist on both platforms (Linux + Windows):

#### Playback
- [ ] Open a native `.mp4` (H.264) — plays immediately with correct aspect ratio
- [ ] Open a native `.webm` — plays immediately
- [ ] Play/pause via button click
- [ ] Play/pause via keyboard (Space)
- [ ] Seek via progress bar click
- [ ] Seek via progress bar drag
- [ ] Skip forward 10s (button + Right Arrow)
- [ ] Skip backward 10s (button + Left Arrow)
- [ ] Volume slider adjusts audio
- [ ] Mute toggle works
- [ ] Mute keyboard shortcut (M)
- [ ] Playback speed: all presets (0.5x, 1x, 1.25x, 1.5x, 2x)
- [ ] Fullscreen toggle (button + F)
- [ ] PiP toggle (button + P)
- [ ] Video ends — auto-plays next item
- [ ] Repeat All — loops entire playlist
- [ ] Repeat One — loops current video
- [ ] Shuffle — random order, no repeats until all played

#### HLS Transcoding
- [ ] Open `.mkv` (H.264 + AAC) — plays via remux path (check footer badge)
- [ ] Open `.mkv` (HEVC) — plays via transcode path
- [ ] Open `.avi` — plays via transcode path
- [ ] Progress bar, seek, speed controls work during HLS playback
- [ ] Switching to a different video stops the previous HLS stream
- [ ] Thumbnail preview works during HLS playback

#### Thumbnails
- [ ] Thumbnails appear in playlist sidebar
- [ ] Thumbnails cache on disk (check `~/.cache/graftik-video-player/thumbnails/`)
- [ ] Replacing a video file regenerates its thumbnail

#### Playlists
- [ ] First launch creates "default" playlist
- [ ] File > Add Video — multi-select opens, items appear in playlist
- [ ] Create new playlist (File > New Playlist)
- [ ] Switch between playlists (File > Choose Playlist)
- [ ] Rename playlist
- [ ] Delete playlist (with confirmation)
- [ ] Drag-and-drop reorder items
- [ ] Detailed view / Simple view toggle
- [ ] Sort: A-Z, Z-A, Shortest, Longest, Recently Watched, Oldest Watched
- [ ] Filter: "Unwatched" toggle works
- [ ] Total duration display updates correctly
- [ ] Right-click context menu: Play/Pause, Remove, Open Containing Folder

#### Persistence
- [ ] Close and reopen app — volume, speed, shuffle, repeat, sidebar settings restored
- [ ] Last played item resumes from saved position
- [ ] Window size/position restored
- [ ] Preferences debounced save (change setting, quickly close — still saved)

#### Plugins
- [ ] Plugin panel opens from menu
- [ ] Install a `.zip` plugin from file picker
- [ ] Install a plugin from URL
- [ ] Plugin appears with correct name/version/description
- [ ] Execute a plugin action
- [ ] Plugin with custom UI — modal renders correctly
- [ ] Remove plugin

#### Auto-Update
- [ ] "Check for Updates" in Help menu
- [ ] Progress bar during download
- [ ] Install button launches installer (dpkg/silent)
- [ ] Update badge appears when update is available

#### UI/UX
- [ ] Sidebar resizes via drag handle (min 230px, max 600px)
- [ ] Sidebar visibility toggle
- [ ] Context menus work in playlist
- [ ] All modals open/close correctly
- [ ] App menu: all items work
- [ ] No console errors in WebView2 devtools

---

## Release Process

Per `RELEASE.md`:

1. Push all beta changes to `main`
2. Go to GitHub → Actions → Create Release → Run workflow
3. Set Version: `0.1.0-beta.1`, check **Prerelease**
4. Wait ~15 minutes for build
5. Release appears on Releases page with `.deb` (Linux) and `.exe` (Windows)

The auto-updater checks `GET /releases/latest` which returns the latest non-prerelease by default. Beta users will need a manual download or an opt-in mechanism.
