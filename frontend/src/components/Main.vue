<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Playlist, VideoPlayer } from './video-player'
import { PlaylistDto as DbPlaylist } from '@renderer/data/playlist'
import PlaylistsModal from './Playlists.vue'
import NewPlaylistModal from './NewPlaylist.vue'

const playlist = ref<Playlist | null>(null)
const showPlaylistsModal = ref(false)
const showNewPlaylistModal = ref(false)

onMounted(async () => {
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
</script>

<template>
  <video-player :playlist="playlist"></video-player>
  <PlaylistsModal v-if="showPlaylistsModal" @close="showPlaylistsModal = false" />
  <NewPlaylistModal v-if="showNewPlaylistModal" @close="showNewPlaylistModal = false" />
</template>
