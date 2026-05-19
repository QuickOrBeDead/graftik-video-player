<script setup lang="ts">
import { watch } from 'vue'
import { usePlayer } from './composables/usePlayer'
import { usePlaylist } from './composables/usePlaylist'
import PlayerView from './Player.vue'
import PlaylistView from './Playlist.vue'
import { Playlist } from './types'

const { playerState, progressPercent, doSidebarResize, startSidebarResizing, stopSidebarResizing } = usePlayer()
const { playlistState } = usePlaylist()
const { playlist } = defineProps<{
  playlist: Playlist | null
}>()

watch(() => playlistState.currentItem, async (newCurrentItem, oldCurrentItem) => {
  if (playlistState.id && newCurrentItem && oldCurrentItem !== newCurrentItem) {
    await window.electron.ipcRenderer.invoke('updatePlaylist', playlistState.id, { currentItem: newCurrentItem  })
  }
})

let updatePlaylistCurrentItemIntervalId: NodeJS.Timeout

updatePlaylistCurrentItemIntervalId = setInterval(
  async () => await updatePlaylistItem(),
  10000
)

window.onbeforeunload = () => {
  try {
    if (updatePlaylistCurrentItemIntervalId) {
      clearInterval(updatePlaylistCurrentItemIntervalId)
    }

    updatePlaylistItem().catch((e) => console.error(e))
  } catch (error) {
    console.error(error)
  }
}

const updatePlaylistItem = async () => {
  if (!playlistState.currentItem) {
    return
  }

  let data: any
  if (playerState.isPlaying) {
    data = {
      currentTime: playerState.currentTime,
      duration: playerState.duration,
      isPlaying: true,
      progressPercent: progressPercent.value,
      lastWatched: Date.now()
    }
  } else {
    data = { isPlaying: false }
  }

  await window.electron.ipcRenderer.invoke('updatePlaylistItem', playlistState.currentItem, data)
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
    <PlaylistView :playlist="playlist"></PlaylistView>
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
