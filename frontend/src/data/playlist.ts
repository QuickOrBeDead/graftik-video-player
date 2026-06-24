export interface PlaylistDto {
  id: string
  playlistId: string
  name: string
  shuffle?: boolean
  repeat?: number
  currentItem?: string
  currentVolume: number
  items: PlaylistItemDto[],
  currentPlaylistItem?: PlaylistItemDto
}

export interface PlaylistItemDto {
  id: string
  playlistId: string
  path: string
  title: string
  isPlaying?: boolean
  elapsedTime?: number
  duration?: number
  progressPercent?: number
  lastWatched?: Date,
  orderIndex: number
}
