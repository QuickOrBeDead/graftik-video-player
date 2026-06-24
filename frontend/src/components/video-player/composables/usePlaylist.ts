import { computed, reactive } from "vue"
import { Playlist, PlaylistItem, PlaylistSortInfo, PlaylistViewMode, RepeatMode } from "../types"

const state = reactive<Playlist>({
  id: '',
  name: '',
  items: [],
  shuffledDeck: []
})

function shuffleArray<T>(array: T[]): T[] {
  const result = [...array]
  for (let i = result.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [result[i], result[j]] = [result[j], result[i]]
  }
  return result
}

function generateShuffledDeck(items: PlaylistItem[]): PlaylistItem[] {
  return shuffleArray(items)
}

export function usePlaylist() {
  const setPlaylist = (data: Playlist) => {
    const defaults = {
      viewMode: PlaylistViewMode.Detailed,
      sortInfo: PlaylistSortInfo.Default,
      showOnlyUnwatched: false
    }

    Object.assign(state, { ...defaults, ...data })
  }

  const changeViewMode = (mode: PlaylistViewMode) => {
    state.viewMode = mode
  }

  const deletePlaylistItem = async (id: string) => {
    state.items = state.items.filter(x => x.id !== id)
  }

  const toggleShowOnlyUnwatched = () => {
    state.showOnlyUnwatched = !state.showOnlyUnwatched
  }

  const viewModeClass = computed(() => {
    if (state.viewMode === PlaylistViewMode.Simple) {
      return 'simple'
    }

    if (state.viewMode === PlaylistViewMode.Detailed) {
      return 'detailed'
    }

    return 'detailed'
  })

  const getCurrentPlaylistItem = () => {
    return state.items.find(x => x.id == state.currentItem)
  }

  const updateCurrentPlaylistItemTime = (newCurrentTime: number | undefined) => {
    const currentItem = getCurrentPlaylistItem()
    if (currentItem) {
        currentItem.elapsedTime = newCurrentTime
    }
  }

  const setPlaylistCurrentItem = (id: string) => {
    state.currentItem = id
  }

  const setCurrentPlaylistItemProgress = (progress: number) => {
    const currentItem = getCurrentPlaylistItem()
    if (currentItem) {
      currentItem.progressPercent = progress
      currentItem.lastWatched = new Date()
    }
  }

  const filteredPlaylist = computed(() => {
    let list = [...state.items]

    if (state.showOnlyUnwatched) {
        list = list.filter(video => {
            return !video.progressPercent || video.progressPercent < 5
        });
    }

    if (state.sortInfo !== PlaylistSortInfo.Default) {
        list.sort((a, b) => {
          switch(state.sortInfo) {
            case PlaylistSortInfo.DurationAsc:
              return (a.duration ?? 0) - (b.duration ?? 0)
            case PlaylistSortInfo.DurationDesc:
              return -((a.duration ?? 0) - (b.duration ?? 0))
            case PlaylistSortInfo.NameAsc:
              return (a.title ?? '').localeCompare((b.title ?? ''))
            case PlaylistSortInfo.NameDesc:
              return -((a.title ?? '').localeCompare((b.title ?? '')))
             case PlaylistSortInfo.Oldest:
              return (a.lastWatched?.getTime() ?? 0) - (b.lastWatched?.getTime() ?? 0)
            case PlaylistSortInfo.Newest:
              return -((a.lastWatched?.getTime() ?? 0) - (b.lastWatched?.getTime() ?? 0))
          }

          return 0
        })
    }

    return list
  })

  const totalPlaylistTimeDuration = computed(() => {
    return filteredPlaylist.value.reduce((acc, video) => acc + (video.duration ?? 0), 0)
  })

  const setNewPlaylistItemsOrderIndexes = (items: PlaylistItem[]) => {
    let maxOrderIndex = state.items.length === 0 ? 0 : state.items[state.items.length - 1].orderIndex
    for (let i = 0; i < items.length; i++) {
      const item = items[i]
      maxOrderIndex += 1000
      item.orderIndex = maxOrderIndex
      item.playlistId = state.id
    }
  }

  const addNewPlaylistItems = (items: PlaylistItem[]) => {
    for (let i = 0; i < items.length; i++) {
      state.items.push(items[i])
    }
  }

  const setPlaylistItemNewOrder = (element: PlaylistItem, oldIndex: number, newIndex: number): { rebalanceOrder: boolean } => {
    const items = state.items

    // calculate index by Fractional Indexing. nexIndex = (prev + next) / 2
    function calculateNewIndex() {
      let newOrderIndex: number
      if (newIndex === 0) {
        newOrderIndex = items.length === 0 ? 1000 : (items[0].orderIndex / 2)
      } else if (newIndex >= items.length - 1) {
        newOrderIndex = items[newIndex].orderIndex + 1000
      } else {
        newOrderIndex = (items[newIndex].orderIndex + items[newIndex + 1].orderIndex) / 2
      }
      return newOrderIndex
    }

    function moveElement(el: PlaylistItem) {
      items.splice(oldIndex, 1)
      items.splice(newIndex, 0, el)
    }

    const newOrderIndex: number = calculateNewIndex()
    element.orderIndex = newOrderIndex
    moveElement(element)

    const prevOrder = items[newIndex]?.orderIndex ?? 0
    const nextOrder = newIndex >= items.length - 1 ? newOrderIndex + 1000 : items[newIndex + 1]?.orderIndex ?? (newOrderIndex + 1000)
    const gap = Math.abs(nextOrder - prevOrder)

    // next - prev < 1e-12 request rebalance list
    const rebalance = gap < 1e-12

    if (rebalance) {
      items.forEach((v, i) => { v.orderIndex = i + 1000 })
    }

    return {
      rebalanceOrder: rebalance
    }
  }

  const getNextPlaylistItem = (repeat: number, shuffle: boolean): PlaylistItem | null => {
    if (repeat === RepeatMode.One) {
      const currentItem = state.items.find(x => x.id === state.currentItem)
      return currentItem ?? null
    }

    let deck = state.items
    if (shuffle) {
      if (state.shuffledDeck.length === 0 && state.items.length > 0) {
        state.shuffledDeck = generateShuffledDeck(state.items)
      }
      deck = state.shuffledDeck
    }

    const currentIndex = deck.findIndex(x => x.id === state.currentItem)
    if (currentIndex === -1) {
      return null
    }

    const nextIndex = currentIndex + 1

    if (nextIndex < deck.length) {
      return deck[nextIndex]
    }

    if (repeat === RepeatMode.All && deck.length > 0) {
      if (shuffle) {
        state.shuffledDeck = generateShuffledDeck(state.items)
        return state.shuffledDeck[0]
      }
      return deck[0]
    }

    return null
  }

  const getPreviousPlaylistItem = (repeat: number, shuffle: boolean): PlaylistItem | null => {
    if (repeat === RepeatMode.One) {
      const currentItem = state.items.find(x => x.id === state.currentItem)
      return currentItem ?? null
    }

    let deck = state.items
    if (shuffle) {
      deck = state.shuffledDeck.length > 0 ? state.shuffledDeck : generateShuffledDeck(state.items)
    }

    const currentIndex = deck.findIndex(x => x.id === state.currentItem)
    if (currentIndex === -1) {
      return null
    }

    const prevIndex = currentIndex - 1

    if (prevIndex >= 0) {
      return deck[prevIndex]
    }

    if (repeat === RepeatMode.All && deck.length > 0) {
      return deck[deck.length - 1]
    }

    return null
  }

  const regenerateShuffledDeck = () => {
    if (state.items.length > 0) {
      state.shuffledDeck = generateShuffledDeck(state.items)
    } else {
      state.shuffledDeck = []
    }
  }

  const clearShuffledDeck = () => {
    state.shuffledDeck = []
  }

  return {
    playlistState: state,
    filteredPlaylist,
    viewModeClass,
    setPlaylist,
    changeViewMode,
    deletePlaylistItem,
    toggleShowOnlyUnwatched,
    getCurrentPlaylistItem,
    updateCurrentPlaylistItemTime,
    setPlaylistCurrentItem,
    setCurrentPlaylistItemProgress,
    setNewPlaylistItemsOrderIndexes,
    setPlaylistItemNewOrder,
    addNewPlaylistItems,
    totalPlaylistTimeDuration,
    getNextPlaylistItem,
    getPreviousPlaylistItem,
    regenerateShuffledDeck,
    clearShuffledDeck
  }
}

