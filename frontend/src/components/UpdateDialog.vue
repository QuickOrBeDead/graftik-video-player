<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Modal } from 'bootstrap'
import { logger } from '@renderer/utils/logger'

const emit = defineEmits<{
  close: []
}>()

const modalRef = ref<HTMLDivElement>()
const updateInfo = ref<{
  hasUpdate: boolean
  latestVersion: string
  downloadUrl: string
  releaseNotes: string
} | null>(null)
const currentVersion = ref('')
const status = ref('')
const progressPercent = ref(0)
const downloading = ref(false)
const downloadedPath = ref('')
const installing = ref(false)
const success = ref(false)
const error = ref('')
const cleanups: (() => void)[] = []
const includePrerelease = ref(false)

onMounted(async () => {
  const modal = new Modal(modalRef.value!)
  modal.show()
  modalRef.value!.addEventListener('hidden.bs.modal', () => emit('close'))

  cleanups.push(window.runtime.EventsOn('update-download-progress', (data: string) => {
    try {
      const p = JSON.parse(data)
      if (typeof p.percent === 'number') {
        progressPercent.value = Math.round(p.percent)
      }
    } catch (e) {
      error.value = 'Could not parse download progress.'
      logger.error('UpdateDialog: failed to parse download progress data:', e)
    }
  }))

  try {
    currentVersion.value = await (window as any).go.main.App.GetAppVersion() as string
  } catch (e) {
    error.value = 'Could not retrieve app version.'
    logger.error('UpdateDialog: failed to get app version:', e)
  }

  try {
    const prefs = await (window as any).go.internal.PlayerService.GetPreferences()
    if (prefs) {
      includePrerelease.value = !!prefs.includePrereleasesForUpdates
    }
  } catch (e) {
    error.value = 'Could not load preferences.'
    logger.error('UpdateDialog: failed to get preferences:', e)
  }

  await checkForUpdates()
})

onUnmounted(() => {
  cleanups.forEach(fn => fn())
})

async function checkForUpdates() {
  status.value = 'Checking for updates...'
  try {
    const info = await (window as any).go.main.App.CheckForUpdates() as any
    if (info) {
      updateInfo.value = info
      status.value = ''
    } else {
      status.value = 'You have the latest version.'
    }
  } catch (e: any) {
    status.value = 'Error checking for updates.'
    error.value = 'Could not check for updates.'
    logger.error('UpdateDialog: failed to check for updates:', e)
  }
}

async function downloadUpdate() {
  if (!updateInfo.value?.downloadUrl) return
  downloading.value = true
  status.value = 'Downloading...'
  progressPercent.value = 0
  error.value = ''
  try {
    const path = await (window as any).go.main.App.DownloadUpdate(updateInfo.value.downloadUrl) as string
    downloadedPath.value = path
    status.value = 'Download complete.'
    downloading.value = false
  } catch (e: any) {
    status.value = 'Download failed.'
    error.value = 'Could not download update.'
    logger.error('UpdateDialog: failed to download update:', e)
    downloading.value = false
  }
}

async function installUpdate() {
  if (!downloadedPath.value) return
  installing.value = true
  status.value = 'Installing...'
  error.value = ''
  try {
    await (window as any).go.main.App.InstallUpdate(downloadedPath.value)
    success.value = true
    status.value = 'Update installed! Please restart the app.'
    installing.value = false
  } catch (e: any) {
    status.value = 'Installation failed.'
    error.value = 'Could not install update.'
    logger.error('UpdateDialog: failed to install update:', e)
    installing.value = false
  }
}

async function onTogglePrerelease() {
  try {
    await (window as any).go.internal.PlayerService.SavePreferences({ includePrereleasesForUpdates: includePrerelease.value })
  } catch (e) {
    error.value = 'Could not save prerelease preference.'
    logger.error('UpdateDialog: failed to save prerelease preference:', e)
  }
  updateInfo.value = null
  error.value = ''
  await checkForUpdates()
}

function formatReleaseNotes(notes: string): string {
  return notes
}
</script>

<template>
  <div ref="modalRef" class="modal fade" tabindex="-1" data-bs-theme="dark">
    <div class="modal-dialog modal-dialog-centered modal-lg">
      <div class="modal-content" style="background-color: #1f1f1f; border: 1px solid #333;">
        <div class="modal-header border-secondary">
          <h5 class="modal-title text-white">
            <i class="bi bi-arrow-up-circle me-2"></i>Updates
          </h5>
          <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
        </div>
        <div class="modal-body">
          <div v-if="currentVersion" class="mb-3 text-secondary small">
            Current version: <span class="text-white">{{ currentVersion }}</span>
            <span v-if="updateInfo" class="ms-3">
              Latest version: <span class="text-success">{{ updateInfo.latestVersion }}</span>
            </span>
          </div>

          <div class="form-check mb-3">
            <input class="form-check-input" type="checkbox" id="includePrerelease" v-model="includePrerelease" @change="onTogglePrerelease">
            <label class="form-check-label text-white small" for="includePrerelease">Include prerelease versions</label>
          </div>

          <div v-if="updateInfo && updateInfo.hasUpdate">
            <div class="mb-3">
              <label class="form-label text-white small fw-semibold">Release Notes</label>
              <div
                class="p-3 rounded"
                style="background-color: #2a2a2a; border: 1px solid #444; color: whitesmoke; max-height: 300px; overflow-y: auto; white-space: pre-wrap; font-size: 0.85rem;"
              >{{ formatReleaseNotes(updateInfo.releaseNotes) }}</div>
            </div>

            <div v-if="downloading || downloadedPath || installing || success" class="mt-3">
              <div class="d-flex justify-content-between small text-secondary mb-1">
                <span>{{ status || ' ' }}</span>
                <span v-if="downloading">{{ progressPercent }}%</span>
              </div>
              <div v-if="downloading" class="progress" style="height: 6px;">
                <div
                  class="progress-bar progress-bar-striped progress-bar-animated"
                  role="progressbar"
                  :style="{ width: progressPercent + '%' }"
                ></div>
              </div>
            </div>

            <div v-if="error" class="alert alert-danger py-2 mt-2 small" role="alert">
              {{ error }}
            </div>

            <div class="d-flex justify-content-end gap-2 mt-3">
              <button
                v-if="!downloadedPath && !success"
                class="btn btn-primary btn-sm"
                @click="downloadUpdate"
                :disabled="downloading || installing"
              >
                <i v-if="downloading" class="bi bi-arrow-repeat me-1"></i>
                <i v-else class="bi bi-download me-1"></i>
                {{ downloading ? 'Downloading...' : 'Download Update' }}
              </button>
              <button
                v-if="downloadedPath && !success"
                class="btn btn-warning btn-sm"
                @click="installUpdate"
                :disabled="installing"
              >
                <i v-if="installing" class="bi bi-arrow-repeat me-1"></i>
                <i v-else class="bi bi-gear me-1"></i>
                {{ installing ? 'Installing...' : 'Install Update' }}
              </button>
              <button
                v-if="success"
                type="button"
                class="btn btn-success btn-sm"
                data-bs-dismiss="modal"
              >
                <i class="bi bi-check-circle me-1"></i>Done
              </button>
            </div>
          </div>

          <div v-else-if="status && !updateInfo" class="text-center py-4">
            <p class="text-secondary mb-0">{{ status }}</p>
          </div>

          <div v-else class="text-center py-4">
            <i class="bi bi-check-circle text-success" style="font-size: 2rem;"></i>
            <p class="text-secondary mt-2 mb-0">You have the latest version.</p>
          </div>
        </div>
        <div v-if="!updateInfo && !status" class="modal-footer border-secondary">
          <button type="button" class="btn btn-secondary btn-sm" data-bs-dismiss="modal">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>
