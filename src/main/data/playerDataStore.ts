import { PlaylistDto, PlaylistItemDto, PlaylistListItemDto } from './playlistDto.js'
import { AppState } from './appState.js'
import Store from 'electron-store'
import path from 'path'
import { v4 as newUUID } from 'uuid'
import { PlayerRepository } from './playerRepository.js'

export class PlayerDataStore {
  playerRepository: PlayerRepository
  store: Store<AppState>

  constructor(userDataPath: string, migrationsPath: string) {
    this.playerRepository = new PlayerRepository(path.join(userDataPath, 'player.db'))
    this.initializeDatabase(migrationsPath)
    this.store = new Store<AppState>()
  }

  private initializeDatabase(migrationsPath: string): void {
    this.playerRepository.initializeDb(migrationsPath)
  }

  public GetCurrentPlaylistId(): string | undefined {
    return this.store.get('currentPlaylist')
  }

  public SetCurrentPlaylistId(id: string): void {
    this.store.set('currentPlaylist', id)
  }

  public async GetPlaylistById(id: string): Promise<PlaylistDto | undefined> {
    this.playerRepository.rebalancePlaylistOrder(id)
    const playlist = await this.playerRepository.getPlaylist(id)
    return playlist
  }

  public async GetPlaylistName(id: string): Promise<string | undefined> {
    const row = await this.playerRepository.getPlaylistProjection(id, { name: true })
    if (row === undefined) {
      return undefined
    }

    return row.name
  }

  public async GetPlaylists(): Promise<PlaylistListItemDto[]> {
    const items = await this.playerRepository.getPlaylists({
      id: true,
      name: true
    })

    return items.map(x => ({
      id: x.id,
      name: x.name
    }))
  }

  public async AddDefaultPlaylist() {
    if (!(await this.playerRepository.hasAnyPlaylist())) {
      const data = await this.AddPlaylist('default')
      this.SetCurrentPlaylistId(data.id)
    }
  }

  public async AddPlaylist(name: string) {
    const data = {
      id: newUUID(),
      name: name
    }
    await this.playerRepository.addPlaylist(data)

    return data
  }

  public UpdatePlaylistItem(id: string, data: { [x: string]: any }) {
    return this.playerRepository.updatePlaylistItem(id, data)
  }

  public UpdatePlaylistName(item: { id: string; name: string }): Promise<void> {
    return this.playerRepository.updatePlaylist(item.id, { name: item.name })
  }

  public UpdatePlaylist(id: string, data: { [x: string]: any }) {
    return this.playerRepository.updatePlaylist(id, data)
  }

  public InitNewPlaylistItems(filePaths: string[]): PlaylistItemDto[] {
    const result: PlaylistItemDto[] = []

    for (let i = 0; i < filePaths.length; i++) {
      const filePath = filePaths[i]
      result.push({
        id: newUUID(),
        path: filePath,
        title: path.basename(filePath),
        orderIndex: 0,
        isPlaying: null,
        currentTime: null,
        duration: null,
        progressPercent: null,
        lastWatched: null,
        playlistId: ''
      })
    }

    return result
  }

  public AddPlaylistItems(items: PlaylistItemDto[]) {
    this.playerRepository.addPlaylistItems(items.map(x => ({
      id: x.id,
      playlistId: x.playlistId,
      path: x.path,
      title: x.title,
      orderIndex: x.orderIndex
    })))
  }

  public DeletePlaylist(id: string) {
    return this.playerRepository.deletePlaylist(id)
  }

  public DeletePlaylistItem(id: string) {
    return this.playerRepository.deletePlaylistItem(id)
  }

  public UpdatePlaylistItemOrder(id: string, orderIndex: number) {
    return this.playerRepository.updatePlaylistItem(id, { orderIndex })
  }

  public RebalancePlaylistOrder(id: string) {
    this.playerRepository.rebalancePlaylistOrder(id)
  }
}
