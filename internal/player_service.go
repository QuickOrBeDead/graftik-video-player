package internal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"graftik-wails/internal/data"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type PlayerService struct {
	ctx            context.Context
	store          *data.PlayerDataStore
	thumbnailStore *data.ThumbnailDataStore
	ffprobePath    string
	ffmpegPath     string
}

func NewPlayerService(store *data.PlayerDataStore, thumbnailStore *data.ThumbnailDataStore) *PlayerService {
	return &PlayerService{
		store:          store,
		thumbnailStore: thumbnailStore,
	}
}

func (s *PlayerService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *PlayerService) SetStore(store *data.PlayerDataStore) {
	s.store = store
}

func (s *PlayerService) SetThumbnailStore(ts *data.ThumbnailDataStore) {
	s.thumbnailStore = ts
}

func (s *PlayerService) SetFFmpegPaths(ffmpegPath, ffprobePath string) {
	s.ffmpegPath = ffmpegPath
	s.ffprobePath = ffprobePath
}

func (s *PlayerService) GetCurrentPlaylist() *data.PlaylistDto {
	playlistID := s.store.GetCurrentPlaylistID()
	if playlistID == "" {
		return nil
	}
	playlist, err := s.store.GetPlaylistByID(playlistID)
	if err != nil || playlist == nil {
		return nil
	}
	return playlist
}

func (s *PlayerService) GetPlaylists() []data.PlaylistListItem {
	items, err := s.store.GetPlaylists()
	if err != nil {
		return nil
	}
	return items
}

func (s *PlayerService) SelectPlaylist(id string) {
	if err := s.store.SetCurrentPlaylistID(id); err != nil {
		return
	}
	playlist, err := s.store.GetPlaylistByID(id)
	if err != nil {
		return
	}
	s.emitEvent("load-current-playlist", playlist)
}

func (s *PlayerService) AddPlaylist(name string) {
	playlist, err := s.store.AddPlaylist(name)
	if err != nil {
		return
	}
	if playlist != nil && playlist.ID != "" {
		s.store.SetCurrentPlaylistID(playlist.ID)
		s.emitEvent("load-current-playlist", playlist)
	}
}

func (s *PlayerService) UpdatePlaylistName(id, name string) {
	s.store.UpdatePlaylist(id, map[string]any{"name": name})
	s.emitEvent("load-playlist-name")
}

func (s *PlayerService) UpdatePlaylist(id string, data map[string]any) {
	s.store.UpdatePlaylist(id, data)
}

func (s *PlayerService) DeletePlaylist(id string) {
	s.store.DeletePlaylist(id)
}

func (s *PlayerService) AddPlaylistItems(items []data.PlaylistItemDto) {
	s.store.AddPlaylistItems(items)
}

func (s *PlayerService) UpdatePlaylistItem(id string, data map[string]any) {
	s.store.UpdatePlaylistItem(id, data)
}

func (s *PlayerService) DeletePlaylistItem(id string) {
	s.store.DeletePlaylistItem(id)
}

func (s *PlayerService) GetPlaylist(id string) *data.PlaylistDto {
	playlist, err := s.store.GetPlaylistByID(id)
	if err != nil {
		return nil
	}
	return playlist
}

func (s *PlayerService) GetPlaylistName(id string) string {
	name, err := s.store.GetPlaylistName(id)
	if err != nil {
		return ""
	}
	return name
}

func (s *PlayerService) GetPlaylistItemVideoMetadata(playlistID, playlistItemID, videoPath string) *data.VideoMetadata {
	stats, err := os.Stat(videoPath)
	if err != nil {
		return nil
	}
	lastModified := float64(stats.ModTime().UnixMilli())
	fileSize := float64(stats.Size())

	fileHash := s.thumbnailStore.CalculateFileHash(videoPath, stats.Size(), stats.ModTime().UnixMilli())

	// Check cache
	thumbnail, _ := s.thumbnailStore.GetThumbnail(playlistID, playlistItemID, fileHash)
	if thumbnail != "" {
		duration := s.probeDuration(videoPath)
		return &data.VideoMetadata{
			Duration:     duration,
			LastModified: lastModified,
			FileSize:     fileSize,
			Thumbnail:    thumbnail,
		}
	}

	// Probe duration
	duration := s.probeDuration(videoPath)

	// Extract thumbnail
	seekTime := duration * 0.1
	if seekTime <= 0 {
		seekTime = 1.0
	}

	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d.jpeg", playlistItemID, time.Now().UnixMilli()))
	defer os.Remove(tempFile)

	var ffmpegStderr bytes.Buffer
	extractCmd := exec.Command(s.ffmpegPath,
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
		fmt.Printf("ffmpeg thumbnail error: %s\n", ffmpegStderr.String())
		return &data.VideoMetadata{
			Duration:     duration,
			LastModified: lastModified,
			FileSize:     fileSize,
			Thumbnail:    "",
		}
	}

	imageData, err := os.ReadFile(tempFile)
	if err != nil {
		return &data.VideoMetadata{
			Duration:     duration,
			LastModified: lastModified,
			FileSize:     fileSize,
			Thumbnail:    "",
		}
	}

	s.thumbnailStore.SetThumbnail(playlistID, playlistItemID, fileHash, imageData)

	return &data.VideoMetadata{
		Duration:     duration,
		LastModified: lastModified,
		FileSize:     fileSize,
		Thumbnail:    "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(imageData),
	}
}

func (s *PlayerService) RebalancePlaylistOrder(id string) {
	s.store.RebalancePlaylistOrder(id)
}

func (s *PlayerService) OpenContainingFolder(filePath string) {
	exec.Command("explorer", "/select,", filePath).Start()
}

func (s *PlayerService) InitNewPlaylistItems(filePaths []string) []data.PlaylistItemDto {
	items := make([]data.PlaylistItemDto, len(filePaths))
	for i, fp := range filePaths {
		items[i] = data.PlaylistItemDto{
			ID:          uuid.New().String(),
			Path:        fp,
			Title:       filepath.Base(fp),
			OrderIndex:  0,
			PlaylistID:  "",
		}
	}
	return items
}

func (s *PlayerService) probeDuration(videoPath string) float64 {
	var ffprobeStderr bytes.Buffer
	cmd := exec.Command(s.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		videoPath,
	)
	cmd.Stderr = &ffprobeStderr
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("ffprobe error: %s\n", ffprobeStderr.String())
		return 0
	}

	format := struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}{}
	if err := json.Unmarshal(output, &format); err != nil {
		return 0
	}

	var duration float64
	if _, err := fmt.Sscanf(format.Format.Duration, "%f", &duration); err != nil {
		return 0
	}
	return math.Round(duration*100) / 100
}

func (s *PlayerService) emitEvent(event string, data ...any) {
	if s.ctx == nil {
		return
	}
	if len(data) > 0 {
		runtime.EventsEmit(s.ctx, event, data[0])
	} else {
		runtime.EventsEmit(s.ctx, event)
	}
}
