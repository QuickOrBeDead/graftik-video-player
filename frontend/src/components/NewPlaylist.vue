<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Modal } from 'bootstrap'

const emit = defineEmits<{ close: [] }>()
const name = ref<string>()
const modalRef = ref<HTMLDivElement>()

onMounted(() => {
  if (modalRef.value) {
    const modal = new Modal(modalRef.value)
    modal.show()
    modalRef.value.addEventListener('hidden.bs.modal', () => emit('close'))
  }
})

async function add() {
  await window.go.internal.PlayerService.AddPlaylist(name.value ?? '')
}
</script>

<template>
  <div ref="modalRef" class="modal fade" tabindex="-1" data-bs-theme="dark">
    <div class="modal-dialog modal-sm modal-dialog-centered">
      <div class="modal-content" style="background-color: #1f1f1f; border: 1px solid #333;">
        <div class="modal-header border-secondary">
          <h5 class="modal-title text-white">
            <i class="bi bi-file-earmark-plus me-2"></i>New Playlist
          </h5>
          <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
        </div>
        <div class="modal-body">
          <form @submit.prevent="add">
            <input v-model="name" type="text" class="form-control bg-dark text-white border-secondary" placeholder="Name" />
          </form>
        </div>
        <div class="modal-footer border-secondary">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
          <button type="button" class="btn btn-primary" @click="async () => await add()">Add</button>
        </div>
      </div>
    </div>
  </div>
</template>
