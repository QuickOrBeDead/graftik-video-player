package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AppConfig struct {
	CurrentPlaylist string `json:"currentPlaylist"`
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
			c.config = AppConfig{}
			return nil
		}
		return fmt.Errorf("failed to read config: %w", err)
	}
	return json.Unmarshal(data, &c.config)
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
