<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Modal } from 'bootstrap'
import { BrowserOpenURL } from '@wailsjs/runtime/runtime'

const emit = defineEmits<{
  close: []
}>()

const modalRef = ref<HTMLDivElement>()
const appVersion = ref('')

const GITHUB_URL = 'https://github.com/QuickOrBeDead/graftik-video-player'
const GITHUB_PROFILE = 'https://github.com/QuickOrBeDead'
const LICENSE_URL = GITHUB_URL + '/blob/main/LICENSE'
const ISSUES_URL = GITHUB_URL + '/issues'

onMounted(async () => {
  const modal = new Modal(modalRef.value!)
  modal.show()
  modalRef.value!.addEventListener('hidden.bs.modal', () => emit('close'))

  appVersion.value = await (window as any).go.main.App.GetAppVersion() as string
})

function openLink(url: string) {
  BrowserOpenURL(url)
}
</script>

<template>
  <div ref="modalRef" class="modal fade" tabindex="-1" data-bs-theme="dark">
    <div class="modal-dialog modal-dialog-centered">
      <div class="modal-content" style="background-color: #1f1f1f; border: 1px solid #333;">
        <div class="modal-header border-secondary py-2">
          <h6 class="modal-title text-white">
            <i class="bi bi-info-circle me-2"></i>About Graftik Video Player
          </h6>
          <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
        </div>
        <div class="modal-body pt-3 pb-2">
          <div class="d-flex align-items-start gap-3 mb-2">
            <img src="/app-icon.png" class="flex-shrink-0" style="width: 3.5rem; height: 3.5rem;">
            <div class="min-width-0">
              <div class="d-flex align-items-baseline gap-2 flex-wrap">
                <span class="text-white fw-semibold" style="font-size: 1rem;">Graftik Video Player</span>
                <span class="text-secondary" style="font-size: 0.8rem;">v{{ appVersion }}</span>
              </div>
              <p class="d-flex flex-wrap gap-3 small mb-0" style="font-size: 0.8rem;">
                <a href="#" class="text-info text-decoration-none" @click.prevent="openLink(GITHUB_URL)">
                  <i class="bi bi-github"></i> GitHub
                </a>
                <a href="#" class="text-info text-decoration-none" @click.prevent="openLink(ISSUES_URL)">
                  <i class="bi bi-bug"></i> Report Bug
                </a>
                <a href="#" class="text-info text-decoration-none" @click.prevent="openLink(LICENSE_URL)">
                  <i class="bi bi-file-text"></i> MIT License
                </a>
              </p>
              <div class="small text-secondary" style="font-size: 0.8rem;">
                Copyright &copy; 2026
              </div>
            </div>
          </div>

          <hr class="border-secondary my-2">

          <div class="text-secondary fw-semibold mb-1" style="font-size: 0.8rem;">Libraries</div>
          <ul class="text-secondary ps-3 mb-0" style="list-style: disc; font-size: 0.75rem;">
            <li>Wails v2.12.0</li>
            <li>SQLite 1.50.1</li>
            <li>FFmpeg 7.1</li>
            <li>Vue 3.5.38</li>
            <li>Bootstrap 5.3.8</li>
            <li>HLS.js 1.6.16</li>
          </ul>

          <div class="text-secondary fw-semibold mb-1 mt-3" style="font-size: 0.8rem;">Developer</div>
          <div class="text-secondary" style="font-size: 0.75rem;">
            <i class="bi bi-person me-1"></i><a href="#" class="text-info text-decoration-none" @click.prevent="openLink(GITHUB_PROFILE)">Bora Akgün</a>
          </div>
        </div>
        <div class="modal-footer border-secondary py-2">
          <button type="button" class="btn btn-secondary btn-sm" data-bs-dismiss="modal">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>
