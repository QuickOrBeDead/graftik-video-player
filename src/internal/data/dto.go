package data

type PlaylistDto struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Shuffle             bool              `json:"shuffle"`
	Repeat              int               `json:"repeat"`
	CurrentItem         *string           `json:"currentItem"`
	CurrentPlaylistItem *PlaylistItemDto  `json:"currentPlaylistItem"`
	CurrentVolume       *float64          `json:"currentVolume"`
	Items               []PlaylistItemDto `json:"items"`
}

type PlaylistItemDto struct {
	ID              string   `json:"id"`
	PlaylistID      string   `json:"playlistId"`
	Path            string   `json:"path"`
	Title           string   `json:"title"`
	IsPlaying       *bool    `json:"isPlaying"`
	ElapsedTime     *float64 `json:"elapsedTime"`
	Duration        *float64 `json:"duration"`
	ProgressPercent *float64 `json:"progressPercent"`
	LastWatched     *int64   `json:"lastWatched"`
	OrderIndex      float64  `json:"orderIndex"`
}

type PlaylistListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type VideoMetadata struct {
	Duration     float64 `json:"duration"`
	LastModified float64 `json:"lastModified"`
	FileSize     float64 `json:"fileSize"`
	Thumbnail    string  `json:"thumbnail"`
}

type StreamInfo struct {
	Container   string `json:"container"`
	VideoCodec  string `json:"videoCodec"`
	AudioCodec  string `json:"audioCodec"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Action      string `json:"action"`
	ActionLabel string `json:"actionLabel"`
	HWEncoder   string `json:"hwEncoder,omitempty"`
}

type StreamURLResult struct {
	URL      string `json:"url"`
	StreamID string `json:"streamId,omitempty"`
}
