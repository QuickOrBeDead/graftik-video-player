package data

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupThumbnailStore(t *testing.T) (*ThumbnailDataStore, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := NewThumbnailDataStore(dir)
	if err != nil {
		t.Fatalf("setupThumbnailStore: %v", err)
	}
	return store, dir
}

func TestNewThumbnailDataStore_Success(t *testing.T) {
	dir := t.TempDir()
	store, err := NewThumbnailDataStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	thumbDir := filepath.Join(dir, "thumbnails")
	info, err := os.Stat(thumbDir)
	if err != nil {
		t.Fatalf("thumbnails directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("thumbnails path is not a directory")
	}

	_ = store
}

func TestNewThumbnailDataStore_InvalidPath(t *testing.T) {
	_, err := NewThumbnailDataStore("/nonexistent/deeply/nested/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestCalculateFileHash_Deterministic(t *testing.T) {
	store, _ := setupThumbnailStore(t)

	h1 := store.CalculateFileHash("/videos/test.mp4", 1024, 1700000000)
	h2 := store.CalculateFileHash("/videos/test.mp4", 1024, 1700000000)

	if h1 != h2 {
		t.Errorf("same inputs produced different hashes: %q vs %q", h1, h2)
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char hex hash, got length %d", len(h1))
	}
}

func TestCalculateFileHash_DifferentInputs(t *testing.T) {
	store, _ := setupThumbnailStore(t)

	h1 := store.CalculateFileHash("/a.mp4", 100, 200)
	h2 := store.CalculateFileHash("/b.mp4", 100, 200)
	h3 := store.CalculateFileHash("/a.mp4", 200, 200)
	h4 := store.CalculateFileHash("/a.mp4", 100, 300)

	if h1 == h2 {
		t.Errorf("different paths produced same hash")
	}
	if h1 == h3 {
		t.Errorf("different sizes produced same hash")
	}
	if h1 == h4 {
		t.Errorf("different modified times produced same hash")
	}
}

func TestGetThumbnail_NotFound(t *testing.T) {
	store, _ := setupThumbnailStore(t)

	result, err := store.GetThumbnail("playlist1", "item1", "hash1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestGetThumbnail_IsDirectory(t *testing.T) {
	store, _ := setupThumbnailStore(t)

	// Create the playlist folder
	playlistPath, err := store.ensurePlaylistFolder("playlist1")
	if err != nil {
		t.Fatalf("ensurePlaylistFolder: %v", err)
	}

	// Create a directory named like a thumbnail file (with the itemID-hash prefix)
	dirName := thumbnailPath(playlistPath, "item1", "hash1")
	if err := os.MkdirAll(dirName, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	result, err := store.GetThumbnail("playlist1", "item1", "hash1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string for directory, got %q", result)
	}
}

func TestGetThumbnail_Success(t *testing.T) {
	store, _ := setupThumbnailStore(t)

	imgData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10} // fake JPEG header
	if err := store.SetThumbnail("playlist1", "item1", "hash1", imgData); err != nil {
		t.Fatalf("SetThumbnail: %v", err)
	}

	result, err := store.GetThumbnail("playlist1", "item1", "hash1")
	if err != nil {
		t.Fatalf("GetThumbnail: %v", err)
	}

	prefix := "data:image/jpeg;base64,"
	if !strings.HasPrefix(result, prefix) {
		t.Fatalf("expected prefix %q, got %q", prefix, result)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(result, prefix))
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	if string(decoded) != string(imgData) {
		t.Errorf("decoded data mismatch: got %v, want %v", decoded, imgData)
	}
}

func TestSetThumbnail_CreatesFile(t *testing.T) {
	store, _ := setupThumbnailStore(t)

	imgData := []byte("fake-image-bytes")
	if err := store.SetThumbnail("pl1", "item1", "h1", imgData); err != nil {
		t.Fatalf("SetThumbnail: %v", err)
	}

	playlistPath, _ := store.ensurePlaylistFolder("pl1")
	thumbFile := thumbnailPath(playlistPath, "item1", "h1")

	content, err := os.ReadFile(thumbFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != string(imgData) {
		t.Errorf("file content mismatch: got %q, want %q", content, imgData)
	}
}

func TestSetThumbnail_CleansUpOldHashes(t *testing.T) {
	store, _ := setupThumbnailStore(t)

	store.SetThumbnail("pl1", "item1", "hashA", []byte("imageA"))
	store.SetThumbnail("pl1", "item1", "hashB", []byte("imageB"))

	playlistPath, _ := store.ensurePlaylistFolder("pl1")
	entries, err := os.ReadDir(playlistPath)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	matches := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "item1-") && filepath.Ext(e.Name()) == ".jpeg" {
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("expected 1 thumbnail file for item1, got %d", matches)
	}

	content, err := os.ReadFile(thumbnailPath(playlistPath, "item1", "hashB"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "imageB" {
		t.Errorf("expected hashB content, got %q", content)
	}
}

func TestSetThumbnail_CreatesPlaylistFolder(t *testing.T) {
	store, _ := setupThumbnailStore(t)

	if err := store.SetThumbnail("newplaylist", "item1", "h1", []byte("data")); err != nil {
		t.Fatalf("SetThumbnail: %v", err)
	}

	folder := filepath.Join(store.cacheFolder, "newplaylist")
	info, err := os.Stat(folder)
	if err != nil {
		t.Fatalf("playlist folder not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("playlist path is not a directory")
	}
}

func TestSetThumbnail_MultipleItems(t *testing.T) {
	store, _ := setupThumbnailStore(t)

	store.SetThumbnail("pl1", "itemA", "hash1", []byte("thumbA"))
	store.SetThumbnail("pl1", "itemB", "hash2", []byte("thumbB"))

	resultA, err := store.GetThumbnail("pl1", "itemA", "hash1")
	if err != nil {
		t.Fatalf("GetThumbnail itemA: %v", err)
	}
	resultB, err := store.GetThumbnail("pl1", "itemB", "hash2")
	if err != nil {
		t.Fatalf("GetThumbnail itemB: %v", err)
	}

	if resultA == "" {
		t.Error("expected non-empty result for itemA")
	}
	if resultB == "" {
		t.Error("expected non-empty result for itemB")
	}
	if resultA == resultB {
		t.Error("itemA and itemB thumbnails should differ")
	}
}
