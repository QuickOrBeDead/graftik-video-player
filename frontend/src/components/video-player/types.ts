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
    playlistItemId?: string,
    seekTime: number
    duration: number,
    sidebarVisible: boolean,
    pictureInPicture: boolean,
    fullScreen: boolean,
    videoSrc?: string | undefined,
    progressBarHoverTime?: number
    showProgressBarHoverPreview: boolean,
    isSidebarResizing: boolean,
    sidebarWidth: number,
    streamId: string,
    shouldAutoplay: boolean
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
  elapsedTime?: number
  duration?: number
  progressPercent?: number,
  lastWatched?: Date,
  orderIndex: number
  streamInfo?: StreamInfo
}

export interface StreamInfo {
  container: string
  videoCodec: string
  audioCodec: string
  width: number
  height: number
  action: string
  actionLabel: string
  hwEncoder?: string
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

export interface StreamURLResult {
  url: string
  streamId?: string
}

export interface AppPreferences {
  shuffle: boolean
  repeatMode: number
  volumeLevel: number
  playbackRate: number
  sidebarVisible: boolean
  sidebarWidth: number
  windowWidth: number
  windowHeight: number
  isPlaying: boolean
  lastPlayedItem: string
}
