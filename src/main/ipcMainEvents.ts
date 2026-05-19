import { BrowserWindow, ipcMain, shell } from 'electron'
import { PlayerDataStore } from './data/playerDataStore.js'
import ffmpeg from 'fluent-ffmpeg'
import fsPromises from 'fs/promises'
import { ThumbnailDataStore } from './data/thumbnailDataStore.js'
import os from 'os'
import path from 'path'
import { VideoMetadata } from './types.js'
import { PlaylistItemDto } from './data/playlistDto.js'

export const ipcMainEvents = (mainWindow: BrowserWindow, store: PlayerDataStore, thumbnailStore: ThumbnailDataStore) => {
  ipcMain.handle('getCurrentPlaylist', async () => {
    const currentPlaylist = store.GetCurrentPlaylistId()
    if (currentPlaylist) {
      return await store.GetPlaylistById(currentPlaylist)
    }

    return null
  })

  ipcMain.handle('getPlaylists', async () => {
    return await store.GetPlaylists()
  })

  ipcMain.handle('selectPlaylist', async (e, id) => {
    store.SetCurrentPlaylistId(id)
    const playlist = await store.GetPlaylistById(id)
    mainWindow.webContents.send('load-current-playlist', playlist)

    e.sender.close()

    return true
  })

  ipcMain.handle('updatePlaylistName', async (_, x: { id: string; name: string }) => {
    await store.UpdatePlaylistName(x)
    mainWindow.webContents.send('load-playlist-name')
    return true
  })

  ipcMain.handle('addPlaylist', async (e, name: string) => {
    const playlist = await store.AddPlaylist(name)
    if (playlist && playlist.id) {
      store.SetCurrentPlaylistId(playlist.id)
      mainWindow.webContents.send('load-current-playlist', playlist)
    }

    e.sender.close()
    return true
  })

  ipcMain.handle('deletePlaylist', async (_, id: string) => {
    await store.DeletePlaylist(id)
    return true
  })

  ipcMain.handle('addPlaylistItems', (_, items: PlaylistItemDto[]) => {
    store.AddPlaylistItems(items)
  })

  ipcMain.handle('updatePlaylistItem', async(_, id: string, data: { [x: string]: any }) => {
    await store.UpdatePlaylistItem(id, data)
    return true
  })

  ipcMain.handle('deletePlaylistItem', async (_, id: string) => {
    await store.DeletePlaylistItem(id)
    return true
  })

  ipcMain.handle('getPlaylist', async (_, id: string) => {
    return await store.GetPlaylistById(id)
  })

  ipcMain.handle('getPlaylistName', async (_, id: string) => {
    return await store.GetPlaylistName(id)
  })

  ipcMain.handle('updatePlaylist', async(_, id: string, data: { [x: string]: any }) => {
    await store.UpdatePlaylist(id, data)
    return true
  })

  ipcMain.handle('getPlaylistItemVideoMetadata', async (_, playlistId: string, playlistItemId: string, videoPath: string): Promise<VideoMetadata> => {
    const stats = await fsPromises.stat(videoPath)
    const lastModified = stats.mtimeMs
    const fileSize = stats.size

    return new Promise((resolve, reject) => {
      ffmpeg.ffprobe(videoPath, async (err, metadata) => {
        if (err) {
          return reject(err)
        }

        const duration = metadata.format.duration || 0
        const thumbnailHash = thumbnailStore.calculateFileHash(videoPath, fileSize, lastModified)
        const thumbnail = await thumbnailStore.getThumbnail(playlistId, playlistItemId, thumbnailHash)
        if (thumbnail) {
          resolve({
            duration,
            lastModified,
            fileSize,
            thumbnail: `data:image/jpeg;base64,${thumbnail}`
          })
          return
        }

        const tempDir = os.tmpdir()
        const tempFileName = `${playlistItemId}-${Date.now()}.jpeg`
        const tempFilePath = path.join(tempDir, tempFileName)

        const seekTime = duration > 0 ? (duration * 0.1) : 1.0

        ffmpeg(videoPath)
          .inputOptions([`-ss ${seekTime}`])
          .outputOptions([
            '-vframes 1',
            '-f image2',
            '-vcodec mjpeg',
            '-q:v 4',
            '-sws_flags fast_bilinear'
          ])
          .size('180x?')
          .aspect('16:9')
          .on('end', async () => {
            const imageBuffer = await fsPromises.readFile(tempFilePath)
            await thumbnailStore.setThumbnail(playlistId, playlistItemId, thumbnailHash, imageBuffer)
            await fsPromises.unlink(tempFilePath).catch(() => {})

            resolve({
              duration,
              lastModified,
              fileSize,
              thumbnail: `data:image/jpeg;base64,${imageBuffer.toString('base64')}`
            })
          })
          .on('error', reject)
          .save(tempFilePath)
      })
    })
  })

  ipcMain.handle('rebalancePlaylistOrder', (_, id: string) => {
    store.RebalancePlaylistOrder(id)
  })

  ipcMain.on('open-containing-folder', (_, filePath: string) => {
    shell.showItemInFolder(filePath)
  })

  ipcMain.on('subwindow.close', (e) => e.sender.close())
}
