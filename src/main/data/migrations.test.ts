import { describe, it, expect, afterEach } from 'vitest'
import fs from 'fs'
import path from 'path'
import { BetterSQLite3Database } from 'drizzle-orm/better-sqlite3'
import { PlayerRepository } from './playerRepository.js'
import { sql } from 'drizzle-orm'
import Database from 'better-sqlite3'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const TEST_DB_PATH = path.join(process.cwd(), 'test-migration.db')
const MIGRATIONS_PATH = path.join(__dirname, '../../../drizzle')

function cleanup() {
  if (fs.existsSync(TEST_DB_PATH)) {
    fs.unlinkSync(TEST_DB_PATH)
  }
}

function migrateToVersion(db: BetterSQLite3Database, versionPrefix: string) {
  const files = fs.readdirSync(MIGRATIONS_PATH)
    .filter(f => f.endsWith('.sql'))
    .sort()

  db.transaction(() => {
    for (const file of files) {
      const currentPrefix = file.split('_')[0]
      if (currentPrefix > versionPrefix) break

      const sql = fs.readFileSync(path.join(MIGRATIONS_PATH, file), 'utf8')
      db.run(sql)
    }
  })
}

function getLocalLatestVersionId(): string | null {
  const journalPath = path.join(MIGRATIONS_PATH, 'meta', '_journal.json')
  if (!fs.existsSync(journalPath)) {
    console.warn(`Journal not found at ${journalPath}`)
    return null
  }

  const journal = JSON.parse(fs.readFileSync(journalPath, 'utf8'))
  if (journal.entries.length === 0) {
    console.warn(`No journal entry found`)
    return null
  }

  const matchedEntry = journal.entries[journal.entries.length - 1]
  return matchedEntry.tag.split('_')[0]
}

function getDbLatestVersionId(db: BetterSQLite3Database): string | null {
  const row = db.get<{ created_at: number }>(sql`
    SELECT created_at
    FROM __drizzle_migrations
    ORDER BY created_at DESC
    LIMIT 1
  `) as { created_at: number } | undefined

  if (!row) return null

  const journalPath = path.join(MIGRATIONS_PATH, 'meta', '_journal.json')
  if (!fs.existsSync(journalPath)) {
    console.warn(`Journal not found at ${journalPath}`)
    return null
  }

  const journal = JSON.parse(fs.readFileSync(journalPath, 'utf8'))
  const matchedEntry = journal.entries.find((entry: any) => entry.when === row.created_at)

  if (!matchedEntry) {
    console.warn(`No journal entry found matching timestamp: ${row.created_at}`)
    return null;
  }

  const versionPart = matchedEntry.tag.split('_')[0]
  return versionPart
}

function createTestDb(startVersion?: string | null): BetterSQLite3Database<any> & { $client: Database.Database } {
  cleanup()
  const repository = new PlayerRepository(TEST_DB_PATH)
  const db = (repository as any).db as BetterSQLite3Database<any> & { $client: Database.Database }

  if (startVersion) {
    migrateToVersion(db, startVersion)
  }

  repository.initializeDb(MIGRATIONS_PATH)

  return db
}

describe('Database Migrations', () => {
  afterEach(() => {
    cleanup()
  })

  it('should run all migrations on fresh database', () => {
    const db = createTestDb()
    const version = getLocalLatestVersionId()
    const latestDbVersion = getDbLatestVersionId(db)
    expect(version).toBe(latestDbVersion)

    db.$client.close()
  })

  it('should create playlists table with all columns', () => {
    const db = createTestDb()

    const tableInfo = db.$client.pragma('table_info(playlists)') as any[]
    const columnNames = tableInfo.map((col) => col.name)

    expect(columnNames).toContain('id')
    expect(columnNames).toContain('name')
    expect(columnNames).toContain('shuffle')
    expect(columnNames).toContain('repeat')
    expect(columnNames).toContain('currentItem')
    expect(columnNames).toContain('currentVolume')

    db.$client.close()
  })

  it('should create playlistItems table in version 1', () => {
    const db = createTestDb()
    const tables = db.$client
      .prepare("SELECT name FROM sqlite_master WHERE type='table' AND name='playlistItems'")
      .all() as any[]

    expect(tables.length).toBe(1)
    db.$client.close()
  })

  it('should allow inserting playlists', () => {
    const db = createTestDb()

    db.$client.exec(`
      INSERT INTO playlists (id, name)
      VALUES ('test-123', 'Test Playlist')
    `)

    const row = db.$client.prepare('SELECT * FROM playlists WHERE id = ?').get('test-123') as any
    expect(row).toBeDefined()
    expect(row.name).toBe('Test Playlist')
    expect(row.shuffle).toBe(0)
    expect(row.repeat).toBe(0)
    db.$client.close()
  })

it('should create playlistItems with all columns', () => {
    const db = createTestDb()

    const tableInfo = db.$client.pragma('table_info(playlistItems)') as any[]
    const columnNames = tableInfo.map((col) => col.name)

    expect(columnNames).toContain('id')
    expect(columnNames).toContain('playlistId')
    expect(columnNames).toContain('path')
    expect(columnNames).toContain('title')
    expect(columnNames).toContain('currentTime')
    expect(columnNames).toContain('isPlaying')
    expect(columnNames).toContain('duration')
    expect(columnNames).toContain('progressPercent')
    expect(columnNames).toContain('lastWatched')
    expect(columnNames).toContain('orderIndex')

    db.$client.close()
  })

it('should handle multiple playlists', () => {
    const db = createTestDb()


    db.$client.exec(`
      INSERT INTO playlists (id, name) VALUES
        ('id-1', 'Playlist 1'),
        ('id-2', 'Playlist 2'),
        ('id-3', 'Playlist 3')
    `)

    const rows = db.$client.prepare('SELECT COUNT(*) as count FROM playlists').get() as any
    expect(rows.count).toBe(3)

    const allRows = db.$client.prepare('SELECT * FROM playlists').all() as any[]
    allRows.forEach((row) => {
      expect(row.shuffle).toBe(0)
      expect(row.repeat).toBe(0)
    })

    db.$client.close()
  })

  describe('playlistItems table tests', () => {
    it('should create composite index', () => {
      const db = createTestDb()

      const indexes = db.$client
        .prepare("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='playlistItems'")
        .all() as any[]

      const indexNames = indexes.map((idx) => idx.name)
      expect(indexNames).toContain('idx_playlistItems_playlistId_orderIndex')
      db.$client.close()
    })

    it('should enforce foreign key constraint', () => {
      const db = createTestDb()
      db.$client.pragma('foreign_keys = ON')

      // Insert a valid playlist first
      db.$client.exec(`
        INSERT INTO playlists (id, name)
        VALUES ('playlist-1', 'Test Playlist')
      `)

      // Insert item with valid foreign key should succeed
      const validInsert = db.$client.prepare(`
        INSERT INTO playlistItems (id, playlistId, path, title, orderIndex)
        VALUES (?, ?, ?, ?, ?)
      `)
      expect(() => {
        validInsert.run('item-1', 'playlist-1', '/video1.mp4', 'Video 1', 0)
      }).not.toThrow()

      // Insert item with invalid foreign key should fail
      const invalidInsert = db.$client.prepare(`
        INSERT INTO playlistItems (id, playlistId, path, title, orderIndex)
        VALUES (?, ?, ?, ?, ?)
      `)
      expect(() => {
        invalidInsert.run('item-2', 'nonexistent-playlist', '/video2.mp4', 'Video 2', 0)
      }).toThrow()

      db.$client.close()
    })

    it('should cascade delete playlist items when playlist is deleted', () => {
      const db = createTestDb()
      db.$client.pragma('foreign_keys = ON')

      // Create playlist and items
      db.$client.exec(`
        INSERT INTO playlists (id, name)
        VALUES ('playlist-cascade', 'Test Playlist')
      `)

      db.$client.exec(`
        INSERT INTO playlistItems (id, playlistId, path, title, orderIndex)
        VALUES
          ('item-1', 'playlist-cascade', '/video1.mp4', 'Video 1', 0),
          ('item-2', 'playlist-cascade', '/video2.mp4', 'Video 2', 1)
      `)

      // Verify items exist
      const beforeDelete = db.$client
        .prepare('SELECT COUNT(*) as count FROM playlistItems WHERE playlistId = ?')
        .get('playlist-cascade') as any
      expect(beforeDelete.count).toBe(2)

      // Delete playlist
      db.$client.exec("DELETE FROM playlists WHERE id = 'playlist-cascade'")

      // Verify items were cascaded deleted
      const afterDelete = db.$client
        .prepare('SELECT COUNT(*) as count FROM playlistItems WHERE playlistId = ?')
        .get('playlist-cascade') as any
      expect(afterDelete.count).toBe(0)

      db.$client.close()
    })

    it('should apply default values for optional columns', () => {
      const db = createTestDb()

      db.$client.exec(`
        INSERT INTO playlists (id, name)
        VALUES ('playlist-defaults', 'Test Playlist')
      `)

      db.$client.exec(`
        INSERT INTO playlistItems (id, playlistId, path, title, orderIndex)
        VALUES ('item-defaults', 'playlist-defaults', '/video.mp4', 'Video', 0)
      `)

      const row = db.$client
        .prepare('SELECT * FROM playlistItems WHERE id = ?')
        .get('item-defaults') as any

      expect(row.currentTime).toBe(0)
      expect(row.isPlaying).toBe(0)
      expect(row.progressPercent).toBe(0)
      expect(row.duration).toBeNull()
      expect(row.lastWatched).toBeNull()

      db.$client.close()
    })

    it('should allow inserting items with all fields', () => {
      const db = createTestDb()

      db.$client.exec(`
        INSERT INTO playlists (id, name)
        VALUES ('playlist-full', 'Test Playlist')
      `)

      db.$client.exec(`
        INSERT INTO playlistItems (id, playlistId, path, title, currentTime, isPlaying, duration, progressPercent, lastWatched, orderIndex)
        VALUES ('item-full', 'playlist-full', '/video.mp4', 'Full Video', 123.45, 1, 600.0, 20.575, 1737734400, 0)
      `)

      const row = db.$client.prepare('SELECT * FROM playlistItems WHERE id = ?').get('item-full') as any

      expect(row.id).toBe('item-full')
      expect(row.playlistId).toBe('playlist-full')
      expect(row.path).toBe('/video.mp4')
      expect(row.title).toBe('Full Video')
      expect(row.currentTime).toBe(123.45)
      expect(row.isPlaying).toBe(1)
      expect(row.duration).toBe(600.0)
      expect(row.progressPercent).toBe(20.575)
      expect(row.lastWatched).toBe(1737734400)
      expect(row.orderIndex).toBe(0)

      db.$client.close()
    })
  })
})
