<script setup lang="ts">
import { onMounted, ref } from 'vue'
import TitleBar from './TitleBar.vue'
import { Modal } from 'bootstrap'

const playlists = ref<{ name: string; id: string }[]>()
const currentPlaylist = ref<{ name: string; id: string }>({ id: '', name: '' })
const editModal = ref(null)
const deleteModal = ref(null)

onMounted(async () => {
  await loadPlaylists()
})

async function loadPlaylists() {
  playlists.value = (await window.electron.ipcRenderer.invoke('getPlaylists', null)) as {
    name: string
    id: string
  }[]
}

function confirmDelete(item: { name: string; id: string }) {
  currentPlaylist.value = item
  const modalInstance = new Modal(deleteModal.value)
  modalInstance.show()
}

function hideDeleteModal() {
  const modalInstance = Modal.getInstance(deleteModal.value)
  if (modalInstance) {
    modalInstance.hide()
  }
}

function showEditModal(item: { name: string; id: string }) {
  currentPlaylist.value = item
  const modalInstance = new Modal(editModal.value)
  modalInstance.show()
}

function hideEditModal() {
  const modalInstance = Modal.getInstance(editModal.value)
  if (modalInstance) {
    modalInstance.hide()
  }
}

async function savePlaylist() {
  await window.electron.ipcRenderer.invoke('updatePlaylistName', {
    id: currentPlaylist.value?.id,
    name: currentPlaylist.value?.name
  })
  await loadPlaylists()

  hideEditModal()
}

async function deletePlaylist() {
  await window.electron.ipcRenderer.invoke('deletePlaylist', currentPlaylist.value.id)
  await loadPlaylists()

  hideDeleteModal()
}

async function selectPlaylist(id: string) {
  await window.electron.ipcRenderer.invoke('selectPlaylist', id)
}
</script>
<template>
  <div class="popup-layout">
    <title-bar title="Playlists">
      <template #icon><i class="bi bi-list-stars"></i></template>
    </title-bar>
    <div class="container">
    <div class="row">
      <div class="col">
        <div id="playlists" class="card" style="height: 300px">
          <div class="overflow-auto">
            <ul class="list-group list-group-flush">
              <li
                v-for="item in playlists"
                :key="item.id"
                class="list-group-item d-flex justify-content-between align-items-center"
              >
                <a @click="async () => await selectPlaylist(item.id)">
                  <span>{{ item.name ? item.name : '[Unnamed]' }}</span>
                </a>
                <div>
                  <a class="text-decoration-none me-2" @click="() => showEditModal(item)">
                    <i class="bi bi-pencil-square"></i>
                  </a>
                  <a class="text-decoration-none text-danger" @click="() => confirmDelete(item)">
                    <i class="bi bi-trash"></i>
                  </a>
                </div>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  </div>
  </div>
  <div ref="editModal" class="modal" tabindex="-1" data-bs-theme="dark">
    <div class="modal-dialog modal-sm">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title text-white fs-6">Edit Playlist</h5>
          <button
            type="button"
            class="btn-close"
            data-bs-dismiss="modal"
            aria-label="Close"
          ></button>
        </div>
        <div class="modal-body text-white">
          <div class="mb-3">
            <label for="title" class="form-label">Name</label>
            <input v-model="currentPlaylist.name" type="text" class="form-control" required />
          </div>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
          <button type="button" class="btn btn-primary" @click="async () => await savePlaylist()">
            Save
          </button>
        </div>
      </div>
    </div>
  </div>
  <div
    ref="deleteModal"
    class="modal fade"
    tabindex="-1"
    aria-labelledby="deleteModalLabel"
    aria-hidden="true"
    data-bs-theme="dark"
  >
    <div class="modal-dialog">
      <div class="modal-content">
        <div class="modal-header">
          <h5 id="deleteModalLabel" class="modal-title text-white fs-6">Confirm Delete</h5>
          <button
            type="button"
            class="btn-close"
            aria-label="Close"
            data-bs-dismiss="modal"
          ></button>
        </div>
        <div class="modal-body text-white">
          Are you sure you want to delete '{{ currentPlaylist.name }}' playlist?
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
          <button type="button" class="btn btn-danger" @click="deletePlaylist">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
<style scoped>
.card#playlists {
  background-color: #1f1f1f;
  border: none;
  border-radius: 1rem;
}
.card#playlists .list-group-item {
  background-color: #1f1f1f;
  border: none;
  border-bottom: 1px solid #444444;
}
.card#playlists .list-group-item a {
  color: whitesmoke;
  text-decoration: none;
  cursor: pointer;
}
.card#playlists .list-group-item a:hover {
  color: rgb(114, 125, 231);
}
.card#playlists .list-group-item:last-child {
  border-bottom-left-radius: 1rem;
  border-bottom-right-radius: 1rem;
}
</style>
