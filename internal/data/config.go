package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AppConfig struct {
	CurrentPlaylist string  `json:"currentPlaylist"`
	Shuffle         bool    `json:"shuffle"`
	RepeatMode      int     `json:"repeatMode"`
	VolumeLevel     float64 `json:"volumeLevel"`
	PlaybackRate    float64 `json:"playbackRate"`
	SidebarVisible  bool    `json:"sidebarVisible"`
	SidebarWidth    int     `json:"sidebarWidth"`
	WindowWidth     int     `json:"windowWidth"`
	WindowHeight    int     `json:"windowHeight"`
	IsPlaying                    bool   `json:"isPlaying"`
	LastPlayedItem               string `json:"lastPlayedItem"`
	IncludePrereleasesForUpdates bool   `json:"includePrereleasesForUpdates"`
}

type ConfigStore struct {
	configPath string
	config     AppConfig
}

func NewConfigStore(userDataPath string) *ConfigStore {
	return &ConfigStore{
		configPath: filepath.Join(userDataPath, "config.json"),
	}
}

func (c *ConfigStore) Load() error {
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.config = AppConfig{
				VolumeLevel:    1.0,
				PlaybackRate:   1.0,
				SidebarVisible: true,
				SidebarWidth:   300,
			}
			return nil
		}
		return fmt.Errorf("failed to read config: %w", err)
	}
	if err := json.Unmarshal(data, &c.config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	return nil
}

func (c *ConfigStore) Save() error {
	dir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	data, err := json.MarshalIndent(c.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(c.configPath, data, 0644)
}

func (c *ConfigStore) GetCurrentPlaylist() string {
	return c.config.CurrentPlaylist
}

func (c *ConfigStore) SetCurrentPlaylist(id string) error {
	c.config.CurrentPlaylist = id
	return c.Save()
}

func (c *ConfigStore) GetPreferences() *AppConfig {
	return &AppConfig{
		CurrentPlaylist:              c.config.CurrentPlaylist,
		Shuffle:                      c.config.Shuffle,
		RepeatMode:                   c.config.RepeatMode,
		VolumeLevel:                  c.config.VolumeLevel,
		PlaybackRate:                 c.config.PlaybackRate,
		SidebarVisible:               c.config.SidebarVisible,
		SidebarWidth:                 c.config.SidebarWidth,
		WindowWidth:                  c.config.WindowWidth,
		WindowHeight:                 c.config.WindowHeight,
		IsPlaying:                    c.config.IsPlaying,
		LastPlayedItem:               c.config.LastPlayedItem,
		IncludePrereleasesForUpdates: c.config.IncludePrereleasesForUpdates,
	}
}

func (c *ConfigStore) UpdateSettings(settings map[string]any) error {
	if v, ok := settings["shuffle"]; ok {
		c.config.Shuffle = v.(bool)
	}
	if v, ok := settings["repeatMode"]; ok {
		c.config.RepeatMode = int(v.(float64))
	}
	if v, ok := settings["volumeLevel"]; ok {
		c.config.VolumeLevel = v.(float64)
	}
	if v, ok := settings["playbackRate"]; ok {
		c.config.PlaybackRate = v.(float64)
	}
	if v, ok := settings["sidebarVisible"]; ok {
		c.config.SidebarVisible = v.(bool)
	}
	if v, ok := settings["sidebarWidth"]; ok {
		c.config.SidebarWidth = int(v.(float64))
	}
	if v, ok := settings["windowWidth"]; ok {
		c.config.WindowWidth = int(v.(float64))
	}
	if v, ok := settings["windowHeight"]; ok {
		c.config.WindowHeight = int(v.(float64))
	}
	if v, ok := settings["isPlaying"]; ok {
		c.config.IsPlaying = v.(bool)
	}
	if v, ok := settings["lastPlayedItem"]; ok {
		c.config.LastPlayedItem = v.(string)
	}
	if v, ok := settings["includePrereleasesForUpdates"]; ok {
		c.config.IncludePrereleasesForUpdates = v.(bool)
	}
	return c.Save()
}

func (c *ConfigStore) GetWindowSize() (int, int) {
	return c.config.WindowWidth, c.config.WindowHeight
}
