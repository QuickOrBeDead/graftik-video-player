package internal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	goruntime "runtime"
	"time"

	"github.com/QuickOrBeDead/graftik-video-player/internal/command"
	"github.com/QuickOrBeDead/graftik-video-player/internal/data"
	"github.com/QuickOrBeDead/graftik-video-player/internal/hls"
	graftikLogger "github.com/QuickOrBeDead/graftik-video-player/internal/logger"
	"github.com/QuickOrBeDead/graftik-video-player/internal/media"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type PlayerService struct {
	ctx            context.Context
	store          *data.PlayerDataStore
	thumbnailStore *data.ThumbnailDataStore
	ffprobePath    string
	ffmpegPath     string
	hlsEngine      *hls.Engine
	log            graftikLogger.Logger
	prober         *media.Prober
}

func NewPlayerService(store *data.PlayerDataStore, thumbnailStore *data.ThumbnailDataStore, hlsEngine *hls.Engine, ffmpegPath, ffprobePath string, log graftikLogger.Logger, prober *media.Prober) *PlayerService {
	if log == nil {
		panic("logger must not be nil")
	}
	if store == nil {
		panic("player data store must not be nil")
	}
	if prober == nil {
		panic("prober must not be nil")
	}
	return &PlayerService{
		store:          store,
		thumbnailStore: thumbnailStore,
		hlsEngine:      hlsEngine,
		ffmpegPath:     ffmpegPath,
		ffprobePath:    ffprobePath,
		log:            log,
		prober:         prober,
	}
}

func (s *PlayerService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *PlayerService) FFprobePath() string {
	return s.ffprobePath
}

func (s *PlayerService) FFmpegPath() string {
	return s.ffmpegPath
}

func (s *PlayerService) GetCurrentPlaylist() *data.PlaylistDto {
	s.log.Debug("GetCurrentPlaylist: started")
	playlistID := s.store.GetCurrentPlaylistID()
	if playlistID == "" {
		s.log.Debug("GetCurrentPlaylist: no current playlist set")
		return nil
	}
	playlist, err := s.store.GetPlaylistByID(playlistID)
	if err != nil || playlist == nil {
		s.log.Error("GetCurrentPlaylist: failed", "playlistID", playlistID, "error", err)
		return nil
	}
	return playlist
}

func (s *PlayerService) GetPlaylists() []data.PlaylistListItem {
	s.log.Debug("GetPlaylists: started")
	items, err := s.store.GetPlaylists()
	if err != nil {
		s.log.Error("GetPlaylists: failed", "error", err)
		return nil
	}

	s.log.Debug("GetPlaylists: finished", "count", len(items))

	return items
}

func (s *PlayerService) SelectPlaylist(id string) {
	s.log.Debug("SelectPlaylist: started", "id", id)
	if err := s.store.SetCurrentPlaylistID(id); err != nil {
		s.log.Error("SelectPlaylist: SetCurrentPlaylistID failed", "id", id, "error", err)
		return
	}
	playlist, err := s.store.GetPlaylistByID(id)
	if err != nil {
		s.log.Error("SelectPlaylist: GetPlaylistByID failed", "id", id, "error", err)
		return
	}
	s.emitEvent("load-current-playlist", playlist)
	s.log.Debug("SelectPlaylist: finished", "playlist", playlist)
}

func (s *PlayerService) AddPlaylist(name string) {
	s.log.Debug("AddPlaylist: started", "name", name)
	playlist, err := s.store.AddPlaylist(name)
	if err != nil {
		s.log.Error("AddPlaylist: AddPlaylist failed", "name", name, "error", err)
		return
	}
	if playlist != nil && playlist.ID != "" {
		s.store.SetCurrentPlaylistID(playlist.ID)
		s.emitEvent("load-current-playlist", playlist)
	}

	s.log.Debug("AddPlaylist: finished")
}

func (s *PlayerService) UpdatePlaylistName(id, name string) {
	s.log.Debug("UpdatePlaylistName: started", "id", id, "name", name)
	s.store.UpdatePlaylist(id, map[string]any{"name": name})
	s.emitEvent("load-playlist-name")
	s.log.Debug("UpdatePlaylistName: finished")
}

func (s *PlayerService) UpdatePlaylist(id string, data map[string]any) {
	s.log.Debug("UpdatePlaylist: started", "id", id, "data", data)
	s.store.UpdatePlaylist(id, data)
	s.log.Debug("UpdatePlaylist: finished")
}

func (s *PlayerService) DeletePlaylist(id string) {
	s.log.Debug("DeletePlaylist: started", "id", id)
	s.store.DeletePlaylist(id)
	s.log.Debug("DeletePlaylist: finished")
}

func (s *PlayerService) AddPlaylistItems(items []data.PlaylistItemDto) {
	s.log.Debug("AddPlaylistItems: started", "count", len(items))
	s.store.AddPlaylistItems(items)
	s.log.Debug("AddPlaylistItems: finished")
}

func (s *PlayerService) UpdatePlaylistItem(id string, data map[string]any) {
	s.log.Debug("UpdatePlaylistItem: started", "id", id, "data", data)
	s.store.UpdatePlaylistItem(id, data)
	s.log.Debug("UpdatePlaylistItem: finished")
}

func (s *PlayerService) DeletePlaylistItem(id string) {
	s.log.Debug("DeletePlaylistItem: started", "id", id)
	s.store.DeletePlaylistItem(id)
	s.log.Debug("DeletePlaylistItem: finished")
}

func (s *PlayerService) GetPlaylist(id string) *data.PlaylistDto {
	s.log.Debug("GetPlaylist: started", "id", id)
	playlist, err := s.store.GetPlaylistByID(id)
	if err != nil {
		s.log.Error("GetPlaylist: failed", "id", id, "error", err)
		return nil
	}
	s.log.Debug("GetPlaylist: finished", "playlist", playlist)
	return playlist
}

func (s *PlayerService) GetPlaylistName(id string) string {
	s.log.Debug("GetPlaylistName: started", "id", id)
	name, err := s.store.GetPlaylistName(id)
	if err != nil {
		s.log.Error("GetPlaylistName: failed", "id", id, "error", err)
		return ""
	}
	s.log.Debug("GetPlaylistName: finished", "name", name)
	return name
}

func (s *PlayerService) GetPlaylistItemVideoMetadata(playlistID, playlistItemID, videoPath string) *data.VideoMetadata {
	s.log.Debug("GetPlaylistItemVideoMetadata: started", "playlistID", playlistID, "playlistItemID", playlistItemID, "videoPath", videoPath)

	stats, err := os.Stat(videoPath)
	if err != nil {
		s.log.Error("GetPlaylistItemVideoMetadata: file stat failed", "path", videoPath, "error", err)
		return nil
	}
	lastModified := float64(stats.ModTime().UnixMilli())
	fileSize := float64(stats.Size())

	s.log.Debug("GetPlaylistItemVideoMetadata: file stat", "path", videoPath, "size", fileSize, "lastModified", lastModified)

	fileHash := s.thumbnailStore.CalculateFileHash(videoPath, stats.Size(), stats.ModTime().UnixMilli())

	// Check cache
	thumbnail, err := s.thumbnailStore.GetThumbnail(playlistID, playlistItemID, fileHash)
	if err != nil {
		s.log.Debug("GetPlaylistItemVideoMetadata: thumbnail cache lookup failed", "playlistID", playlistID, "playlistItemID", playlistItemID, "error", err)
	}
	if thumbnail != "" {
		s.log.Debug("GetPlaylistItemVideoMetadata: thumbnail cache hit", "fileHash", fileHash, "playlistItemID", playlistItemID)
		duration := s.probeDuration(videoPath)
		md := &data.VideoMetadata{
			Duration:     duration,
			LastModified: lastModified,
			FileSize:     fileSize,
			Thumbnail:    thumbnail,
		}
		s.log.Debug("GetPlaylistItemVideoMetadata: finished", "duration", md.Duration, "fileSize", md.FileSize, "lastModified", md.LastModified)
		return md
	}

	s.log.Debug("GetPlaylistItemVideoMetadata: thumbnail cache miss", "fileHash", fileHash, "playlistItemID", playlistItemID)

	// Probe duration
	duration := s.probeDuration(videoPath)
	s.log.Debug("GetPlaylistItemVideoMetadata: probed duration", "path", videoPath, "duration", duration)

	// Extract thumbnail
	seekTime := duration * 0.1
	if seekTime <= 0 {
		seekTime = 1.0
	}

	s.log.Debug("GetPlaylistItemVideoMetadata: extracting thumbnail", "path", videoPath, "seekTime", seekTime)

	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d.jpeg", playlistItemID, time.Now().UnixMilli()))
	defer os.Remove(tempFile)

	var ffmpegStderr bytes.Buffer
	extractCmd := command.CreateHiddenCmd(s.ffmpegPath,
		"-ss", fmt.Sprintf("%.1f", seekTime),
		"-i", videoPath,
		"-vframes", "1",
		"-f", "image2",
		"-vcodec", "mjpeg",
		"-q:v", "4",
		"-vf", "scale=180:-2",
		"-sws_flags", "fast_bilinear",
		tempFile,
	)
	extractCmd.Stderr = &ffmpegStderr
	if err := extractCmd.Run(); err != nil {
		stderr := ffmpegStderr.String()
		if stderr == "" {
			stderr = err.Error()
		}
		s.log.Error("GetPlaylistItemVideoMetadata: ffmpeg thumbnail extraction failed", "path", videoPath, "error", stderr)
		md := &data.VideoMetadata{
			Duration:     duration,
			LastModified: lastModified,
			FileSize:     fileSize,
			Thumbnail:    "",
		}
		s.log.Debug("GetPlaylistItemVideoMetadata: finished", "duration", md.Duration, "fileSize", md.FileSize, "lastModified", md.LastModified)
		return md
	}

	s.log.Debug("GetPlaylistItemVideoMetadata: thumbnail extracted to temp file", "tempFile", tempFile)

	imageData, err := os.ReadFile(tempFile)
	if err != nil {
		s.log.Error("GetPlaylistItemVideoMetadata: failed to read temp thumbnail", "tempFile", tempFile, "error", err)
		md := &data.VideoMetadata{
			Duration:     duration,
			LastModified: lastModified,
			FileSize:     fileSize,
			Thumbnail:    "",
		}
		s.log.Debug("GetPlaylistItemVideoMetadata: finished", "duration", md.Duration, "fileSize", md.FileSize, "lastModified", md.LastModified)
		return md
	}

	s.thumbnailStore.SetThumbnail(playlistID, playlistItemID, fileHash, imageData)

	s.log.Debug("GetPlaylistItemVideoMetadata: thumbnail cached and returning", "playlistItemID", playlistItemID, "imageSize", len(imageData))

	md := &data.VideoMetadata{
		Duration:     duration,
		LastModified: lastModified,
		FileSize:     fileSize,
		Thumbnail:    "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(imageData),
	}
	s.log.Debug("GetPlaylistItemVideoMetadata: finished", "duration", md.Duration, "fileSize", md.FileSize, "lastModified", md.LastModified)
	return md
}

func (s *PlayerService) RebalancePlaylistOrder(id string) {
	s.log.Debug("RebalancePlaylistOrder: started", "id", id)
	s.store.RebalancePlaylistOrder(id)
	s.log.Debug("RebalancePlaylistOrder: finished")
}

func (s *PlayerService) OpenContainingFolder(filePath string) {
	s.log.Debug("OpenContainingFolder: started", "filePath", filePath)

	switch goruntime.GOOS {
	case "windows":
		if err := command.CreateHiddenCmd("explorer", "/select,", filePath).Start(); err != nil {
			s.log.Error("OpenContainingFolder: failed to open explorer", "path", filePath, "error", err)
		}
	case "darwin":
		if err := command.CreateHiddenCmd("open", "-R", filePath).Start(); err != nil {
			s.log.Error("OpenContainingFolder: failed to open finder", "path", filePath, "error", err)
		}
	default:
		if err := command.CreateHiddenCmd("xdg-open", filepath.Dir(filePath)).Start(); err != nil {
			s.log.Error("OpenContainingFolder: failed to open xdg-open", "path", filepath.Dir(filePath), "error", err)
		}
	}

	s.log.Debug("OpenContainingFolder: finished")
}

func (s *PlayerService) InitNewPlaylistItems(filePaths []string) []data.PlaylistItemDto {
	s.log.Debug("InitNewPlaylistItems: started", "count", len(filePaths))
	items := make([]data.PlaylistItemDto, len(filePaths))
	for i, fp := range filePaths {
		items[i] = data.PlaylistItemDto{
			ID:         uuid.New().String(),
			Path:       fp,
			Title:      filepath.Base(fp),
			OrderIndex: 0,
			PlaylistID: "",
		}
	}
	s.log.Debug("InitNewPlaylistItems: finished", "count", len(items))
	return items
}

func (s *PlayerService) GetStreamInfo(videoPath string) *data.StreamInfo {
	s.log.Debug("GetStreamInfo: started", "videoPath", videoPath)

	info, err := s.prober.Probe(s.ffprobePath, videoPath)
	if err != nil {
		s.log.Error("GetStreamInfo: media probe failed, falling back to SW transcode", "path", videoPath, "error", err)
		info := &data.StreamInfo{
			Action:      "sw_transcode",
			ActionLabel: "SW Transcode",
		}
		s.log.Debug("GetStreamInfo: finished", "action", info.Action, "actionLabel", info.ActionLabel)
		return info
	}

	s.log.Debug("GetStreamInfo: media probe result", "path", videoPath, "action", info.Action, "actionLabel", info.ActionLabel)

	if info.Action == "sw_transcode" && s.hlsEngine != nil {
		hwEncoder := s.prober.DetectHWEncoder(s.ffmpegPath)
		if hwEncoder != "" {
			info.Action = "hw_transcode"
			info.ActionLabel = hwEncoderShortLabel(hwEncoder)
			info.HWEncoder = hwEncoderShortLabel(hwEncoder)
			s.log.Debug("GetStreamInfo: hw encoder detected, upgrading to HW transcode", "encoder", hwEncoder, "path", videoPath)
		} else {
			s.log.Debug("GetStreamInfo: no hw encoder detected, keeping SW transcode", "path", videoPath)
		}
	}

	s.log.Debug("GetStreamInfo: finished", "action", info.Action, "actionLabel", info.ActionLabel)
	return info
}

func hwEncoderShortLabel(name string) string {
	switch name {
	case "h264_nvenc":
		return "NVENC"
	case "h264_qsv":
		return "QSV"
	case "h264_amf":
		return "AMF"
	}
	return name
}

func (s *PlayerService) GetPreferences() *data.AppConfig {
	s.log.Debug("GetPreferences: started")
	cfg := s.store.GetPreferences()
	s.log.Debug("GetPreferences: finished", "cfg", cfg)
	return cfg
}

func (s *PlayerService) SavePreferences(settings map[string]any) {
	s.log.Debug("SavePreferences: started", "settings", settings)
	s.store.UpdateSettings(settings)
	s.log.Debug("SavePreferences: finished")
}

func (s *PlayerService) probeDuration(videoPath string) float64 {
	s.log.Debug("probeDuration: started", "videoPath", videoPath)
	var ffprobeStderr bytes.Buffer
	cmd := command.CreateHiddenCmd(s.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		videoPath,
	)
	cmd.Stderr = &ffprobeStderr
	output, err := cmd.Output()
	if err != nil {
		stderr := ffprobeStderr.String()
		if stderr == "" {
			stderr = err.Error()
		}
		s.log.Error("ffprobe duration probe failed", "path", videoPath, "error", err)
		return 0
	}

	format := struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}{}
	if err := json.Unmarshal(output, &format); err != nil {
		s.log.Error("probeDuration: json unmarshal failed", "path", videoPath, "output", string(output))
		return 0
	}

	var duration float64
	if _, err := fmt.Sscanf(format.Format.Duration, "%f", &duration); err != nil {
		s.log.Error("probeDuration: failed to parse duration", "path", videoPath, "durationStr", format.Format.Duration)
		return 0
	}
	duration = math.Round(duration*100) / 100
	s.log.Debug("probeDuration: finished", "duration", duration)
	return duration
}

func (s *PlayerService) emitEvent(event string, data ...any) {
	s.log.Debug("emitEvent: started", "event", event, "data", data)
	if s.ctx == nil {
		s.log.Error("emitEvent: context is nil", "event", event)
		return
	}
	if len(data) > 0 {
		runtime.EventsEmit(s.ctx, event, data[0])
	} else {
		runtime.EventsEmit(s.ctx, event)
	}
	s.log.Debug("emitEvent: finished")
}
