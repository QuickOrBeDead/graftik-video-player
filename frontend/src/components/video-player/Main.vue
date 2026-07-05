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
    logger.debug('[SAVE:PLAYLIST-META] currentItem changed — updating playlist.current_item:', { playlistId: playlistState.id, currentItem: newCurrentItem })
    await window.go.internal.PlayerService.UpdatePlaylist(playlistState.id, { current_item: newCurrentItem })
  }
})

let updatePlaylistCurrentItemIntervalId: ReturnType<typeof setInterval>

updatePlaylistCurrentItemIntervalId = setInterval(
  async () => await updatePlaylistItem(),
  10000
)

window.onbeforeunload = () => {
  logger.debug('[SAVE:BEFOREUNLOAD] Page unloading — performing final playlist item save')
  try {
    if (updatePlaylistCurrentItemIntervalId) {
      clearInterval(updatePlaylistCurrentItemIntervalId)
    }

    updatePlaylistItem().catch((e) => logger.error(e))
  } catch (error) {
    logger.error(error)
  }
}

const updatePlaylistItem = async () => {
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
    data = { is_playing: false }
  }

  logger.debug('[SAVE:ITEM] Saving playlist item data via UpdatePlaylistItem:', { itemId: playlistState.currentItem, data })
  await window.go.internal.PlayerService.UpdatePlaylistItem(playlistState.currentItem, data)
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
    <PlaylistView :playlist="playlist" @before-playlist-item-change="updatePlaylistItem"></PlaylistView>
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
