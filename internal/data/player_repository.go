package data

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type PlayerRepository struct {
	db *sql.DB
}

func NewPlayerRepository(dbPath string) (*PlayerRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)

	return &PlayerRepository{db: db}, nil
}

func (r *PlayerRepository) InitializeDB() error {
	return RunMigrations(r.db)
}

func (r *PlayerRepository) Close() error {
	if r.db == nil {
		return nil
	}
	r.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return r.db.Close()
}

func (r *PlayerRepository) GetPlaylist(id string) (*PlaylistDto, error) {
	// Get playlist
	row := r.db.QueryRow(`SELECT id, name, shuffle, repeat, current_item, current_volume FROM playlists WHERE id = ?`, id)
	var p PlaylistDto
	var shuffle int
	var currentItem sql.NullString
	var currentVolume sql.NullFloat64
	err := row.Scan(&p.ID, &p.Name, &shuffle, &p.Repeat, &currentItem, &currentVolume)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query playlist: %w", err)
	}
	p.Shuffle = shuffle != 0
	if currentItem.Valid {
		p.CurrentItem = &currentItem.String
	}
	if currentVolume.Valid {
		p.CurrentVolume = &currentVolume.Float64
	}

	// Rebalance order first
	r.rebalancePlaylistOrder(id)

	// Get items
	items, err := r.getPlaylistItems(id)
	if err != nil {
		return nil, err
	}
	p.Items = items

	// Get current item
	if p.CurrentItem != nil {
		for _, item := range items {
			if item.ID == *p.CurrentItem {
				ci := item
				p.CurrentPlaylistItem = &ci
				break
			}
		}
	}

	return &p, nil
}

func (r *PlayerRepository) getPlaylistItems(playlistID string) ([]PlaylistItemDto, error) {
	rows, err := r.db.Query(`
		SELECT id, playlist_id, path, title, is_playing, elapsed_time, duration, progress_percent, last_watched, order_index
		FROM playlist_items
		WHERE playlist_id = ?
		ORDER BY order_index ASC
	`, playlistID)
	if err != nil {
		return nil, fmt.Errorf("failed to query playlist items: %w", err)
	}
	defer rows.Close()

	var items []PlaylistItemDto
	for rows.Next() {
		var item PlaylistItemDto
		var isPlaying int
		var elapsedTime, duration, progressPercent sql.NullFloat64
		var lastWatched sql.NullInt64
		err := rows.Scan(&item.ID, &item.PlaylistID, &item.Path, &item.Title,
			&isPlaying, &elapsedTime, &duration, &progressPercent, &lastWatched, &item.OrderIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to scan playlist item: %w", err)
		}
		isPlayingBool := isPlaying != 0
		item.IsPlaying = &isPlayingBool
		if elapsedTime.Valid {
			item.ElapsedTime = &elapsedTime.Float64
		}
		if duration.Valid {
			item.Duration = &duration.Float64
		}
		if progressPercent.Valid {
			item.ProgressPercent = &progressPercent.Float64
		}
		if lastWatched.Valid {
			item.LastWatched = &lastWatched.Int64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PlayerRepository) GetPlaylistProjection(id string, columns []string) (map[string]any, error) {
	query := fmt.Sprintf("SELECT %s FROM playlists WHERE id = ?", strings.Join(columns, ", "))
	row := r.db.QueryRow(query, id)

	values := make([]any, len(columns))
	ptrs := make([]any, len(columns))
	for i := range values {
		ptrs[i] = &values[i]
	}

	if err := row.Scan(ptrs...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query playlist projection: %w", err)
	}

	result := make(map[string]any)
	for i, col := range columns {
		result[col] = values[i]
	}
	return result, nil
}

func (r *PlayerRepository) GetPlaylistItemByID(id string) *PlaylistItemDto {
	row := r.db.QueryRow(`SELECT id, playlist_id, path, title, is_playing, elapsed_time, duration, progress_percent, last_watched, order_index FROM playlist_items WHERE id = ?`, id)
	var item PlaylistItemDto
	var isPlaying int
	var elapsedTime, duration, progressPercent sql.NullFloat64
	var lastWatched sql.NullInt64
	err := row.Scan(&item.ID, &item.PlaylistID, &item.Path, &item.Title,
		&isPlaying, &elapsedTime, &duration, &progressPercent, &lastWatched, &item.OrderIndex)
	if err != nil {
		return nil
	}
	isPlayingBool := isPlaying != 0
	item.IsPlaying = &isPlayingBool
	if elapsedTime.Valid {
		item.ElapsedTime = &elapsedTime.Float64
	}
	if duration.Valid {
		item.Duration = &duration.Float64
	}
	if progressPercent.Valid {
		item.ProgressPercent = &progressPercent.Float64
	}
	if lastWatched.Valid {
		item.LastWatched = &lastWatched.Int64
	}
	return &item
}

func (r *PlayerRepository) GetPlaylists() ([]PlaylistListItem, error) {
	rows, err := r.db.Query("SELECT id, name FROM playlists ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query playlists: %w", err)
	}
	defer rows.Close()

	var items []PlaylistListItem
	for rows.Next() {
		var item PlaylistListItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("failed to scan playlist: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PlayerRepository) HasAnyPlaylist() (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM playlists").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to count playlists: %w", err)
	}
	return count > 0, nil
}

func (r *PlayerRepository) AddPlaylist(id, name string) error {
	_, err := r.db.Exec("INSERT INTO playlists (id, name) VALUES (?, ?)", id, name)
	return err
}

func (r *PlayerRepository) updateRecord(table, id string, data map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	var setClauses []string
	var args []any
	for k, v := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", k))
		args = append(args, v)
	}
	args = append(args, id)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", table, strings.Join(setClauses, ", "))
	_, err := r.db.Exec(query, args...)
	return err
}

func (r *PlayerRepository) UpdatePlaylist(id string, data map[string]any) error {
	return r.updateRecord("playlists", id, data)
}

func (r *PlayerRepository) DeletePlaylist(id string) error {
	_, err := r.db.Exec("DELETE FROM playlists WHERE id = ?", id)
	return err
}

func (r *PlayerRepository) AddPlaylistItems(items []struct {
	ID         string
	PlaylistID string
	Path       string
	Title      string
	OrderIndex float64
}) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO playlist_items (id, playlist_id, path, title, order_index) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		if _, err := stmt.Exec(item.ID, item.PlaylistID, item.Path, item.Title, item.OrderIndex); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PlayerRepository) UpdatePlaylistItem(id string, data map[string]any) error {
	return r.updateRecord("playlist_items", id, data)
}

func (r *PlayerRepository) DeletePlaylistItem(id string) error {
	_, err := r.db.Exec("DELETE FROM playlist_items WHERE id = ?", id)
	return err
}

func (r *PlayerRepository) rebalancePlaylistOrder(playlistID string) error {
	_, err := r.db.Exec(`
		WITH reordered AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY order_index) AS pos
			FROM playlist_items
			WHERE playlist_id = ?
		)
		UPDATE playlist_items
		SET order_index = (SELECT pos * 1000 FROM reordered WHERE reordered.id = playlist_items.id)
		WHERE playlist_id = ?
	`, playlistID, playlistID)

	return err
}

func (r *PlayerRepository) RebalancePlaylistOrder(playlistID string) error {
	return r.rebalancePlaylistOrder(playlistID)
}
