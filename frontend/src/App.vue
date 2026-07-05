<template>
  <Main />
  <DevConsole v-if="showDevConsole" @close="showDevConsole = false" />
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Main from './components/Main.vue'
import DevConsole from './components/DevConsole.vue'
import { logger } from './utils/logger'
import Mousetrap from 'mousetrap'

const showDevConsole = ref(false)

onMounted(() => {
  window.runtime.EventsOn('log', (data: unknown) => {
    const entry = data as Record<string, unknown>
    logger.fromBackend({
      level: String(entry.level || 'info'),
      message: String(entry.message || ''),
      time: String(entry.time || ''),
      source: String(entry.source || ''),
      attrs: entry.attrs as Record<string, unknown> | undefined,
    })
  })

  Mousetrap.bind('ctrl+shift+d', () => {
    showDevConsole.value = !showDevConsole.value
  })

  window.runtime.EventsEmit('frontend-ready')
})
</script>
