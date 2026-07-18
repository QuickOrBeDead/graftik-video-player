<script setup lang="ts">
import { watch } from 'vue'
import { usePlayer } from './composables/usePlayer'
import { usePlaylist } from './composables/usePlaylist'
import PlayerView from './Player.vue'
import PlaylistView from './Playlist.vue'
import { Playlist } from './types'
import { logger } from '@renderer/utils/logger'

const { playerState, progressPercent, doSidebarResize, startSidebarResizing, stopSidebarResizing } = usePlayer()
const { playlistState } = usePlaylist()
const { playlist } = defineProps<{
  playlist: Playlist | null
}>()

watch(() => playlistState.currentItem, async (newCurrentItem, oldCurrentItem) => {
  if (playlistState.id && newCurrentItem && oldCurrentItem !== newCurrentItem) {
    logger.debug('[SAVE:PLAYLIST-META] currentItem changed — updating playlist.current_item', 'playlistId', playlistState.id, 'currentItem', newCurrentItem)
    try {
      await window.go.internal.PlayerService.UpdatePlaylist(playlistState.id, { current_item: newCurrentItem })
    } catch (err) {
      logger.error('[SAVE:PLAYLIST-META] failed to update playlist', 'error', err)
    }
  }
})

watch(() => playerState.isPlaying, async (newIsPlaying, oldIsPlaying) => {
  if (newIsPlaying !== oldIsPlaying) {
    await updatePlaylistItem()
  }
})

let updatePlaylistCurrentItemIntervalId: ReturnType<typeof setInterval>

updatePlaylistCurrentItemIntervalId = setInterval(
  async () => await updatePlaylistItem(),
  10000
)

window.runtime.EventsOn('before-app-close', async () => {
  logger.debug('[SAVE:BEFORE-CLOSE] Performing final playlist item save')
  try {
    await updatePlaylistItem(true)
    await window.go.main.App.SetReadyToClose()
  } catch (err) {
    logger.error('[SAVE:BEFORE-CLOSE] failed', 'error', err)
  }
})

window.onbeforeunload = () => {
  logger.debug('[WINDOW:BEFOREUNLOAD] Page unloading')
  try {
    if (updatePlaylistCurrentItemIntervalId) {
      clearInterval(updatePlaylistCurrentItemIntervalId)
    }
  } catch (error) {
    logger.error('clearInterval error', 'error', error)
  }
}

const updatePlaylistItem = async (closing?: boolean) => {
  if (!playlistState.currentItem) {
    logger.debug('[SAVE:ITEM-SKIP] updatePlaylistItem skipped — no currentItem')
    return
  }

  let data: any
  if (playerState.isPlaying) {
    data = {
      elapsed_time: playerState.currentTime,
      duration: playerState.duration,
      is_playing: true,
      progress_percent: progressPercent.value,
      last_watched: Date.now()
    }
  } else {
    if (closing) {
      data = {
        elapsed_time: playerState.currentTime,
        duration: playerState.duration,
        progress_percent: progressPercent.value
      }
    } else {
      data = { is_playing: false }
    }
  }

  logger.debug('[SAVE:ITEM] Saving playlist item data', 'itemId', playlistState.currentItem, 'data', data)
  try {
    await window.go.internal.PlayerService.UpdatePlaylistItem(playlistState.currentItem, data)
  } catch (err) {
    logger.error('[SAVE:ITEM] failed to update playlist item', 'error', err)
  }
}

const startResizing = () => {
  startSidebarResizing()
  window.addEventListener('mousemove', doResize)
  window.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'col-resize'
}

const doResize = (e: MouseEvent) => {
  doSidebarResize(window.innerWidth - e.clientX)
}

const stopResize = () => {
  stopSidebarResizing()
  window.removeEventListener('mousemove', doResize)
  window.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = 'default'
}
</script>

<template>
    <PlayerView></PlayerView>
    <div
        v-if="playerState.sidebarVisible"
        class="resize-handle"
        :class="{ active: playerState.isSidebarResizing }"
        @mousedown="startResizing"
    ></div>
    <PlaylistView :playlist="playlist" @before-playlist-item-change="async () => await updatePlaylistItem()"></PlaylistView>
</template>

<style lang="css" scoped>
.resize-handle {
  width: 6px;
  cursor: col-resize;
  background: #222;
  transition: background 0.2s;
  z-index: 30;
}

.resize-handle:hover, .resize-handle.active {
  background: var(--accent-blue);
}
</style>
