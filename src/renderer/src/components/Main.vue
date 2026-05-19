<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Playlist, VideoPlayer } from './video-player'
import { PlaylistDto as DbPlaylist } from '@renderer/data/playlist'

const playlist = ref<Playlist | null>(null)

onMounted(async () => {
  await loadPlaylist()
})

const loadPlaylist =  async () => {
  const dbPlaylist = await window.electron.ipcRenderer.invoke('getCurrentPlaylist', null) as DbPlaylist
  setPlaylist(dbPlaylist)
}

const setPlaylist = (dbPlaylist: DbPlaylist) => {
  playlist.value = {
    id: dbPlaylist.id,
    name: dbPlaylist.name,
    items: dbPlaylist.items.map(x => {
      return {
        id: x.id,
        playlistId: x.playlistId,
        title: x.title,
        path: x.path,
        currentTime: x.currentTime,
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

window.electron.ipcRenderer.on('load-current-playlist', async (_, p: DbPlaylist) => {
  setPlaylist(p)
})

window.electron.ipcRenderer.on('load-playlist-name', async () => {
  if (playlist.value) {
    playlist.value.name = (await window.electron.ipcRenderer.invoke(
      'getPlaylistName',
      playlist.value.id
    )) as string
  }
})

window.electron.ipcRenderer.on('load-playlist', async () => {
  await loadPlaylist()
})
</script>

<template>
  <video-player :playlist="playlist"></video-player>
</template>
