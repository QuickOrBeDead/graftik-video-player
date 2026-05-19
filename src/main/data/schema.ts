import { AnySQLiteColumn, index, integer, real, sqliteTable, text } from "drizzle-orm/sqlite-core"
import { relations } from 'drizzle-orm/relations'

export const playlists = sqliteTable('playlists', {
  id: text('id').primaryKey(),
  name: text('name').notNull(),
  shuffle: integer('shuffle', { mode: "boolean" }).default(false).notNull(),
  repeat: integer('repeat').default(0).notNull(),
  currentItem: text('currentItem').references((): AnySQLiteColumn => playlistItems.id, { onDelete: 'set null' }),
  currentVolume: real('currentVolume')
})

export const playlistItems = sqliteTable('playlistItems', {
  id: text('id').primaryKey(),
  playlistId: text('playlistId').notNull().references(() => playlists.id, { onDelete: 'cascade' }),
  path: text('path').notNull(),
  title: text('title').notNull(),
  currentTime: integer('currentTime').default(0).notNull(),
  isPlaying: integer('isPlaying', { mode: "boolean" }).default(false).notNull(),
  duration: real('duration'),
  progressPercent: real('progressPercent').default(0).notNull(),
  lastWatched: integer('lastWatched'),
  orderIndex: real('orderIndex').notNull()
}, (table) => [
    index('idx_playlistItems_playlistId_orderIndex').on(table.playlistId, table.orderIndex),
])

export const playlistsRelations = relations(playlists, ({ many, one }) => ({
  playlistItems: many(playlistItems),
  currentPlaylistItem: one(playlistItems, {
    fields: [playlists.currentItem],
    references: [playlistItems.id],
  }),
}))

export const playlistItemsRelations = relations(playlistItems, ({ one }) => ({
  playlist: one(playlists, {
    fields: [playlistItems.playlistId],
    references: [playlists.id],
  }),
}))

export type PlaylistType = typeof playlists.$inferSelect
