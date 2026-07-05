<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { usePlayer } from './video-player/composables/usePlayer'
import { usePlaylist } from './video-player/composables/usePlaylist'
import { Playlist, VideoPlayer } from './video-player'
import { PlaylistDto as DbPlaylist } from '@renderer/data/playlist'
import PlaylistsModal from './Playlists.vue'
import NewPlaylistModal from './NewPlaylist.vue'
import PluginPanel from './PluginPanel.vue'
import PluginUIHost from './PluginUIHost.vue'
import UpdateDialog from './UpdateDialog.vue'
import AboutDialog from './AboutDialog.vue'
import type { PluginInfo } from '@renderer/data/plugin'
import { logger } from '@renderer/utils/logger'

const { playerState, applyPreferences } = usePlayer()
const { playlistState } = usePlaylist()

const playlist = ref<Playlist | null>(null)
const showPlaylistsModal = ref(false)
const showNewPlaylistModal = ref(false)
const showPluginPanel = ref(false)
const activePluginUI = ref<PluginInfo | null>(null)
const showUpdateDialog = ref(false)
const showAboutDialog = ref(false)
const updateAvailable = ref('')

let saveTimer: ReturnType<typeof setTimeout> | null = null

function scheduleSave(data: Record<string, any>) {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    window.go.internal.PlayerService.SavePreferences(data)
  }, 500)
}

onMounted(async () => {
  // Load preferences first so isPlaying state is known before playlist loads
  try {
    const prefs = await window.go.internal.PlayerService.GetPreferences()
    if (prefs) {
      applyPreferences(prefs)
    }
  } catch (e) {
    logger.error('Load preferences error:', e)
  }

  await loadPlaylist()

  window.runtime.EventsOn('load-current-playlist', (p: unknown) => {
    if (p) setPlaylist(p as DbPlaylist)
  })

  window.runtime.EventsOn('load-playlist-name', async () => {
    if (playlist.value) {
      playlist.value.name = (await window.go.internal.PlayerService.GetPlaylistName(playlist.value.id)) as string
    }
  })

  window.runtime.EventsOn('load-playlist', async () => {
    await loadPlaylist()
  })

  window.runtime.EventsOn('open-choose-playlist', () => {
    showPlaylistsModal.value = true
  })

  window.runtime.EventsOn('open-new-playlist', () => {
    showNewPlaylistModal.value = true
  })

  window.runtime.EventsOn('open-plugin-panel', () => {
    showPluginPanel.value = true
  })

  window.runtime.EventsOn('update-available', (version: string) => {
    updateAvailable.value = version
  })

  window.runtime.EventsOn('check-for-updates', () => {
    showUpdateDialog.value = true
  })

  window.runtime.EventsOn('show-about', () => {
    showAboutDialog.value = true
  })
})

onUnmounted(() => {
  if (saveTimer) clearTimeout(saveTimer)
})

watch(() => playerState.shuffle, (v) => scheduleSave({ shuffle: v }))
watch(() => playerState.repeat, (v) => scheduleSave({ repeatMode: v }))
watch(() => playerState.volumeLevel, (v) => scheduleSave({ volumeLevel: v }))
watch(() => playerState.playbackRate, (v) => scheduleSave({ playbackRate: v }))
watch(() => playerState.sidebarVisible, (v) => scheduleSave({ sidebarVisible: v }))
watch(() => playerState.sidebarWidth, (v) => scheduleSave({ sidebarWidth: v }))
watch(() => playerState.isPlaying, (v) => scheduleSave({ isPlaying: v }))
watch(() => playlistState.currentItem, (v) => {
  if (v) scheduleSave({ lastPlayedItem: v })
})

const loadPlaylist = async () => {
  const dbPlaylist = await window.go.internal.PlayerService.GetCurrentPlaylist() as DbPlaylist | null
  if (dbPlaylist) {
    setPlaylist(dbPlaylist)
  }
}

const setPlaylist = (dbPlaylist: DbPlaylist) => {
  if (!dbPlaylist) return
  playlist.value = {
    id: dbPlaylist.id,
    name: dbPlaylist.name,
    items: (dbPlaylist.items || []).map(x => {
      return {
        id: x.id,
        playlistId: x.playlistId,
        title: x.title,
        path: x.path,
        elapsedTime: x.elapsedTime,
        duration: x.duration,
        progressPercent: x.progressPercent,
        isPlaying: x.isPlaying,
        orderIndex: x.orderIndex
      }
    }),
    currentItem: dbPlaylist.currentItem,
    currentPlaylistItem: dbPlaylist.currentPlaylistItem,
    currentVolume: dbPlaylist.currentVolume,
    shuffledDeck: []
  }
}

function onOpenPlugin(plugin: PluginInfo, action: string) {
  if (plugin.ui) {
    activePluginUI.value = plugin
  }
}
</script>

<template>
  <video-player :playlist="playlist"></video-player>
  <div
    v-if="updateAvailable"
    class="update-badge"
    @click="showUpdateDialog = true"
    title="Update available"
  >
    <i class="bi bi-arrow-up-circle-fill"></i>
  </div>
  <PlaylistsModal v-if="showPlaylistsModal" @close="showPlaylistsModal = false" />
  <NewPlaylistModal v-if="showNewPlaylistModal" @close="showNewPlaylistModal = false" />
  <PluginPanel v-if="showPluginPanel" @close="showPluginPanel = false" @openPlugin="onOpenPlugin" />
  <PluginUIHost v-if="activePluginUI" :plugin="activePluginUI" @close="activePluginUI = null" />
  <UpdateDialog v-if="showUpdateDialog" @close="showUpdateDialog = false" />
  <AboutDialog v-if="showAboutDialog" @close="showAboutDialog = false" />
</template>

<style scoped>
.update-badge {
  position: fixed;
  bottom: 16px;
  right: 16px;
  z-index: 9999;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background-color: #0d6efd;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  font-size: 1.25rem;
  box-shadow: 0 2px 8px rgba(0,0,0,0.4);
  transition: transform 0.15s;
}
.update-badge:hover {
  transform: scale(1.1);
  background-color: #0b5ed7;
}
</style>
