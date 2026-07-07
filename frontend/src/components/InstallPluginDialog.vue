<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Modal } from 'bootstrap'
import { logger } from '@renderer/utils/logger'

const emit = defineEmits<{
  close: []
  installed: []
}>()

const modalRef = ref<HTMLDivElement>()
const tab = ref<'url' | 'file'>('url')
const url = ref('')
const status = ref('')
const progressPercent = ref(0)
const installing = ref(false)
const success = ref(false)
const errors = ref<string[]>([])
const cleanups: (() => void)[] = []

onMounted(() => {
  const modal = new Modal(modalRef.value!)
  modal.show()
  modalRef.value!.addEventListener('hidden.bs.modal', () => emit('close'))

  cleanups.push(window.runtime.EventsOn('plugin-install-progress', (data: string) => {
    try {
      const p = JSON.parse(data)
      if (typeof p.percent === 'number') {
        progressPercent.value = Math.round(p.percent)
      }
    } catch (e) {
      errors.value.push('Failed to read install progress.')
      logger.error('InstallPluginDialog: failed to parse install progress:', e)
    }
  }))

  cleanups.push(window.runtime.EventsOn('plugin-install-log', (data: string) => {
    try {
      const p = JSON.parse(data)
      if (p.message) status.value = p.message
    } catch (e) {
      errors.value.push('Failed to read install log.')
      logger.error('InstallPluginDialog: failed to parse install log:', e)
    }
  }))

  cleanups.push(window.runtime.EventsOn('plugin-install-complete', () => {
    installing.value = false
    success.value = true
    status.value = 'Plugin installed!'
  }))
})

onUnmounted(() => {
  cleanups.forEach(fn => fn())
})

async function installFromURL() {
  if (!url.value) return
  installing.value = true
  status.value = 'Connecting...'
  progressPercent.value = 0
  success.value = false
  try {
    await (window as any).go.main.App.InstallPluginFromURL(url.value)
    emit('installed')
  } catch (e: any) {
    errors.value.push('Failed to install plugin from URL.')
    logger.error('InstallPluginDialog: failed to install from URL:', e)
    installing.value = false
  }
}

async function pickAndInstall() {
  installing.value = true
  status.value = 'Reading file...'
  progressPercent.value = 0
  success.value = false
  try {
    const filePath = await (window as any).go.main.App.PickPluginFile()
    if (!filePath) {
      installing.value = false
      return
    }
    await (window as any).go.main.App.InstallPluginFromFile(filePath)
    emit('installed')
  } catch (e: any) {
    errors.value.push('Failed to install plugin from file.')
    logger.error('InstallPluginDialog: failed to install from file:', e)
    installing.value = false
  }
}
</script>

<template>
  <div ref="modalRef" class="modal fade" tabindex="-1" data-bs-theme="dark">
    <div class="modal-dialog modal-dialog-centered">
      <div class="modal-content" style="background-color: #1f1f1f; border: 1px solid #333;">
        <div class="modal-header border-secondary">
          <h5 class="modal-title text-white">
            <i class="bi bi-plus-circle me-2"></i>Install Plugin
          </h5>
          <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
        </div>
        <div class="modal-body">
          <ul class="nav nav-tabs border-secondary mb-3">
            <li class="nav-item">
              <button
                class="nav-link"
                :class="tab === 'url' ? 'active text-white' : 'text-secondary'"
                style="background-color: tab === 'url' ? '#2a2a2a' : 'transparent'; border-color: #444;"
                @click="tab = 'url'"
                :disabled="installing"
              >From URL</button>
            </li>
            <li class="nav-item">
              <button
                class="nav-link"
                :class="tab === 'file' ? 'active text-white' : 'text-secondary'"
                style="background-color: tab === 'file' ? '#2a2a2a' : 'transparent'; border-color: #444;"
                @click="tab = 'file'"
                :disabled="installing"
              >From File</button>
            </li>
          </ul>

          <div v-if="tab === 'url'">
            <div class="mb-3">
              <label class="form-label text-white small">Plugin ZIP URL</label>
              <input
                v-model="url"
                type="text"
                class="form-control form-control-sm"
                placeholder="https://example.com/plugin.zip"
                :disabled="installing"
                style="background-color: #2a2a2a; border-color: #444; color: whitesmoke;"
              />
            </div>
            <div class="d-flex justify-content-end">
              <button
                class="btn btn-primary btn-sm"
                @click="installFromURL"
                :disabled="installing || !url"
              >
                <i v-if="installing" class="bi bi-arrow-repeat me-1"></i>
                <i v-else class="bi bi-download me-1"></i>
                {{ installing ? 'Installing...' : 'Download & Install' }}
              </button>
            </div>
          </div>

          <div v-if="tab === 'file'">
            <p class="text-secondary small mb-3">
              Select a plugin ZIP file from your computer.
            </p>
            <div class="d-flex justify-content-end">
              <button
                class="btn btn-primary btn-sm"
                @click="pickAndInstall"
                :disabled="installing"
              >
                <i v-if="installing" class="bi bi-arrow-repeat me-1"></i>
                <i v-else class="bi bi-folder2-open me-1"></i>
                {{ installing ? 'Installing...' : 'Select & Install' }}
              </button>
            </div>
          </div>

          <div v-if="installing || success || status" class="mt-3">
            <div class="d-flex justify-content-between small text-secondary mb-1">
              <span>{{ status || ' ' }}</span>
              <span v-if="installing">{{ progressPercent }}%</span>
            </div>
            <div v-if="installing" class="progress" style="height: 6px;">
              <div
                class="progress-bar progress-bar-striped progress-bar-animated"
                role="progressbar"
                :style="{ width: progressPercent + '%' }"
              ></div>
            </div>
          </div>
          <div v-for="(err, i) in errors" :key="i" class="text-danger small mt-1">{{ err }}</div>
        </div>
        <div class="modal-footer border-secondary">
          <button v-if="success" type="button" class="btn btn-success btn-sm" data-bs-dismiss="modal">
            <i class="bi bi-check-circle me-1"></i>Done
          </button>
          <button v-else type="button" class="btn btn-secondary btn-sm" data-bs-dismiss="modal" :disabled="installing">
            Cancel
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
