package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	graftikLogger "github.com/QuickOrBeDead/graftik-video-player/internal/logger"
)

type AppConfig struct {
	CurrentPlaylist              string  `json:"currentPlaylist"`
	Shuffle                      bool    `json:"shuffle"`
	RepeatMode                   int     `json:"repeatMode"`
	VolumeLevel                  float64 `json:"volumeLevel"`
	PlaybackRate                 float64 `json:"playbackRate"`
	SidebarVisible               bool    `json:"sidebarVisible"`
	SidebarWidth                 int     `json:"sidebarWidth"`
	WindowWidth                  int     `json:"windowWidth"`
	WindowHeight                 int     `json:"windowHeight"`
	IncludePrereleasesForUpdates bool    `json:"includePrereleasesForUpdates"`
}

type ConfigStore struct {
	configPath string
	config     AppConfig
	log        graftikLogger.Logger
}

func NewConfigStore(userDataPath string, log graftikLogger.Logger) *ConfigStore {
	if log == nil {
		panic("data: logger is required")
	}
	log.Debug("config: creating config store", "userDataPath", userDataPath)
	return &ConfigStore{
		configPath: filepath.Join(userDataPath, "config.json"),
		log:        log,
	}
}

func (c *ConfigStore) Load() error {
	c.log.Debug("config: loading config", "path", c.configPath)
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
	c.log.Debug("config: config loaded", "path", c.configPath)
	return nil
}

func (c *ConfigStore) Save() error {
	c.log.Debug("config: saving config", "path", c.configPath)
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
	c.log.Debug("config: setting current playlist", "id", id)
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
		IncludePrereleasesForUpdates: c.config.IncludePrereleasesForUpdates,
	}
}

func (c *ConfigStore) UpdateSettings(settings map[string]any) error {
	c.log.Debug("config: updating settings", "settings", settings)
	if v, ok := settings["shuffle"]; ok {
		if val, ok := v.(bool); ok {
			c.config.Shuffle = val
		}
	}
	if v, ok := settings["repeatMode"]; ok {
		if val, ok := v.(float64); ok {
			c.config.RepeatMode = int(val)
		}
	}
	if v, ok := settings["volumeLevel"]; ok {
		if val, ok := v.(float64); ok {
			c.config.VolumeLevel = val
		}
	}
	if v, ok := settings["playbackRate"]; ok {
		if val, ok := v.(float64); ok {
			c.config.PlaybackRate = val
		}
	}
	if v, ok := settings["sidebarVisible"]; ok {
		if val, ok := v.(bool); ok {
			c.config.SidebarVisible = val
		}
	}
	if v, ok := settings["sidebarWidth"]; ok {
		if val, ok := v.(float64); ok {
			c.config.SidebarWidth = int(val)
		}
	}
	if v, ok := settings["windowWidth"]; ok {
		if val, ok := v.(float64); ok {
			c.config.WindowWidth = int(val)
		}
	}
	if v, ok := settings["windowHeight"]; ok {
		if val, ok := v.(float64); ok {
			c.config.WindowHeight = int(val)
		}
	}
	if v, ok := settings["includePrereleasesForUpdates"]; ok {
		if val, ok := v.(bool); ok {
			c.config.IncludePrereleasesForUpdates = val
		}
	}
	return c.Save()
}

func (c *ConfigStore) GetWindowSize() (int, int) {
	return c.config.WindowWidth, c.config.WindowHeight
}
