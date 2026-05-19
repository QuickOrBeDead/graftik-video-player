<script setup lang="ts">
import { ref } from 'vue'
import TitleBar from './TitleBar.vue'

const name = ref<string>()

async function add() {
  await window.electron.ipcRenderer.invoke('addPlaylist', name.value)
}
</script>
<template>
  <div class="popup-layout">
    <title-bar title="New Playlist">
      <template #icon><i class="bi bi-file-earmark-plus"></i></template>
    </title-bar>
    <div class="container" data-bs-theme="dark">
      <form class="row g-3 mt-1 mb-1">
        <div class="col-auto">
          <input v-model="name" type="text" class="form-control" placeholder="Name" />
        </div>
        <div class="col-auto">
          <button type="button" class="btn btn-primary" @click="async () => await add()">Add</button>
        </div>
      </form>
    </div>
  </div>
</template>
