package data

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
)

type PlayerDataStore struct {
	repo   *PlayerRepository
	config *ConfigStore
}

type addItemInput struct {
	ID         string
	PlaylistID string
	Path       string
	Title      string
	OrderIndex float64
}

func NewPlayerDataStore(userDataPath, dbPath string) (*PlayerDataStore, error) {
	repo, err := NewPlayerRepository(dbPath)
	if err != nil {
		return nil, err
	}
	config := NewConfigStore(userDataPath)
	if err := config.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &PlayerDataStore{repo: repo, config: config}, nil
}

func (s *PlayerDataStore) Initialize() error {
	return s.repo.InitializeDB()
}

func (s *PlayerDataStore) Close() error {
	return s.repo.Close()
}

func (s *PlayerDataStore) GetCurrentPlaylistID() string {
	return s.config.GetCurrentPlaylist()
}

func (s *PlayerDataStore) SetCurrentPlaylistID(id string) error {
	return s.config.SetCurrentPlaylist(id)
}

func (s *PlayerDataStore) GetPlaylistByID(id string) (*PlaylistDto, error) {
	if err := s.repo.RebalancePlaylistOrder(id); err != nil {
		return nil, err
	}
	return s.repo.GetPlaylist(id)
}

func (s *PlayerDataStore) GetPlaylistName(id string) (string, error) {
	result, err := s.repo.GetPlaylistProjection(id, []string{"name"})
	if err != nil || result == nil {
		return "", err
	}
	name, _ := result["name"].(string)
	return name, nil
}

func (s *PlayerDataStore) GetPlaylists() ([]PlaylistListItem, error) {
	return s.repo.GetPlaylists()
}

func (s *PlayerDataStore) AddDefaultPlaylist() error {
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
	id := uuid.New().String()
	if err := s.repo.AddPlaylist(id, name); err != nil {
		return nil, err
	}
	return s.repo.GetPlaylist(id)
}

func (s *PlayerDataStore) UpdatePlaylist(id string, data map[string]any) error {
	return s.repo.UpdatePlaylist(id, data)
}

func (s *PlayerDataStore) GetPlaylistItem(id string) *PlaylistItemDto {
	return s.repo.GetPlaylistItemByID(id)
}

func (s *PlayerDataStore) UpdatePlaylistItem(id string, data map[string]any) error {
	return s.repo.UpdatePlaylistItem(id, data)
}

func (s *PlayerDataStore) DeletePlaylist(id string) error {
	if err := s.repo.DeletePlaylist(id); err != nil {
		return err
	}
	if s.config.GetCurrentPlaylist() == id {
		return s.config.SetCurrentPlaylist("")
	}
	return nil
}

func (s *PlayerDataStore) DeletePlaylistItem(id string) error {
	return s.repo.DeletePlaylistItem(id)
}

func (s *PlayerDataStore) InitNewPlaylistItems(filePaths []string) []PlaylistItemDto {
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
	return s.repo.RebalancePlaylistOrder(id)
}

func (s *PlayerDataStore) GetPreferences() *AppConfig {
	return s.config.GetPreferences()
}

func (s *PlayerDataStore) UpdateSettings(settings map[string]any) error {
	return s.config.UpdateSettings(settings)
}
