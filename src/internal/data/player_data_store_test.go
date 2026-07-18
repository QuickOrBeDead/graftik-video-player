package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/QuickOrBeDead/graftik-video-player/internal/testutil"
)

func setupTestStore(t *testing.T) *PlayerDataStore {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	log := testutil.NopLogger{}
	store, err := NewPlayerDataStore(dir, dbPath, log)
	if err != nil {
		t.Fatalf("setupTestStore: NewPlayerDataStore failed: %v", err)
	}
	if err := store.Initialize(); err != nil {
		t.Fatalf("setupTestStore: Initialize failed: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func addTestPlaylist(t *testing.T, store *PlayerDataStore, id, name string) {
	t.Helper()
	if err := store.repo.AddPlaylist(id, name); err != nil {
		t.Fatalf("addTestPlaylist: %v", err)
	}
}

func addTestPlaylistItem(t *testing.T, store *PlayerDataStore, item PlaylistItemDto) {
	t.Helper()
	input := []struct {
		ID         string
		PlaylistID string
		Path       string
		Title      string
		OrderIndex float64
	}{
		{
			ID:         item.ID,
			PlaylistID: item.PlaylistID,
			Path:       item.Path,
			Title:      item.Title,
			OrderIndex: item.OrderIndex,
		},
	}
	store.repo.AddPlaylistItems(input)
}

func TestNewPlayerDataStore_NilLogger(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil logger")
		}
	}()
	NewPlayerDataStore(t.TempDir(), filepath.Join(t.TempDir(), "db.db"), nil)
}

func TestNewPlayerDataStore_Success(t *testing.T) {
	dir := t.TempDir()
	store, err := NewPlayerDataStore(dir, filepath.Join(dir, "db.db"), testutil.NopLogger{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	store.Close()
}

func TestInitialize(t *testing.T) {
	store := setupTestStore(t)
	var count int
	store.repo.db.QueryRow("SELECT COUNT(*) FROM playlists").Scan(&count)
	if count != 0 {
		t.Fatalf("expected 0 playlists, got %d", count)
	}
}

func TestClose(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewPlayerDataStore(dir, dbPath, testutil.NopLogger{})
	if err != nil {
		t.Fatalf("NewPlayerDataStore: %v", err)
	}
	if err := store.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestGetCurrentPlaylistID_Empty(t *testing.T) {
	store := setupTestStore(t)
	if id := store.GetCurrentPlaylistID(); id != "" {
		t.Fatalf("expected empty string, got %q", id)
	}
}

func TestSetCurrentPlaylistID(t *testing.T) {
	store := setupTestStore(t)
	if err := store.SetCurrentPlaylistID("abc-123"); err != nil {
		t.Fatalf("SetCurrentPlaylistID: %v", err)
	}
	if id := store.GetCurrentPlaylistID(); id != "abc-123" {
		t.Fatalf("expected %q, got %q", "abc-123", id)
	}
}

func TestGetPlaylistByID_Found(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "My Playlist")

	item := PlaylistItemDto{
		ID:         "i1",
		PlaylistID: "p1",
		Path:       "/videos/test.mp4",
		Title:      "test.mp4",
		OrderIndex: 1000,
	}
	addTestPlaylistItem(t, store, item)

	p, err := store.GetPlaylistByID("p1")
	if err != nil {
		t.Fatalf("GetPlaylistByID: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil playlist")
	}
	if p.ID != "p1" {
		t.Errorf("ID = %q, want %q", p.ID, "p1")
	}
	if p.Name != "My Playlist" {
		t.Errorf("Name = %q, want %q", p.Name, "My Playlist")
	}
	if len(p.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(p.Items))
	}
	if p.Items[0].ID != "i1" {
		t.Errorf("item ID = %q, want %q", p.Items[0].ID, "i1")
	}
}

func TestGetPlaylistByID_WithCurrentItemAndVolume(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "My Playlist")

	item1 := PlaylistItemDto{ID: "i1", PlaylistID: "p1", Path: "/a.mp4", Title: "a.mp4", OrderIndex: 1000}
	item2 := PlaylistItemDto{ID: "i2", PlaylistID: "p1", Path: "/b.mp4", Title: "b.mp4", OrderIndex: 2000}
	addTestPlaylistItem(t, store, item1)
	addTestPlaylistItem(t, store, item2)

	vol := 0.65
	store.repo.UpdatePlaylist("p1", map[string]any{
		"current_item":   "i2",
		"current_volume": vol,
	})

	p, err := store.GetPlaylistByID("p1")
	if err != nil {
		t.Fatalf("GetPlaylistByID: %v", err)
	}
	if p.CurrentItem == nil {
		t.Fatal("expected non-nil CurrentItem")
	}
	if *p.CurrentItem != "i2" {
		t.Errorf("CurrentItem = %q, want %q", *p.CurrentItem, "i2")
	}
	if p.CurrentVolume == nil {
		t.Fatal("expected non-nil CurrentVolume")
	}
	if *p.CurrentVolume != 0.65 {
		t.Errorf("CurrentVolume = %f, want 0.65", *p.CurrentVolume)
	}
	if p.CurrentPlaylistItem == nil {
		t.Fatal("expected non-nil CurrentPlaylistItem")
	}
	if p.CurrentPlaylistItem.ID != "i2" {
		t.Errorf("CurrentPlaylistItem.ID = %q, want %q", p.CurrentPlaylistItem.ID, "i2")
	}
	if p.CurrentPlaylistItem.Path != "/b.mp4" {
		t.Errorf("CurrentPlaylistItem.Path = %q, want %q", p.CurrentPlaylistItem.Path, "/b.mp4")
	}
}

func TestGetPlaylistByID_NotFound(t *testing.T) {
	store := setupTestStore(t)
	p, err := store.GetPlaylistByID("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Fatalf("expected nil playlist, got %+v", p)
	}
}

func TestGetPlaylistName_Found(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Cool Videos")

	name, err := store.GetPlaylistName("p1")
	if err != nil {
		t.Fatalf("GetPlaylistName: %v", err)
	}
	if name != "Cool Videos" {
		t.Errorf("name = %q, want %q", name, "Cool Videos")
	}
}

func TestGetPlaylistName_NotFound(t *testing.T) {
	store := setupTestStore(t)
	name, err := store.GetPlaylistName("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}
}

func TestGetPlaylists_Empty(t *testing.T) {
	store := setupTestStore(t)
	playlists, err := store.GetPlaylists()
	if err != nil {
		t.Fatalf("GetPlaylists: %v", err)
	}
	if len(playlists) != 0 {
		t.Fatalf("expected 0 playlists, got %d", len(playlists))
	}
}

func TestGetPlaylists_Multiple(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Bravo")
	addTestPlaylist(t, store, "p2", "Alpha")

	playlists, err := store.GetPlaylists()
	if err != nil {
		t.Fatalf("GetPlaylists: %v", err)
	}
	if len(playlists) != 2 {
		t.Fatalf("expected 2 playlists, got %d", len(playlists))
	}
	if playlists[0].Name != "Alpha" {
		t.Errorf("playlists[0].Name = %q, want %q", playlists[0].Name, "Alpha")
	}
	if playlists[1].Name != "Bravo" {
		t.Errorf("playlists[1].Name = %q, want %q", playlists[1].Name, "Bravo")
	}
}

func TestAddDefaultPlaylist_NoPlaylists(t *testing.T) {
	store := setupTestStore(t)
	if err := store.AddDefaultPlaylist(); err != nil {
		t.Fatalf("AddDefaultPlaylist: %v", err)
	}

	playlists, err := store.GetPlaylists()
	if err != nil {
		t.Fatalf("GetPlaylists: %v", err)
	}
	if len(playlists) != 1 {
		t.Fatalf("expected 1 playlist, got %d", len(playlists))
	}
	if playlists[0].Name != "default" {
		t.Errorf("name = %q, want %q", playlists[0].Name, "default")
	}
	if id := store.GetCurrentPlaylistID(); id != playlists[0].ID {
		t.Errorf("current playlist = %q, want %q", id, playlists[0].ID)
	}
}

func TestAddDefaultPlaylist_PlaylistsExist_NoCurrent(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Existing")

	if err := store.AddDefaultPlaylist(); err != nil {
		t.Fatalf("AddDefaultPlaylist: %v", err)
	}

	if id := store.GetCurrentPlaylistID(); id != "p1" {
		t.Errorf("current playlist = %q, want %q", id, "p1")
	}
}

func TestAddDefaultPlaylist_PlaylistsExist_HasCurrent(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Existing")
	store.SetCurrentPlaylistID("p1")

	if err := store.AddDefaultPlaylist(); err != nil {
		t.Fatalf("AddDefaultPlaylist: %v", err)
	}

	if id := store.GetCurrentPlaylistID(); id != "p1" {
		t.Errorf("current playlist = %q, want %q", id, "p1")
	}
}

func TestAddPlaylist(t *testing.T) {
	store := setupTestStore(t)
	p, err := store.AddPlaylist("New Playlist")
	if err != nil {
		t.Fatalf("AddPlaylist: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil playlist")
	}
	if p.Name != "New Playlist" {
		t.Errorf("Name = %q, want %q", p.Name, "New Playlist")
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestUpdatePlaylist(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Old Name")

	if err := store.UpdatePlaylist("p1", map[string]any{"name": "New Name"}); err != nil {
		t.Fatalf("UpdatePlaylist: %v", err)
	}

	name, _ := store.GetPlaylistName("p1")
	if name != "New Name" {
		t.Errorf("name = %q, want %q", name, "New Name")
	}
}

func TestGetPlaylistItem_Found(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Playlist")
	item := PlaylistItemDto{
		ID:         "i1",
		PlaylistID: "p1",
		Path:       "/videos/movie.mp4",
		Title:      "movie.mp4",
		OrderIndex: 1000,
	}
	addTestPlaylistItem(t, store, item)

	got := store.GetPlaylistItem("i1")
	if got == nil {
		t.Fatal("expected non-nil item")
	}
	if got.Path != "/videos/movie.mp4" {
		t.Errorf("Path = %q, want %q", got.Path, "/videos/movie.mp4")
	}
}

func TestGetPlaylistItem_NotFound(t *testing.T) {
	store := setupTestStore(t)
	got := store.GetPlaylistItem("nonexistent")
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestUpdatePlaylistItem(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Playlist")
	item := PlaylistItemDto{
		ID:         "i1",
		PlaylistID: "p1",
		Path:       "/videos/old.mp4",
		Title:      "old.mp4",
		OrderIndex: 1000,
	}
	addTestPlaylistItem(t, store, item)

	if err := store.UpdatePlaylistItem("i1", map[string]any{"title": "updated.mp4"}); err != nil {
		t.Fatalf("UpdatePlaylistItem: %v", err)
	}

	got := store.GetPlaylistItem("i1")
	if got.Title != "updated.mp4" {
		t.Errorf("Title = %q, want %q", got.Title, "updated.mp4")
	}
}

func TestDeletePlaylist_IsCurrent(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Playlist")
	store.SetCurrentPlaylistID("p1")

	if err := store.DeletePlaylist("p1"); err != nil {
		t.Fatalf("DeletePlaylist: %v", err)
	}

	if id := store.GetCurrentPlaylistID(); id != "" {
		t.Errorf("current playlist = %q, want empty", id)
	}

	p, _ := store.GetPlaylistByID("p1")
	if p != nil {
		t.Error("expected nil playlist after deletion")
	}
}

func TestDeletePlaylist_NotCurrent(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Playlist 1")
	addTestPlaylist(t, store, "p2", "Playlist 2")
	store.SetCurrentPlaylistID("p1")

	if err := store.DeletePlaylist("p2"); err != nil {
		t.Fatalf("DeletePlaylist: %v", err)
	}

	if id := store.GetCurrentPlaylistID(); id != "p1" {
		t.Errorf("current playlist = %q, want %q", id, "p1")
	}
}

func TestDeletePlaylistItem(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Playlist")
	item := PlaylistItemDto{
		ID:         "i1",
		PlaylistID: "p1",
		Path:       "/videos/test.mp4",
		Title:      "test.mp4",
		OrderIndex: 1000,
	}
	addTestPlaylistItem(t, store, item)

	if err := store.DeletePlaylistItem("i1"); err != nil {
		t.Fatalf("DeletePlaylistItem: %v", err)
	}

	if got := store.GetPlaylistItem("i1"); got != nil {
		t.Error("expected nil after deletion")
	}
}

func TestInitNewPlaylistItems(t *testing.T) {
	store := setupTestStore(t)
	paths := []string{
		"/home/user/Videos/movie1.mp4",
		"/home/user/Videos/movie2.mkv",
		"/another/path/sub/video.avi",
	}

	items := store.InitNewPlaylistItems(paths)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	for i, item := range items {
		if item.ID == "" {
			t.Errorf("items[%d].ID is empty", i)
		}
		if item.Path != paths[i] {
			t.Errorf("items[%d].Path = %q, want %q", i, item.Path, paths[i])
		}
		if item.OrderIndex != 0 {
			t.Errorf("items[%d].OrderIndex = %f, want 0", i, item.OrderIndex)
		}
	}

	if items[0].Title != "movie1.mp4" {
		t.Errorf("items[0].Title = %q, want %q", items[0].Title, "movie1.mp4")
	}
	if items[1].Title != "movie2.mkv" {
		t.Errorf("items[1].Title = %q, want %q", items[1].Title, "movie2.mkv")
	}
	if items[2].Title != "video.avi" {
		t.Errorf("items[2].Title = %q, want %q", items[2].Title, "video.avi")
	}
}

func TestInitNewPlaylistItems_Empty(t *testing.T) {
	store := setupTestStore(t)
	items := store.InitNewPlaylistItems([]string{})
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestInitNewPlaylistItems_UUIDUniqueness(t *testing.T) {
	store := setupTestStore(t)
	items := store.InitNewPlaylistItems([]string{"/a.mp4", "/b.mp4", "/c.mp4"})
	seen := make(map[string]bool)
	for _, item := range items {
		if seen[item.ID] {
			t.Fatalf("duplicate ID: %s", item.ID)
		}
		seen[item.ID] = true
	}
}

func TestAddPlaylistItems(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Playlist")

	items := store.InitNewPlaylistItems([]string{"/a.mp4", "/b.mp4"})
	for i := range items {
		items[i].PlaylistID = "p1"
		items[i].OrderIndex = float64(i+1) * 1000
	}
	store.AddPlaylistItems(items)

	p, err := store.GetPlaylistByID("p1")
	if err != nil {
		t.Fatalf("GetPlaylistByID: %v", err)
	}
	if len(p.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(p.Items))
	}
}

func TestRebalancePlaylistOrder(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Playlist")

	item1 := PlaylistItemDto{ID: "i1", PlaylistID: "p1", Path: "/a.mp4", Title: "a.mp4", OrderIndex: 5000}
	item2 := PlaylistItemDto{ID: "i2", PlaylistID: "p1", Path: "/b.mp4", Title: "b.mp4", OrderIndex: 30000}
	item3 := PlaylistItemDto{ID: "i3", PlaylistID: "p1", Path: "/c.mp4", Title: "c.mp4", OrderIndex: 10000}
	addTestPlaylistItem(t, store, item1)
	addTestPlaylistItem(t, store, item2)
	addTestPlaylistItem(t, store, item3)

	if err := store.RebalancePlaylistOrder("p1"); err != nil {
		t.Fatalf("RebalancePlaylistOrder: %v", err)
	}

	p, err := store.GetPlaylistByID("p1")
	if err != nil {
		t.Fatalf("GetPlaylistByID: %v", err)
	}
	if len(p.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(p.Items))
	}
	expectedOrder := []float64{1000, 2000, 3000}
	for i, item := range p.Items {
		if item.OrderIndex != expectedOrder[i] {
			t.Errorf("items[%d].OrderIndex = %f, want %f", i, item.OrderIndex, expectedOrder[i])
		}
	}
}

func TestGetPreferences_Default(t *testing.T) {
	store := setupTestStore(t)
	preferences := store.GetPreferences()
	if preferences == nil {
		t.Fatal("expected non-nil preferences")
	}
	if preferences.VolumeLevel != 1.0 {
		t.Errorf("VolumeLevel = %f, want 1.0", preferences.VolumeLevel)
	}
	if preferences.PlaybackRate != 1.0 {
		t.Errorf("PlaybackRate = %f, want 1.0", preferences.PlaybackRate)
	}
	if !preferences.SidebarVisible {
		t.Error("SidebarVisible = false, want true")
	}
	if preferences.SidebarWidth != 300 {
		t.Errorf("SidebarWidth = %d, want 300", preferences.SidebarWidth)
	}
}

func TestUpdateSettings(t *testing.T) {
	store := setupTestStore(t)

	err := store.UpdateSettings(map[string]any{
		"shuffle":      true,
		"repeatMode":   float64(2),
		"volumeLevel":  0.75,
		"playbackRate": 1.5,
	})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	preferences := store.GetPreferences()
	if !preferences.Shuffle {
		t.Error("Shuffle = false, want true")
	}
	if preferences.RepeatMode != 2 {
		t.Errorf("RepeatMode = %d, want 2", preferences.RepeatMode)
	}
	if preferences.VolumeLevel != 0.75 {
		t.Errorf("VolumeLevel = %f, want 0.75", preferences.VolumeLevel)
	}
	if preferences.PlaybackRate != 1.5 {
		t.Errorf("PlaybackRate = %f, want 1.5", preferences.PlaybackRate)
	}
}

func TestUpdateSettings_Persistence(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	log := testutil.NopLogger{}

	store1, err := NewPlayerDataStore(dir, dbPath, log)
	if err != nil {
		t.Fatalf("NewPlayerDataStore: %v", err)
	}
	store1.Initialize()
	store1.UpdateSettings(map[string]any{"volumeLevel": 0.42})
	store1.Close()

	store2, err := NewPlayerDataStore(dir, dbPath, log)
	if err != nil {
		t.Fatalf("NewPlayerDataStore (reopen): %v", err)
	}
	store2.Initialize()
	defer store2.Close()

	preferences := store2.GetPreferences()
	if preferences.VolumeLevel != 0.42 {
		t.Errorf("VolumeLevel = %f, want 0.42", preferences.VolumeLevel)
	}
}

func TestConfigFileCreated(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewPlayerDataStore(dir, dbPath, testutil.NopLogger{})
	if err != nil {
		t.Fatalf("NewPlayerDataStore: %v", err)
	}
	store.Initialize()
	store.SetCurrentPlaylistID("test-id")
	store.Close()

	configPath := filepath.Join(dir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config.json was not created")
	}
}

func TestAddDefaultPlaylist_NilCurrentAfterDelete(t *testing.T) {
	store := setupTestStore(t)
	addTestPlaylist(t, store, "p1", "Playlist")
	store.SetCurrentPlaylistID("p1")

	if err := store.DeletePlaylist("p1"); err != nil {
		t.Fatalf("DeletePlaylist: %v", err)
	}

	if id := store.GetCurrentPlaylistID(); id != "" {
		t.Errorf("expected empty current playlist after deleting current, got %q", id)
	}

	store.AddDefaultPlaylist()

	playlists, err := store.GetPlaylists()
	if err != nil {
		t.Fatalf("GetPlaylists: %v", err)
	}
	if len(playlists) != 1 {
		t.Fatalf("expected 1 playlist after AddDefaultPlaylist, got %d", len(playlists))
	}
	if playlists[0].Name != "default" {
		t.Errorf("name = %q, want %q", playlists[0].Name, "default")
	}
}
