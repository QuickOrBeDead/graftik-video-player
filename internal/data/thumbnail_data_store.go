package data

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

type ThumbnailDataStore struct {
	cacheFolder string
}

func NewThumbnailDataStore(userDataPath string) *ThumbnailDataStore {
	folder := filepath.Join(userDataPath, "thumbnails")
	if err := os.MkdirAll(folder, 0755); err != nil {
		panic(fmt.Sprintf("failed to create thumbnail cache dir: %v", err))
	}
	return &ThumbnailDataStore{cacheFolder: folder}
}

func (s *ThumbnailDataStore) ensurePlaylistFolder(playlistID string) (string, error) {
	folder := filepath.Join(s.cacheFolder, playlistID)
	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", fmt.Errorf("failed to create playlist thumbnail folder: %w", err)
	}
	return folder, nil
}

func thumbnailPath(playlistPath, itemID, fileHash string) string {
	return filepath.Join(playlistPath, fmt.Sprintf("%s-%s.jpeg", itemID, fileHash))
}

func (s *ThumbnailDataStore) CalculateFileHash(filePath string, fileSize int64, lastModified int64) string {
	input := fmt.Sprintf("%s-%d-%d", filePath, fileSize, lastModified)
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)
}

func (s *ThumbnailDataStore) GetThumbnail(playlistID, itemID, fileHash string) (string, error) {
	playlistPath, err := s.ensurePlaylistFolder(playlistID)
	if err != nil {
		return "", err
	}

	thumbPath := thumbnailPath(playlistPath, itemID, fileHash)
	stats, err := os.Stat(thumbPath)
	if err != nil || stats.IsDir() {
		return "", nil
	}

	data, err := os.ReadFile(thumbPath)
	if err != nil {
		return "", nil
	}

	return fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(data)), nil
}

func (s *ThumbnailDataStore) SetThumbnail(playlistID, itemID, fileHash string, data []byte) error {
	playlistPath, err := s.ensurePlaylistFolder(playlistID)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(playlistPath)
	if err == nil {
		prefix := fmt.Sprintf("%s-", itemID)
		for _, entry := range entries {
			if !entry.IsDir() && len(entry.Name()) > len(prefix) && entry.Name()[:len(prefix)] == prefix && filepath.Ext(entry.Name()) == ".jpeg" {
				os.Remove(filepath.Join(playlistPath, entry.Name()))
			}
		}
	}

	newPath := thumbnailPath(playlistPath, itemID, fileHash)
	return os.WriteFile(newPath, data, 0644)
}
