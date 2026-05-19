export interface PlayerState {
    isPlaying: boolean
    isMuted: boolean
    isFullScreen: boolean
    volumeLevel: number
    shuffle: boolean
    repeat: RepeatMode
    playbackRate: number
    controlsVisible: boolean
    currentTime: number
    seekTime: number
    duration: number,
    sidebarVisible: boolean,
    pictureInPicture: boolean,
    fullScreen: boolean,
    videoSrc?: string | undefined,
    progressBarHoverTime?: number
    showProgressBarHoverPreview: boolean,
    isSidebarResizing: boolean,
    sidebarWidth: number
}

export enum RepeatMode {
    Off = 0,
    All = 1,
    One = 2
}

export interface Playlist {
  id: string
  name: string
  viewMode?: PlaylistViewMode
  sortInfo?: PlaylistSortInfo
  shuffle?: boolean
  repeat?: number
  showOnlyUnwatched?: boolean
  currentItem?: string
  currentTime?: number
  currentVolume?: number
  playbackRate?: number
  items: PlaylistItem[]
  currentPlaylistItem?: PlaylistItem
  shuffledDeck: PlaylistItem[]
}

export interface PlaylistItem {
  id: string
  playlistId: string
  path: string
  title: string
  isPlaying?: boolean
  thumbnailImage?: string
  currentTime?: number
  duration?: number
  progressPercent?: number,
  lastWatched?: Date,
  orderIndex: number
}

export enum PlaylistViewMode {
  Detailed = 0,
  Simple = 1
}

export enum PlaylistSortInfo {
  Default = 0,
  NameAsc = 1,
  NameDesc = 2,
  DurationAsc = 3,
  DurationDesc = 4,
  Newest = 5,
  Oldest = 6
}

export interface VideoMetadata {
  duration: number
  lastModified: number
  fileSize: number
  thumbnail: string
}
