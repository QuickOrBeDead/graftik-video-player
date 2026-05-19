import path from 'path'
import fs from 'fs'
import fsPromises from 'fs/promises'
import crypto from 'crypto'

export class ThumbnailDataStore {
  cacheFolder: string

  constructor(userDataPath: string) {
    this.cacheFolder = path.join(userDataPath, 'thumbnails')

    if (!fs.existsSync(this.cacheFolder)) {
      fs.mkdirSync(this.cacheFolder)
    }
  }

  private async ensurePlaylistFolder(playlistId: string): Promise<string> {
    const folder = path.join(this.cacheFolder, playlistId)

    try {
      await fsPromises.mkdir(folder, { recursive: true })
    } catch (err: any) {
      // If the error is "File exists", we don't care. It's a success for us.
      if (err.code !== 'EEXIST') {
        throw err // Rethrow actual errors (permissions, disk full, etc.)
      }
    }
    return folder
  }

  private static getThumbnailPathWithPlaylistPath(playlistPath: string, playlistItemId: string, fileHash: string): string {
    return path.join(playlistPath, `${playlistItemId}-${fileHash}.jpeg`)
  }

  private async getThumbnailPath(playlistId: string, playlistItemId: string, fileHash: string): Promise<string> {
    return ThumbnailDataStore.getThumbnailPathWithPlaylistPath(await this.ensurePlaylistFolder(playlistId), playlistItemId, fileHash)
  }

  public calculateFileHash(filePath: string, fileSize: number, lastModified: number): string {
    const input = `${filePath}-${fileSize}-${lastModified}`
    return crypto.createHash('sha256').update(input).digest('hex')
  }

  public async getThumbnail(playlistId: string, playlistItemId: string, fileHash: string): Promise<string | null> {
    const thumbPath = await this.getThumbnailPath(playlistId, playlistItemId, fileHash)
    const stats = await fsPromises.stat(thumbPath).catch(() => null)

    if (!stats || !stats.isFile()) {
      return null
    }

    try {
      const imageBuffer = await fsPromises.readFile(thumbPath)
      return imageBuffer.toString('base64')
    } catch (e) {
      console.error("Failed to read thumbnail:", e)
      return null
    }
  }

  public async setThumbnail(playlistId: string, playlistItemId: string, fileHash: string, imageBuffer: Buffer) {
    const playlistFolder = await this.ensurePlaylistFolder(playlistId)

    const files = await fsPromises.readdir(playlistFolder)
    const prefix = `${playlistItemId}-`

    const deletePromises = files
      .filter(file => file.startsWith(prefix) && file.endsWith('.jpeg'))
      .map(file => fsPromises.unlink(path.join(playlistFolder, file)).catch(() => {
        // Silently catch errors (e.g., file already deleted)
      }))

    await Promise.all(deletePromises)

    const newThumbPath = ThumbnailDataStore.getThumbnailPathWithPlaylistPath(playlistFolder, playlistItemId, fileHash)
    await fsPromises.writeFile(newThumbPath, imageBuffer)
  }
}
