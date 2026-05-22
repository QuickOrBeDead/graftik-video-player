/// <reference types="vite/client" />

interface PluginInstallResult {
  id: string
  name: string
  version: string
  status: string
  menu: any[]
  ui?: string
}

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetVideoServerPort: () => Promise<number>
          GetPlugins: () => Promise<any>
          ExecutePluginAction: (pluginId: string, action: string, argsJson: string) => Promise<any>
          PickDirectory: () => Promise<string>
          PickPluginFile: () => Promise<string>
          GetPluginFile: (pluginId: string, fileName: string) => Promise<string>
          InstallPluginFromFile: (filePath: string) => Promise<PluginInstallResult>
          InstallPluginFromURL: (url: string) => Promise<PluginInstallResult>
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
