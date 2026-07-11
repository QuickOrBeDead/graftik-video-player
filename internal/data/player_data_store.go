package data

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	graftikLogger "github.com/QuickOrBeDead/graftik-video-player/internal/logger"
)

type PlayerDataStore struct {
	repo   *PlayerRepository
	config *ConfigStore
	log    graftikLogger.Logger
}

type addItemInput struct {
	ID         string
	PlaylistID string
	Path       string
	Title      string
	OrderIndex float64
}

func NewPlayerDataStore(userDataPath, dbPath string, log graftikLogger.Logger) (*PlayerDataStore, error) {
	if log == nil {
		panic("data: logger is required")
	}
	log.Debug("data: creating player data store", "userDataPath", userDataPath, "dbPath", dbPath)
	repo, err := NewPlayerRepository(dbPath, log)
	if err != nil {
		return nil, err
	}
	config := NewConfigStore(userDataPath, log)
	if err := config.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &PlayerDataStore{repo: repo, config: config, log: log}, nil
}

func (s *PlayerDataStore) Initialize() error {
	s.log.Debug("data: initializing database")
	return s.repo.InitializeDB()
}

func (s *PlayerDataStore) Close() error {
	s.log.Debug("data: closing player data store")
	return s.repo.Close()
}

func (s *PlayerDataStore) GetCurrentPlaylistID() string {
	id := s.config.GetCurrentPlaylist()
	s.log.Debug("data: get current playlist id", "id", id)
	return id
}

func (s *PlayerDataStore) SetCurrentPlaylistID(id string) error {
	s.log.Debug("data: set current playlist id", "id", id)
	return s.config.SetCurrentPlaylist(id)
}

func (s *PlayerDataStore) GetPlaylistByID(id string) (*PlaylistDto, error) {
	s.log.Debug("data: getting playlist by id", "id", id)
	if err := s.repo.RebalancePlaylistOrder(id); err != nil {
		return nil, err
	}
	return s.repo.GetPlaylist(id)
}

func (s *PlayerDataStore) GetPlaylistName(id string) (string, error) {
	s.log.Debug("data: getting playlist name", "id", id)
	result, err := s.repo.GetPlaylistProjection(id, []string{"name"})
	if err != nil || result == nil {
		return "", err
	}
	name, _ := result["name"].(string)
	return name, nil
}

func (s *PlayerDataStore) GetPlaylists() ([]PlaylistListItem, error) {
	s.log.Debug("data: getting all playlists")
	return s.repo.GetPlaylists()
}

func (s *PlayerDataStore) AddDefaultPlaylist() error {
	s.log.Debug("data: adding default playlist")
	exists, err := s.repo.HasAnyPlaylist()
	if err != nil {
		return err
	}
	if !exists {
		id := uuid.New().String()
		if err := s.repo.AddPlaylist(id, "default"); err != nil {
			return err
		}
		return s.config.SetCurrentPlaylist(id)
	}

	// Playlists exist but no current playlist set — pick the first one
	if s.config.GetCurrentPlaylist() == "" {
		playlists, err := s.repo.GetPlaylists()
		if err != nil {
			return err
		}
		if len(playlists) > 0 {
			return s.config.SetCurrentPlaylist(playlists[0].ID)
		}
	}
	return nil
}

func (s *PlayerDataStore) AddPlaylist(name string) (*PlaylistDto, error) {
	s.log.Debug("data: adding playlist", "name", name)
	id := uuid.New().String()
	if err := s.repo.AddPlaylist(id, name); err != nil {
		return nil, err
	}
	return s.repo.GetPlaylist(id)
}

func (s *PlayerDataStore) UpdatePlaylist(id string, data map[string]any) error {
	s.log.Debug("data: updating playlist", "id", id)
	return s.repo.UpdatePlaylist(id, data)
}

func (s *PlayerDataStore) GetPlaylistItem(id string) *PlaylistItemDto {
	s.log.Debug("data: getting playlist item", "id", id)
	return s.repo.GetPlaylistItemByID(id)
}

func (s *PlayerDataStore) UpdatePlaylistItem(id string, data map[string]any) error {
	s.log.Debug("data: updating playlist item", "id", id)
	return s.repo.UpdatePlaylistItem(id, data)
}

func (s *PlayerDataStore) DeletePlaylist(id string) error {
	s.log.Debug("data: deleting playlist", "id", id)
	if err := s.repo.DeletePlaylist(id); err != nil {
		return err
	}
	if s.config.GetCurrentPlaylist() == id {
		return s.config.SetCurrentPlaylist("")
	}
	return nil
}

func (s *PlayerDataStore) DeletePlaylistItem(id string) error {
	s.log.Debug("data: deleting playlist item", "id", id)
	return s.repo.DeletePlaylistItem(id)
}

func (s *PlayerDataStore) InitNewPlaylistItems(filePaths []string) []PlaylistItemDto {
	s.log.Debug("data: initiating new playlist items", "count", len(filePaths))
	items := make([]PlaylistItemDto, len(filePaths))
	for i, fp := range filePaths {
		items[i] = PlaylistItemDto{
			ID:         uuid.New().String(),
			Path:       fp,
			Title:      filepath.Base(fp),
			OrderIndex: 0,
		}
	}
	return items
}

func (s *PlayerDataStore) AddPlaylistItems(items []PlaylistItemDto) {
	s.log.Debug("data: adding playlist items", "count", len(items))
	input := make([]struct {
		ID         string
		PlaylistID string
		Path       string
		Title      string
		OrderIndex float64
	}, len(items))
	for i, item := range items {
		input[i] = struct {
			ID         string
			PlaylistID string
			Path       string
			Title      string
			OrderIndex float64
		}{
			ID:         item.ID,
			PlaylistID: item.PlaylistID,
			Path:       item.Path,
			Title:      item.Title,
			OrderIndex: item.OrderIndex,
		}
	}
	s.repo.AddPlaylistItems(input)
}

func (s *PlayerDataStore) RebalancePlaylistOrder(id string) error {
	s.log.Debug("data: rebalancing playlist order", "id", id)
	return s.repo.RebalancePlaylistOrder(id)
}

func (s *PlayerDataStore) GetPreferences() *AppConfig {
	s.log.Debug("data: getting preferences")
	return s.config.GetPreferences()
}

func (s *PlayerDataStore) UpdateSettings(settings map[string]any) error {
	s.log.Debug("data: updating settings")
	return s.config.UpdateSettings(settings)
}
