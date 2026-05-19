export interface PlaylistDto {
  id: string
  name: string
  shuffle?: boolean
  repeat?: number
  currentItem: string | null
  currentPlaylistItem: PlaylistItemDto | null
  currentVolume: number | null
  items: PlaylistItemDto[]
}

export interface PlaylistItemDto {
  id: string
  playlistId: string
  path: string
  title: string
  isPlaying: boolean | null
  currentTime: number | null
  duration: number | null
  progressPercent: number | null
  lastWatched: number | null
  orderIndex: number
}

export interface PlaylistListItemDto {
  id: string
  name: string
}
