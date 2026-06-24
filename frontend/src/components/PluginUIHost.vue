<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Modal } from 'bootstrap'
import type { PluginInfo } from '@renderer/data/plugin'

const props = defineProps<{ plugin: PluginInfo }>()
const emit = defineEmits<{
  close: []
}>()

const modalRef = ref<HTMLDivElement>()
const bodyRef = ref<HTMLDivElement>()
const error = ref('')

onMounted(async () => {
  const modal = new Modal(modalRef.value!)
  modal.show()
  modalRef.value!.addEventListener('hidden.bs.modal', () => emit('close'))

  if (props.plugin.ui) {
    try {
      await loadPluginUI()
    } catch (e: any) {
      error.value = e?.message || String(e)
    }
  }
})

async function loadPluginUI() {
  const jsCode = await window.go.main.App.GetPluginFile(props.plugin.id, props.plugin.ui!)

  // Execute in global scope so customElements.define() takes effect
  new Function(jsCode)()

  const tagName = `plugin-${props.plugin.id.replace(/[^a-zA-Z0-9-]/g, '-')}`
  const el = document.createElement(tagName)
  bodyRef.value!.appendChild(el)
}
</script>

<template>
  <div ref="modalRef" class="modal fade" tabindex="-1" data-bs-theme="dark">
    <div class="modal-dialog modal-dialog-centered">
      <div class="modal-content" style="background-color: #1f1f1f; border: 1px solid #333;">
        <div class="modal-header border-secondary">
          <h5 class="modal-title text-white">
            <i class="bi bi-puzzle me-2"></i>{{ plugin.name }}
          </h5>
          <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
        </div>
        <div ref="bodyRef" class="modal-body">
          <div v-if="error" class="text-danger small">{{ error }}</div>
        </div>
      </div>
    </div>
  </div>
</template>
