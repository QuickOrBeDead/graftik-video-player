<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Modal } from 'bootstrap'
import type { PluginInfo } from '@renderer/data/plugin'
import InstallPluginDialog from './InstallPluginDialog.vue'
import { logger } from '@renderer/utils/logger'

const emit = defineEmits<{
  close: []
  openPlugin: [plugin: PluginInfo, action: string]
}>()

const modalRef = ref<HTMLDivElement>()
const plugins = ref<PluginInfo[]>([])
const showInstallDialog = ref(false)
const error = ref('')

onMounted(async () => {
  await loadPlugins()
  if (modalRef.value) {
    const modal = new Modal(modalRef.value!)
    modal.show()
    modalRef.value!.addEventListener('hidden.bs.modal', () => emit('close'))
  }
})

async function loadPlugins() {
  try {
    plugins.value = (await window.go.main.App.GetPlugins()) as PluginInfo[]
  } catch (err) {
    error.value = 'Could not load plugins.'
    logger.error('PluginPanel: failed to load plugins', 'error', err)
  }
}

function openPlugin(plugin: PluginInfo, action: string) {
  const modal = Modal.getInstance(modalRef.value!)
  if (modal) modal.hide()
  emit('openPlugin', plugin, action)
}

function onInstalled() {
  showInstallDialog.value = false
  loadPlugins()
}
</script>

<template>
  <div ref="modalRef" class="modal fade" tabindex="-1" data-bs-theme="dark">
    <div class="modal-dialog modal-dialog-centered">
      <div class="modal-content" style="background-color: #1f1f1f; border: 1px solid #333;">
        <div class="modal-header border-secondary">
          <h5 class="modal-title text-white">
            <i class="bi bi-puzzle me-2"></i>Plugins
          </h5>
          <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
        </div>
        <div class="modal-body p-0">
          <ul v-if="plugins.length > 0" class="list-group list-group-flush">
            <li
              v-for="p in plugins"
              :key="p.id"
              class="list-group-item"
              style="background-color: #1f1f1f; border-color: #333; color: whitesmoke;"
            >
              <div class="d-flex justify-content-between align-items-center mb-1">
                <strong>{{ p.name }}</strong>
                <span class="badge" :class="p.status === 'active' ? 'bg-success' : 'bg-secondary'">
                  {{ p.status }}
                </span>
              </div>
              <div class="small text-secondary mb-1">v{{ p.version }}</div>
              <div class="d-flex gap-1 flex-wrap">
                <button
                  v-for="entry in p.menu"
                  :key="entry.action"
                  class="btn btn-sm btn-outline-primary"
                  @click="openPlugin(p, entry.action)"
                >
                  {{ entry.label }}
                </button>
              </div>
            </li>
          </ul>

          <div v-else class="text-center text-secondary p-4">
            <i class="bi bi-puzzle fs-1 d-block mb-2"></i>
            <p class="mb-0">No plugins registered.</p>
          </div>
          <div v-if="error" class="text-danger small px-3 pb-2">{{ error }}</div>
        </div>
        <div class="modal-footer border-secondary">
          <button class="btn btn-sm btn-outline-success" @click="showInstallDialog = true">
            <i class="bi bi-plus-circle me-1"></i>Add Plugin
          </button>
          <button type="button" class="btn btn-secondary btn-sm" data-bs-dismiss="modal">Close</button>
        </div>
      </div>
    </div>
  </div>

  <InstallPluginDialog
    v-if="showInstallDialog"
    @close="showInstallDialog = false"
    @installed="onInstalled"
  />
</template>
