import { BetterSQLite3Database, drizzle } from 'drizzle-orm/better-sqlite3'
import Database from 'better-sqlite3'
import * as schema from './schema.js'
import { eq, sql, SQL } from 'drizzle-orm'
import { SQLiteTable } from 'drizzle-orm/sqlite-core'
import { migrate } from 'drizzle-orm/better-sqlite3/migrator'
import { PlaylistDto } from './playlistDto.js'

export type Projection<TTable extends SQLiteTable> = {
  [K in keyof TTable['_']['columns']]?: boolean;
}

export class PlayerRepository {
  private db: BetterSQLite3Database<typeof schema> & { $client: Database.Database }

  constructor(dbPath: string) {
    const sqlite = new Database(dbPath)
    sqlite.pragma('journal_mode = WAL')
    sqlite.pragma('synchronous = NORMAL')
    sqlite.pragma('journal_size_limit = 10485760')

    this.db = drizzle(sqlite, { schema })
  }

  public initializeDb(migrationsPath: string) {
    migrate(this.db, { migrationsFolder: migrationsPath })
  }

  private updateRecord<T extends SQLiteTable>(
    table: T,
    data: Partial<T['$inferInsert']>,
    whereCondition: SQL<unknown>
  ) {
    // Filter out undefined values to keep the update dynamic
    const cleanData = Object.fromEntries(
      Object.entries(data).filter(([_, v]) => v !== undefined)
    )

    if (Object.keys(cleanData).length === 0) return

    return this.db.update(table)
      .set(cleanData as any)
      .where(whereCondition)
  }

  public async getPlaylist(id: string): Promise<PlaylistDto | undefined> {
    const playlist = await this.db.query.playlists.findFirst({
      where: eq(schema.playlists.id, id),
      with: {
        playlistItems: {
          orderBy: (x, { asc }) => [asc(x.orderIndex)]
        },
        currentPlaylistItem: true
      },
    })

    if (playlist) {
      return {
        id: playlist.id,
        name: playlist.name,
        shuffle: playlist.shuffle,
        repeat: playlist.repeat ?? 0,
        currentItem: playlist.currentItem,
        currentVolume: playlist.currentVolume,
        currentPlaylistItem: !playlist.currentPlaylistItem ? null : {
          id: playlist.currentPlaylistItem.id,
          playlistId: playlist.currentPlaylistItem.playlistId,
          title: playlist.currentPlaylistItem.title,
          path: playlist.currentPlaylistItem.path,
          isPlaying: playlist.currentPlaylistItem.isPlaying,
          currentTime: playlist.currentPlaylistItem.currentTime,
          duration: playlist.currentPlaylistItem.duration,
          lastWatched: playlist.currentPlaylistItem.lastWatched,
          progressPercent: playlist.currentPlaylistItem.progressPercent,
          orderIndex: playlist.currentPlaylistItem.orderIndex
        },
        items: playlist.playlistItems.map(x => ({
          id: x.id,
          title: x.title,
          path: x.path,
          isPlaying: x.isPlaying,
          currentTime: x.currentTime,
          duration: x.duration,
          lastWatched: x.lastWatched,
          playlistId: x.playlistId,
          progressPercent: x.progressPercent,
          orderIndex: x.orderIndex
        }))
      }
    }

    return undefined
  }

  public async getPlaylistProjection(id: string, projection?: Projection<typeof schema.playlists>) {
    return await this.db.query.playlists.findFirst({
      where: eq(schema.playlists.id, id),
      columns: projection
    })
  }

  public async getPlaylists(projection?: Projection<typeof schema.playlists>) {
    return await this.db.query.playlists.findMany({
      columns: projection
    })
  }

  public async hasAnyPlaylist() {
    const firstRecord = await this.db.query.playlists.findFirst({
      columns: { id: true }
    })

    return !!firstRecord
  }

  public async addPlaylist(data: typeof schema.playlists.$inferInsert) {
    await this.db.insert(schema.playlists).values(data)
  }

  public async updatePlaylist(id: string, data: Partial<typeof schema.playlists.$inferInsert>) {
    await this.updateRecord(schema.playlists, data, eq(schema.playlists.id, id))
  }

  public async deletePlaylist(id: string) {
    await this.db.delete(schema.playlists).where(eq(schema.playlists.id, id))
  }

  public addPlaylistItems(data: typeof schema.playlistItems.$inferInsert[]) {
    const chunkSize = 50
    this.db.transaction((tx) => {
      for (let i = 0; i < data.length; i += chunkSize) {
        const chunk = data.slice(i, i + chunkSize)
        tx.insert(schema.playlistItems).values(chunk).run()
      }
    })
  }

  public async updatePlaylistItem(id: string, data: Partial<typeof schema.playlistItems.$inferInsert>) {
    await this.updateRecord(schema.playlistItems, data, eq(schema.playlistItems.id, id))
  }

  public async deletePlaylistItem(id: string) {
    await this.db.delete(schema.playlistItems).where(eq(schema.playlistItems.id, id))
  }

  /*

  WITH "reordered" AS (
    SELECT "id", ROW_NUMBER() OVER (ORDER BY "orderIndex") AS "pos"
    FROM "playlistItems"
    WHERE "playlistId" = ?
  )
  UPDATE "playlistItems"
  SET "orderIndex" = "reordered"."pos" * 1000
  FROM "reordered"
  WHERE "playlistItems"."id" = "reordered"."id" AND "playlistItems"."playlistId" = ?;

   */
  public rebalancePlaylistOrder(playlistId: string) {
    this.db.transaction(tx => {
      const reordered = tx
        .select({
          id: schema.playlistItems.id,
          pos: sql<number> `row_number() over (order by ${schema.playlistItems.orderIndex})`.as('pos'),
        })
        .from(schema.playlistItems)
        .where(sql`${schema.playlistItems.playlistId} = ${playlistId}`)
        .as('reordered')

      const update = tx
        .with(reordered)
        .update(schema.playlistItems)
        .set({ orderIndex: sql`reordered.pos * 1000` })
        .from(sql`reordered`)
        .where(sql`${schema.playlistItems.id} = ${reordered.id} AND ${schema.playlistItems.playlistId} = ${playlistId}`)

      update.run()
    })
  }

  /**
   * Performs a clean shutdown of the database.
   * Merges WAL files and closes the connection.
   */
  public dbShutdown() {
    try {
      console.log('Starting database checkpoint...')
      this.db.$client.pragma('wal_checkpoint(TRUNCATE)')
      this.db.$client.close()
      console.log('Database connection closed safely.')
    } catch (err) {
      console.error('Database shutdown error:', err)
    }
  }
}
