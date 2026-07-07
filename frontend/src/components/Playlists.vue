<script setup lang="ts">
import { onMounted, ref, inject } from 'vue'
import { Modal } from 'bootstrap'
import { logger } from '@renderer/utils/logger'

const emit = defineEmits<{ close: [] }>()

const modalRef = ref<HTMLDivElement>()
const playlists = ref<{ name: string; id: string }[]>()
const currentPlaylist = ref<{ name: string; id: string }>({ id: '', name: '' })
const editModal = ref<HTMLDivElement>()
const deleteModal = ref<HTMLDivElement>()
const showErrorModal = inject('showErrorModal') as (msg: string) => void

onMounted(async () => {
  await loadPlaylists()

  if (modalRef.value) {
    const modal = new Modal(modalRef.value!)
    modal.show()
    modalRef.value!.addEventListener('hidden.bs.modal', () => emit('close'))
  }
})

async function loadPlaylists() {
  try {
    playlists.value = (await window.go.internal.PlayerService.GetPlaylists()) as { name: string; id: string }[]
  } catch (err) {
    showErrorModal('Could not load playlists.')
    logger.error('Playlists: failed to load playlists:', err)
  }
}

function confirmDelete(item: { name: string; id: string }) {
  currentPlaylist.value = item
  if (deleteModal.value) {
    const modalInstance = new Modal(deleteModal.value!)
    modalInstance.show()
  }
}

function hideDeleteModal() {
  if (deleteModal.value) {
    const modalInstance = Modal.getInstance(deleteModal.value!)
    if (modalInstance) {
      modalInstance.hide()
    }
  }
}

function showEditModal(item: { name: string; id: string }) {
  currentPlaylist.value = item
  if (editModal.value) {
    const modalInstance = new Modal(editModal.value!)
    modalInstance.show()
  }
}

function hideEditModal() {
  if (editModal.value) {
    const modalInstance = Modal.getInstance(editModal.value!)
    if (modalInstance) {
      modalInstance.hide()
    }
  }
}

async function savePlaylist() {
  try {
    await window.go.internal.PlayerService.UpdatePlaylistName(currentPlaylist.value.id, currentPlaylist.value.name)
    await loadPlaylists()
    hideEditModal()
  } catch (err) {
    showErrorModal('Could not save playlist.')
    logger.error('Playlists: failed to save playlist:', err)
  }
}

async function deletePlaylist() {
  try {
    await window.go.internal.PlayerService.DeletePlaylist(currentPlaylist.value.id)
    await loadPlaylists()
  } catch (err) {
    showErrorModal('Could not delete playlist.')
    logger.error('Playlists: failed to delete playlist:', err)
  }

  hideDeleteModal()
}

async function selectPlaylist(id: string) {
  try {
    await window.go.internal.PlayerService.SelectPlaylist(id)
  } catch (err) {
    showErrorModal('Could not select playlist.')
    logger.error('Playlists: failed to select playlist:', err)
  }
  if (modalRef.value) {
    const modal = Modal.getInstance(modalRef.value!)
    if (modal) modal.hide()
  }
}
</script>

<template>
  <div ref="modalRef" class="modal fade" tabindex="-1" data-bs-theme="dark">
    <div class="modal-dialog modal-dialog-centered">
      <div class="modal-content" style="background-color: #1f1f1f; border: 1px solid #333;">
        <div class="modal-header border-secondary">
          <h5 class="modal-title text-white">
            <i class="bi bi-list-stars me-2"></i>Playlists
          </h5>
          <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
        </div>
        <div class="modal-body p-0">
          <ul class="list-group list-group-flush">
            <li
              v-for="item in playlists"
              :key="item.id"
              class="list-group-item d-flex justify-content-between align-items-center"
              style="background-color: #1f1f1f; border-color: #333; color: whitesmoke;"
            >
              <a class="text-decoration-none" style="color: whitesmoke; cursor: pointer; flex-grow: 1;" @click="async () => await selectPlaylist(item.id)">
                <span>{{ item.name ? item.name : '[Unnamed]' }}</span>
              </a>
              <div>
                <a class="text-decoration-none me-2" style="color: whitesmoke;" @click="() => showEditModal(item)">
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

  <div ref="editModal" class="modal fade" tabindex="-1" data-bs-theme="dark">
    <div class="modal-dialog modal-sm modal-dialog-centered">
      <div class="modal-content" style="background-color: #1f1f1f;">
        <div class="modal-header border-secondary">
          <h5 class="modal-title text-white fs-6">Edit Playlist</h5>
          <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
        </div>
        <div class="modal-body text-white">
          <div class="mb-3">
            <label for="title" class="form-label">Name</label>
            <input v-model="currentPlaylist.name" type="text" class="form-control bg-dark text-white border-secondary" required />
          </div>
        </div>
        <div class="modal-footer border-secondary">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
          <button type="button" class="btn btn-primary" @click="async () => await savePlaylist()">Save</button>
        </div>
      </div>
    </div>
  </div>

  <div ref="deleteModal" class="modal fade" tabindex="-1" data-bs-theme="dark">
    <div class="modal-dialog modal-dialog-centered">
      <div class="modal-content" style="background-color: #1f1f1f;">
        <div class="modal-header border-secondary">
          <h5 class="modal-title text-white fs-6">Confirm Delete</h5>
          <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
        </div>
        <div class="modal-body text-white">
          Are you sure you want to delete '{{ currentPlaylist.name }}' playlist?
        </div>
        <div class="modal-footer border-secondary">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
          <button type="button" class="btn btn-danger" @click="deletePlaylist">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
