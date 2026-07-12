import { describe, it, expect, vi, beforeEach } from 'vitest'
import { usePlaylist } from '../usePlaylist'
import { PlaylistItem } from '../../types'

vi.mock('@renderer/utils/logger', () => ({
  logger: {
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn()
  }
}))

function assertIncreasingOrderIndexes(playlistState: ReturnType<typeof usePlaylist>['playlistState']) {
  const items = playlistState.items
  for (let i = 0; i < items.length - 1; i++) {
    expect(items[i].orderIndex).toBeLessThan(items[i + 1].orderIndex)
  }
}

function makeItem(id: string, orderIndex: number): PlaylistItem {
  return {
    id,
    playlistId: 'pl-1',
    path: `/path/${id}.mp4`,
    title: `Title ${id}`,
    orderIndex
  }
}

describe('setPlaylistItemNewOrder', () => {
  let playlist: ReturnType<typeof usePlaylist>

  beforeEach(() => {
    playlist = usePlaylist()
    playlist.setPlaylist({
      id: 'pl-1',
      name: 'Test Playlist',
      items: [],
      shuffledDeck: []
    })
  })

  describe('move to beginning (newIndex === 0)', () => {
    it('halves the first item orderIndex', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000), makeItem('c', 3000)]
      playlist.addNewPlaylistItems(items)

      const element = items[2]
      const result = playlist.setPlaylistItemNewOrder(element, 2, 0)

      expect(result.rebalanceOrder).toBe(false)
      expect(element.orderIndex).toBe(500)
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['c', 'a', 'b'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([500, 1000, 2000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('uses 1000 for empty list', () => {
      const element = makeItem('x', 0)
      playlist.playlistState.items = []

      const result = playlist.setPlaylistItemNewOrder(element, 0, 0)

      expect(result.rebalanceOrder).toBe(false)
      expect(element.orderIndex).toBe(1000)
      expect(playlist.playlistState.items).toHaveLength(1)
      expect(playlist.playlistState.items[0].id).toBe('x')
      assertIncreasingOrderIndexes(playlist.playlistState)
    })
  })

  describe('move to end (newIndex >= items.length - 1)', () => {
    it('adds 1000 to the last item orderIndex', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000), makeItem('c', 3000)]
      playlist.addNewPlaylistItems(items)

      const element = items[0]
      const result = playlist.setPlaylistItemNewOrder(element, 0, 2)

      expect(result.rebalanceOrder).toBe(false)
      expect(element.orderIndex).toBe(4000)
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['b', 'c', 'a'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([2000, 3000, 4000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('moves single item (newIndex=0, single-element list returns 1000)', () => {
      const items = [makeItem('a', 1000)]
      playlist.addNewPlaylistItems(items)

      const result = playlist.setPlaylistItemNewOrder(items[0], 0, 0)

      expect(result.rebalanceOrder).toBe(false)
      expect(items[0].orderIndex).toBe(1000)
      assertIncreasingOrderIndexes(playlist.playlistState)
    })
  })

  describe('move to middle', () => {
    it('averages the neighboring orderIndexes', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000), makeItem('c', 3000), makeItem('d', 4000)]
      playlist.addNewPlaylistItems(items)

      const element = items[0]
      const result = playlist.setPlaylistItemNewOrder(element, 0, 2)

      expect(result.rebalanceOrder).toBe(false)
      expect(element.orderIndex).toBe(3500)
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['b', 'c', 'a', 'd'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([2000, 3000, 3500, 4000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })
  })

  describe('array ordering', () => {
    it('moves element to correct position when dragging forward', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000), makeItem('c', 3000), makeItem('d', 4000), makeItem('e', 5000)]
      playlist.addNewPlaylistItems(items)

      playlist.setPlaylistItemNewOrder(items[1], 1, 3)

      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['a', 'c', 'd', 'b', 'e'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([1000, 3000, 4000, 4500, 5000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('moves element to correct position when dragging backward', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000), makeItem('c', 3000), makeItem('d', 4000), makeItem('e', 5000)]
      playlist.addNewPlaylistItems(items)

      playlist.setPlaylistItemNewOrder(items[3], 3, 1)

      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['a', 'd', 'b', 'c', 'e'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([1000, 1500, 2000, 3000, 5000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('preserves relative order of other elements', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000), makeItem('c', 3000), makeItem('d', 4000)]
      playlist.addNewPlaylistItems(items)

      playlist.setPlaylistItemNewOrder(items[0], 0, 2)

      const ids = playlist.playlistState.items.map(i => i.id)
      expect(ids.filter(id => id !== 'a')).toEqual(['b', 'c', 'd'])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })
  })

  describe('rebalance', () => {
    it('triggers rebalance when gap < 1e-12', () => {
      const items = [
        makeItem('x', 1000),
        makeItem('y', 2000),
        makeItem('z', 2000 + 1e-13)
      ]
      playlist.addNewPlaylistItems(items)

      const result = playlist.setPlaylistItemNewOrder(items[0], 0, 1)

      expect(result.rebalanceOrder).toBe(true)
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([1000, 2000, 3000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('does not trigger rebalance when gap is large enough', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000), makeItem('c', 3000)]
      playlist.addNewPlaylistItems(items)

      const result = playlist.setPlaylistItemNewOrder(items[0], 0, 2)

      expect(result.rebalanceOrder).toBe(false)
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('reassigns sequential orderIndexes on rebalance', () => {
      const items = [
        makeItem('a', 5000),
        makeItem('b', 2000),
        makeItem('c', 2000 + 1e-13)
      ]
      playlist.addNewPlaylistItems(items)

      playlist.setPlaylistItemNewOrder(items[0], 0, 1)

      const orderIndexes = playlist.playlistState.items.map(i => i.orderIndex)
      expect(orderIndexes).toEqual([1000, 2000, 3000])
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['b', 'a', 'c'])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('triggers rebalance during backward drag when neighbors are too close', () => {
      const items = [
        makeItem('a', 2000),
        makeItem('b', 2000 + 1e-13),
        makeItem('c', 1000),
        makeItem('d', 3000)
      ]
      playlist.addNewPlaylistItems(items)

      const result = playlist.setPlaylistItemNewOrder(items[2], 2, 1)

      expect(result.rebalanceOrder).toBe(true)
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['a', 'c', 'b', 'd'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([1000, 2000, 3000, 4000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })
  })

  describe('edge cases', () => {
    it('handles two-item list moving first to last', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000)]
      playlist.addNewPlaylistItems(items)

      const result = playlist.setPlaylistItemNewOrder(items[0], 0, 1)

      expect(result.rebalanceOrder).toBe(false)
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['b', 'a'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([2000, 3000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('handles two-item list moving last to first', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000)]
      playlist.addNewPlaylistItems(items)

      const result = playlist.setPlaylistItemNewOrder(items[1], 1, 0)

      expect(result.rebalanceOrder).toBe(false)
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['b', 'a'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([500, 1000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('handles move to same position', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000), makeItem('c', 3000)]
      playlist.addNewPlaylistItems(items)

      const result = playlist.setPlaylistItemNewOrder(items[1], 1, 1)

      expect(result.rebalanceOrder).toBe(false)
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['a', 'b', 'c'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([1000, 2000, 3000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('handles move when oldIndex and newIndex are the same for first element', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000)]
      playlist.addNewPlaylistItems(items)

      const result = playlist.setPlaylistItemNewOrder(items[0], 0, 0)

      expect(result.rebalanceOrder).toBe(false)
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['a', 'b'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([1000, 2000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('handles same position at end of list', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000), makeItem('c', 3000)]
      playlist.addNewPlaylistItems(items)

      const result = playlist.setPlaylistItemNewOrder(items[2], 2, 2)

      expect(result.rebalanceOrder).toBe(false)
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['a', 'b', 'c'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([1000, 2000, 3000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('handles last to first in a five-item list', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000), makeItem('c', 3000), makeItem('d', 4000), makeItem('e', 5000)]
      playlist.addNewPlaylistItems(items)

      const result = playlist.setPlaylistItemNewOrder(items[4], 4, 0)

      expect(result.rebalanceOrder).toBe(false)
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['e', 'a', 'b', 'c', 'd'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([500, 1000, 2000, 3000, 4000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })

    it('handles first to last in a five-item list', () => {
      const items = [makeItem('a', 1000), makeItem('b', 2000), makeItem('c', 3000), makeItem('d', 4000), makeItem('e', 5000)]
      playlist.addNewPlaylistItems(items)

      const result = playlist.setPlaylistItemNewOrder(items[0], 0, 4)

      expect(result.rebalanceOrder).toBe(false)
      expect(playlist.playlistState.items.map(i => i.id)).toEqual(['b', 'c', 'd', 'e', 'a'])
      expect(playlist.playlistState.items.map(i => i.orderIndex)).toEqual([2000, 3000, 4000, 5000, 6000])
      assertIncreasingOrderIndexes(playlist.playlistState)
    })
  })
})
