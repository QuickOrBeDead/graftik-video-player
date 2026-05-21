/// <reference types="vite/client" />

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetVideoServerPort: () => Promise<number>
        }
      }
      internal: {
        PlayerService: {
          GetCurrentPlaylist: () => Promise<any>
          GetPlaylists: () => Promise<any>
          SelectPlaylist: (id: string) => Promise<any>
          AddPlaylist: (name: string) => Promise<any>
          UpdatePlaylistName: (id: string, name: string) => Promise<any>
          UpdatePlaylist: (id: string, data: any) => Promise<any>
          DeletePlaylist: (id: string) => Promise<any>
          AddPlaylistItems: (items: any[]) => Promise<any>
          UpdatePlaylistItem: (id: string, data: any) => Promise<any>
          DeletePlaylistItem: (id: string) => Promise<any>
          GetPlaylist: (id: string) => Promise<any>
          GetPlaylistName: (id: string) => Promise<any>
          GetPlaylistItemVideoMetadata: (playlistId: string, itemId: string, path: string) => Promise<any>
          RebalancePlaylistOrder: (id: string) => Promise<any>
          OpenContainingFolder: (path: string) => Promise<any>
          InitNewPlaylistItems: (filePaths: string[]) => Promise<any>
        }
      }
    }
    runtime: {
      EventsOn: (event: string, callback: (...args: any[]) => void) => () => void
      EventsOff: (event: string, ...args: any[]) => void
      EventsEmit: (event: string, ...data: any) => void
      WindowClose: () => void
      WindowSetTitle: (title: string) => void
    }
  }
}

export {}
