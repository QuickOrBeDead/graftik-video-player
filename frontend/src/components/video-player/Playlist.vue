<script setup lang="ts">
import draggable from 'vuedraggable'
import { usePlayer } from './composables/usePlayer'
import { Modal } from 'bootstrap'
import { nextTick, ref, watch, inject } from 'vue'
import { formatTime } from './utils'
import ContextMenu from '@imengyu/vue3-context-menu'
import { Playlist, PlaylistItem, PlaylistSortInfo, PlaylistViewMode, VideoMetadata, StreamInfo } from './types'
import pLimit from 'p-limit'
import { usePlaylist } from './composables/usePlaylist'
import { logger } from '@renderer/utils/logger'
import { PlaylistItemDto } from '@renderer/data/playlist'

const emit = defineEmits<{
  beforePlaylistItemChange: []
}>()

const props = defineProps<{ playlist: Playlist | null }>()
const { playerState, playVideo, calculatePercent, pause } = usePlayer()
const {
  playlistState,
  filteredPlaylist,
  viewModeClass,
  setPlaylist,
  deletePlaylistItem,
  changeViewMode,
  toggleShowOnlyUnwatched,
  getCurrentPlaylistItem,
  updatePlaylistItem,
  setPlaylistCurrentItem,
  setNewPlaylistItemsOrderIndexes,
  setPlaylistItemNewOrder,
  addNewPlaylistItems,
  totalPlaylistTimeDuration
} = usePlaylist()

const currentPlaylistItem = ref<PlaylistItem>()
const deletePlaylistItemModal = ref<HTMLDivElement>()
const showErrorModal = inject('showErrorModal') as (msg: string) => void

const loadVideoMetadataPLimit = pLimit(2)

watch(
  () => props.playlist,
  async (newData) => {
    if (newData) {
      setPlaylist(newData)

      const currentPlaylistItem = getCurrentPlaylistItem()
      if (currentPlaylistItem) {
        nextTick(async () => {
          await playItem(currentPlaylistItem)
        })
      }

      playlistState.items.map(i => loadVideoMetadata(i))
    }
  },
  { immediate: true }
)

watch(
  [() => playerState.currentTime, () => playerState.playlistItemId, () => playerState.duration],
  ([newCurrentTime, newPlaylistItemId, newDuration], []) => {
    updatePlaylistItem(newPlaylistItemId, newCurrentTime, calculatePercent(newCurrentTime, newDuration))
  }
)

window.runtime.EventsOn('add-playlist-item', async (items: unknown) => {
  const itemsTyped = items as PlaylistItemDto[]
  logger.debug('Playlist: add-playlist-item event', { count: itemsTyped.length })
  setNewPlaylistItemsOrderIndexes(itemsTyped)

  try {
    await window.go.internal.PlayerService.AddPlaylistItems(itemsTyped)
  } catch (err) {
    showErrorModal('Could not add playlist items.')
    logger.error('Playlist: failed to add playlist items:', err)
  }

  addNewPlaylistItems(itemsTyped)

  playlistState.items.filter(x => itemsTyped.some(y => y.id === x.id)).map(i => loadVideoMetadata(i))
})

const loadVideoMetadata = (i: PlaylistItem) => {
  logger.debug('Playlist: loading video metadata', { id: i.id, path: i.path })
  loadVideoMetadataPLimit(() => getVideoMetadata(playlistState.id, i.id, i.path))
    .then(data => {
      i.thumbnailImage = data.thumbnail
      i.duration = data.duration
    })
    .catch(err => {
      logger.error('Playlist: failed to load video metadata', { id: i.id, path: i.path, error: err })
    })

  loadVideoMetadataPLimit(() => getStreamInfo(i.path))
    .then(info => {
      i.streamInfo = info
    })
    .catch(err => {
      logger.error('Playlist: failed to load stream info', { id: i.id, path: i.path, error: err })
    })
}

const getVideoMetadata = async (playlistId: string, playlistItemId: string, videoPath: string): Promise<VideoMetadata> => {
  return await window.go.internal.PlayerService.GetPlaylistItemVideoMetadata(playlistId, playlistItemId, videoPath) as VideoMetadata
}

const getStreamInfo = async (videoPath: string): Promise<StreamInfo | undefined> => {
  try {
    return await window.go.internal.PlayerService.GetStreamInfo(videoPath) as StreamInfo
  } catch (e) {
    logger.error('GetStreamInfo error:', e)
    return undefined
  }
}

const playItem = async (item: PlaylistItem, forceStart?: boolean) => {
  logger.debug('Playlist: play item', { id: item.id, title: item.title })
  emit("beforePlaylistItemChange")

  const restartTime = item.progressPercent !== undefined && item.progressPercent >= 100 ? 0 : (item.elapsedTime ?? 0)
  await playVideo(item.path, restartTime, item.id, forceStart ? true : item.isPlaying)
  setPlaylistCurrentItem(item.id)
}

const confirmDeletePlaylistItem = (item: PlaylistItem) => {
    currentPlaylistItem.value = item
    if (deletePlaylistItemModal.value) {
      const modalInstance = new Modal(deletePlaylistItemModal.value!)
      modalInstance.show()
    }
}

const hideDeletePlaylistItemModal = () => {
    if (deletePlaylistItemModal.value) {
      const modalInstance = Modal.getInstance(deletePlaylistItemModal.value!)
      if (modalInstance) {
          modalInstance.hide()
      }
    }
}

const deleteItem = async () => {
  if (currentPlaylistItem.value) {
    logger.debug('Playlist: deleting item', { id: currentPlaylistItem.value.id, title: currentPlaylistItem.value.title })
    try {
      await window.go.internal.PlayerService.DeletePlaylistItem(currentPlaylistItem.value.id)
      deletePlaylistItem(currentPlaylistItem.value.id)
    } catch (err) {
      showErrorModal('Could not delete playlist item.')
      logger.error('Playlist: failed to delete item:', err)
    }
  }

  hideDeletePlaylistItemModal()
}

const openContainingFolder = (item: PlaylistItem) => {
  window.go.internal.PlayerService.OpenContainingFolder(item.path)
}

const showContextMenu = (e: MouseEvent, item: PlaylistItem) => {
  e.preventDefault()
  ContextMenu.showContextMenu({
    x: e.x,
    y: e.y,
    theme: 'mac dark',
    items: [
      {
        label: item.isPlaying && playerState.isPlaying ? 'Pause' : 'Play',
        icon: item.isPlaying && playerState.isPlaying ? 'bi bi-pause-fill' : 'bi bi-play-fill',
        iconFontClass: item.isPlaying && playerState.isPlaying ? 'text-warning' : 'text-success',
        onClick: async () => {
          if (item.isPlaying && playerState.isPlaying) {
            pause()
          } else {
            await playItem(item, true)
          }
        }
      },
      {
        label: 'Remove from Playlist',
        icon: 'bi bi-trash',
        iconFontClass: 'text-danger',
        onClick: () => {
          confirmDeletePlaylistItem(item)
        }
      },
      {
        label: 'Open Containing Folder',
        icon: 'bi bi-folder2-open',
        iconFontClass: 'text-primary',
        onClick: () => {
          openContainingFolder(item)
        }
      }
    ]
  })
}

const updatePlaylistItemOrder = async (event: { moved: { element: PlaylistItem; newIndex: number, oldIndex: number } }) => {
  if (event.moved) {
    const { element, newIndex, oldIndex } = event.moved
    logger.debug('Playlist: update item order', { elementId: element.id, oldIndex, newIndex })
    const { rebalanceOrder } = setPlaylistItemNewOrder(element, oldIndex, newIndex)

    try {
      await window.go.internal.PlayerService.UpdatePlaylistItem(element.id, { order_index: element.orderIndex })
    } catch (err) {
      showErrorModal('Could not update item order.')
      logger.error('Playlist: failed to update item order:', err)
    }

    if (rebalanceOrder) {
      try {
        await window.go.internal.PlayerService.RebalancePlaylistOrder(playlistState.id)
      } catch (err) {
        showErrorModal('Could not rebalance playlist order.')
        logger.error('Playlist: failed to rebalance playlist order:', err)
      }
    }
  }
}
</script>

<template>
  <div
    id="playlist" ref="playlistElement"
    v-show="playerState.sidebarVisible"
    class="playlist-sidebar rounded-4 overflow-hidden"
    :style="{ width: playerState.sidebarWidth + 'px' }"
    >
    <div class="playlist-header">
        <div class="playlist-title-row">
          <span class="playlist-name">{{ playlistState.name ?? '[Unnamed]' }}</span>
          <i class="bi bi-collection-play text-secondary"></i>
        </div>
        <div class="playlist-stats">
          <span>{{ playlistState.items.length}} videos</span>
          <span>Total: {{ formatTime(totalPlaylistTimeDuration) }}</span>
        </div>
        <div class="playlist-controls">
            <button class="filter-btn" :class="{ active: playlistState.viewMode === PlaylistViewMode.Detailed }" @click="() => changeViewMode(PlaylistViewMode.Detailed)" title="Detailed View">
              <i class="bi bi-view-list"></i>
            </button>
            <button class="filter-btn" :class="{ active: playlistState.viewMode === PlaylistViewMode.Simple }" @click="() => changeViewMode(PlaylistViewMode.Simple)" title="Simple View">
              <i class="bi bi-list"></i>
            </button>

            <div class="d-flex align-items-center gap-1 flex-grow-1">
              <select v-model="playlistState.sortInfo" class="sort-select">
                <option :value="PlaylistSortInfo.Default">Default</option>
                <option :value="PlaylistSortInfo.NameAsc">Name (A-Z)</option>
                <option :value="PlaylistSortInfo.NameDesc">Name (Z-A)</option>
                <option :value="PlaylistSortInfo.DurationAsc">Length (Shortest)</option>
                <option :value="PlaylistSortInfo.DurationDesc">Length (Longest)</option>
                <option :value="PlaylistSortInfo.Newest">Recently Watched</option>
                <option :value="PlaylistSortInfo.Oldest">Oldest Activity</option>
              </select>
            </div>

            <div>
              <button class="filter-btn" :class="{ active: playlistState.showOnlyUnwatched }" @click="toggleShowOnlyUnwatched">
                Unwatched
              </button>
            </div>
        </div>
    </div>

    <div class="playlist-items overflow-auto" :class="viewModeClass">
      <draggable
        v-if="playlistState"
        v-model="filteredPlaylist"
        group="items"
        item-key="id"
        tag="div"
        @change="updatePlaylistItemOrder"
      >
        <template #item="{ element }: { element: PlaylistItem }">
          <div
            :class="{ active: element.id === playlistState.currentItem }"
            class="playlist-item"
            @click="async () => await playItem(element, true)"
            @contextmenu="(e) => showContextMenu(e, element)"
          >
            <div class="playlist-item-thumb">
              <img v-if="element.thumbnailImage" :src="element.thumbnailImage" :title="element.title">
              <i v-else class="bi bi-play-btn text-secondary fs-2"></i>
              <span class="duration-badge">{{ formatTime(element.duration) }}</span>
            </div>
            <div class="playlist-item-content">
            <div class="playlist-item-title text-truncate">{{ element.title }}</div>
               <div class="playlist-item-footer">
                <div class="playlist-progress-container" v-if="true">
                  <div class="playlist-progress-bar" :style="{ width: (element.progressPercent ?? 0) + '%' }"></div>
                </div>
                <div class="playlist-time-info">
                  <span v-if="element.elapsedTime">
                    {{ formatTime(element.elapsedTime) }} watched
                  </span>
                </div>
              </div>
            </div>
            <div v-if="false">
              <a
                class="text-decoration-none text-danger"
                @click="() => confirmDeletePlaylistItem(element)"
              >
                <i class="bi bi-trash"></i>
              </a>
            </div>
          </div>
        </template>
      </draggable>
    </div>
  </div>

  <div
    ref="deletePlaylistItemModal"
    class="modal fade"
    tabindex="-1"
    aria-labelledby="deleteModalLabel"
    aria-hidden="true"
    data-bs-theme="dark"
  >
    <div class="modal-dialog">
      <div class="modal-content">
        <div class="modal-header">
          <h5 id="deleteModalLabel" class="modal-title text-white fs-6">Confirm Remove</h5>
          <button
            type="button"
            class="btn-close"
            aria-label="Close"
            data-bs-dismiss="modal"
          ></button>
        </div>
        <div class="modal-body text-white">
          Are you sure you want to remove '{{ currentPlaylistItem?.title }}' playlist item?
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
          <button type="button" class="btn btn-danger" @click="deleteItem">Remove</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style lang="css" scoped>
  .playlist-sidebar {
    background-color: var(--sidebar-bg);
    border: 1px solid #333;
    display: flex;
    flex-direction: column;
    z-index: 20;
    width: 350px;
    min-width: 280px;
  }

  @media (max-width: 768px) {
    .playlist-sidebar {
      position: absolute;
      right: 0;
      height: 100%;
      box-shadow: -5px 0 15px rgba(0,0,0,0.5);
    }
  }

  .playlist-header {
    padding: 15px;
    border-bottom: 1px solid #333;
    background: var(--sidebar-bg);
  }

  .playlist-title-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 5px;
    gap: 10px;
  }

  .playlist-name {
    font-weight: bold;
    text-transform: uppercase;
    font-size: 0.85rem;
    letter-spacing: 1.2px;
    color: var(--accent-blue);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .playlist-stats {
    display: flex;
    justify-content: space-between;
    font-size: 0.72rem;
    color: #888;
    margin-bottom: 12px;
    font-family: monospace;
  }

  .playlist-controls {
    display: flex;
    gap: 6px;
    align-items: center;
    flex-wrap: wrap;
  }

  .filter-btn {
    background: #2a2a2a;
    border: 1px solid #444;
    color: #aaa;
    font-size: 0.7rem;
    padding: 4px 8px;
    border-radius: 4px;
    transition: all 0.2s;
  }

  .filter-btn:hover {
    color: white;
    background: #333;
  }

  .filter-btn.active {
    background: var(--accent-blue);
    color: black;
    border-color: var(--accent-blue);
    font-weight: bold;
  }

  .sort-select {
    background: #2a2a2a;
    border: 1px solid #444;
    color: #ccc;
    font-size: 0.75rem;
    padding: 4px 8px;
    border-radius: 4px;
    outline: none;
    flex-grow: 1;
  }

  .playlist-items {
    flex-grow: 1;
    overflow-y: auto;
  }

  .playlist-item {
    padding: 12px;
    border-bottom: 1px solid #252525;
    cursor: pointer;
    transition: background 0.2s;
    display: flex;
    gap: 12px;
    align-items: flex-start;
  }

  .playlist-items.simple .playlist-item {
    padding: 10px 15px;
    align-items: center;
    gap: 8px;
  }

  .playlist-items.simple .playlist-item-thumb,
  .playlist-items.simple .playlist-progress-container,
  .playlist-items.simple .playlist-time-info span:first-child {
    display: none;
  }

  .playlist-item:hover {
    background-color: var(--hover-bg);
  }

  .playlist-item.active {
    background-color: #262626;
    border-left: 4px solid var(--accent-blue);
  }

  .playlist-item-thumb {
    width: 100px;
    aspect-ratio: 16/9;
    background: #000;
    border-radius: 4px;
    overflow: hidden;
    flex-shrink: 0;
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1px solid #333;
  }

  .playlist-item-thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .playlist-item-thumb .duration-badge {
    position: absolute;
    bottom: 2px;
    right: 2px;
    background: rgba(0,0,0,0.8);
    color: white;
    font-size: 0.65rem;
    padding: 1px 4px;
    border-radius: 2px;
    font-family: monospace;
  }

  .playlist-item-content {
    flex-grow: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .playlist-item-title {
    font-size: 0.85rem;
    font-weight: 500;
    line-height: 1.2;
    color: #efefef;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .playlist-item.active .playlist-item-title {
    color: var(--accent-blue);
  }

  .playlist-item-footer {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-top: 2px;
  }

  .playlist-progress-container {
    width: 100%;
    height: 3px;
    background: #333;
    border-radius: 1px;
    overflow: hidden;
  }

  .playlist-progress-bar {
    height: 100%;
    background: var(--accent-blue);
    transition: width 0.3s ease;
  }

  .playlist-time-info {
    font-size: 0.68rem;
    color: #888;
    font-family: monospace;
  }


</style>
