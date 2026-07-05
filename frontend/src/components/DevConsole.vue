<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, shallowRef, watch, reactive } from 'vue'
import { LogEntry, logger, type LogLevel } from '@renderer/utils/logger'

const emit = defineEmits<{ close: [] }>()

const allLevels: LogLevel[] = ['trace', 'debug', 'info', 'warn', 'error']
const selectedLevels = reactive<Record<LogLevel, boolean>>({
  trace: true,
  debug: true,
  info: true,
  warn: true,
  error: true,
})
const filterOpen = ref(false)

const allSelected = computed(() => allLevels.every(l => selectedLevels[l]))
const filterSummary = computed(() => {
  if (allSelected.value) return 'all'
  const count = allLevels.filter(l => selectedLevels[l]).length
  return count === 0 ? 'none' : `${count}/5`
})
const visible = ref(true)
const autoScroll = ref(true)
const containerRef = ref<HTMLDivElement>()

const entriesRaw = shallowRef<Array<LogEntry>>([]);
const maxBuffer = ref(logger.maxBuffer)

watch(maxBuffer, v => { logger.maxBuffer = v })
function formatTime(d: Date): string {
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  const ms = String(d.getMilliseconds()).padStart(3, '0')
  return `${hh}:${mm}:${ss}.${ms}`
}

function formatAttrValue(v: unknown): string {
  if (v === null) return 'null'
  if (v === undefined) return 'undefined'
  if (typeof v === 'string') return v
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  try { return JSON.stringify(v) } catch { return String(v) }
}

const entries = computed(() => {
  const all = entriesRaw.value
  if (allSelected.value) return all
  return all.filter(e => selectedLevels[e.level])
})

function levelBadgeClass(level: LogLevel): string {
  switch (level) {
    case 'trace': return 'bg-secondary'
    case 'debug': return 'bg-secondary'
    case 'info':  return 'bg-info text-dark'
    case 'warn':  return 'bg-warning text-dark'
    case 'error': return 'bg-danger'
  }
}

function levelIcon(level: LogLevel): string {
  switch (level) {
    case 'trace': return 'bi bi-arrow-return-right'
    case 'debug': return 'bi bi-bug'
    case 'info':  return 'bi bi-info-circle'
    case 'warn':  return 'bi bi-exclamation-triangle'
    case 'error': return 'bi bi-x-circle'
  }
}

let unsubscribeFunc: () => void

onMounted(() => {
  unsubscribeFunc = logger.subscribe(() => {
    loadEntries()
    scrollDown()
  })

  loadEntries()
  scrollDown()
  document.addEventListener('click', onDocClick)
})

onUnmounted(() => {
  if (unsubscribeFunc) unsubscribeFunc()
  document.removeEventListener('click', onDocClick)
})

function onDocClick() {
  filterOpen.value = false
}

function scrollDown() {
  if (autoScroll.value && containerRef.value) {
    nextTick(() => {
      containerRef.value!.scrollTop = containerRef.value!.scrollHeight
    })
  }
}

function onCopyAll() {
  const text = logger.getBuffer()
    .map(e => {
      let line = `[${e.time.toISOString()}] [${e.level.toUpperCase()}] ${e.message}`
      if (e.attrs && Object.keys(e.attrs).length > 0) {
        for (const [k, v] of Object.entries(e.attrs)) {
          line += `\n  ${k}: ${formatAttrValue(v)}`
        }
      }
      return line
    })
    .join('\n')
  navigator.clipboard.writeText(text)
}

function onClear() {
  logger.clear()
  loadEntries()
}

function loadEntries() {
  entriesRaw.value = [...logger.getBuffer()]
}

function toggleAll() {
  const next = !allSelected.value
  for (const l of allLevels) selectedLevels[l] = next
}

function toggleLevel(level: LogLevel) {
  selectedLevels[level] = !selectedLevels[level]
}
</script>

<template>
  <div v-if="visible" class="dev-console" @click.stop>
    <div class="dev-console-header">
      <div class="d-flex align-items-center gap-2">
        <i class="bi bi-terminal-fill"></i>
        <span class="fw-semibold small">Dev Console</span>
        <span class="badge bg-secondary">{{ entries.length }} entries</span>
        <div class="vr mx-1 opacity-25"></div>
        <label class="d-flex align-items-center gap-1 buffer-label" title="Max buffer entries">
          <i class="bi bi-database"></i>
          Buffer Size:
          <input type="number" v-model.number="maxBuffer" class="buffer-input" min="1" />
        </label>
      </div>
      <div class="d-flex align-items-center gap-1 position-relative">
        <button
          class="btn btn-sm border-0 text-white-50"
          :class="{ 'text-white': !allSelected }"
          @click="filterOpen = !filterOpen"
          title="Filter levels"
        >
          <i class="bi bi-funnel"></i>
          <span class="ms-1 small">{{ filterSummary }}</span>
        </button>
        <div v-if="filterOpen" class="filter-panel" @click.stop>
          <label class="filter-item">
            <input type="checkbox" :checked="allSelected" @change="toggleAll" />
            <span>All</span>
          </label>
          <label v-for="level in allLevels" :key="level" class="filter-item">
            <input type="checkbox" :checked="selectedLevels[level]" @change="toggleLevel(level)" />
            <i :class="levelIcon(level)" class="me-1"></i>
            {{ level }}
          </label>
        </div>
        <button class="btn btn-sm border-0 text-white-50" @click="onCopyAll" title="Copy all">
          <i class="bi bi-clipboard"></i>
        </button>
        <button class="btn btn-sm border-0 text-white-50" @click="onClear" title="Clear">
          <i class="bi bi-trash"></i>
        </button>
        <button class="btn btn-sm border-0 text-white-50" @click="visible = false" title="Hide">
          <i class="bi bi-dash-lg"></i>
        </button>
        <button class="btn btn-sm border-0 text-white-50" @click="emit('close')" title="Close">
          <i class="bi bi-x-lg"></i>
        </button>
      </div>
    </div>
    <div ref="containerRef" class="dev-console-body">
      <div v-for="entry in entries" :key="entry.id" class="dev-console-entry-row" :class="'entry-' + entry.level">
        <div class="dev-console-entry">
          <span class="entry-time">{{ formatTime(entry.time) }}</span>
          <span class="entry-badge" :class="levelBadgeClass(entry.level)">
            <i :class="levelIcon(entry.level)" class="me-1"></i>
            {{ entry.level.toUpperCase() }}
          </span>
          <span v-if="entry.fromBackend" class="entry-source-badge">BACKEND</span>
          <span v-else class="entry-source-badge">FRONTEND</span>
          <span class="entry-message">{{ entry.message }}</span>
          <template v-if="entry.attrs">
            <span v-for="(val, key) in entry.attrs" :key="key" class="entry-inline-attr">
              <span class="entry-inline-key">{{ key }}:</span>
              <span class="entry-inline-val">{{ formatAttrValue(val) }}</span>
            </span>
          </template>
          <span v-if="entry.source" class="entry-source text-white-50">({{ entry.source }})</span>
        </div>
      </div>
    </div>
    <div class="dev-console-footer">
      <label class="small text-white-50 d-flex align-items-center gap-1 cursor-pointer">
        <input type="checkbox" v-model="autoScroll" class="form-check-input m-0" />
        Auto-scroll
      </label>
    </div>
  </div>
</template>

<style scoped>
.dev-console {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: 300px;
  z-index: 99999;
  background: rgba(10, 10, 10, 0.95);
  border-top: 1px solid #333;
  display: flex;
  flex-direction: column;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', monospace;
  font-size: 0.75rem;
  backdrop-filter: blur(8px);
}

.dev-console-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 12px;
  background: #1a1a1a;
  border-bottom: 1px solid #333;
  flex-shrink: 0;
  color: #ccc;
}

.dev-console-body {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.dev-console-entry-row {
  border-bottom: 1px solid rgba(255, 255, 255, 0.03);
}

.dev-console-entry-row:hover {
  background: rgba(255, 255, 255, 0.03);
}

.dev-console-entry {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 2px 12px;
  line-height: 1.5;
}

.entry-time {
  color: #666;
  flex-shrink: 0;
  font-size: 0.7rem;
  width: 100px;
}

.entry-badge {
  font-size: 0.65rem;
  padding: 1px 6px;
  border-radius: 3px;
  flex-shrink: 0;
  min-width: 55px;
  text-align: center;
}

.entry-source-badge {
  font-size: 0.6rem;
  padding: 0 4px;
  border-radius: 2px;
  background: #333;
  color: #888;
  flex-shrink: 0;
}

.entry-message {
  color: #ddd;
  white-space: nowrap;
  flex-shrink: 0;
}

.entry-trace .entry-message { color: #666; }
.entry-debug .entry-message { color: #999; }
.entry-warn .entry-message { color: #ffc107; }
.entry-error .entry-message { color: #ff6b6b; }

.entry-inline-attr {
  display: inline-flex;
  gap: 2px;
  align-items: center;
  background: #2a2a2a;
  border: 1px solid #3a3a3a;
  border-radius: 3px;
  padding: 0 5px;
  font-size: 0.65rem;
  line-height: 1.4;
  flex-shrink: 0;
}

.entry-inline-key {
  color: #6ea8fe;
}

.entry-inline-val {
  color: #aaa;
}

.dev-console-footer {
  padding: 4px 12px;
  background: #1a1a1a;
  border-top: 1px solid #333;
  flex-shrink: 0;
}

.cursor-pointer {
  cursor: pointer;
}

.buffer-label {
  color: #888;
  font-size: 0.7rem;
}

.buffer-input {
  width: 60px;
  height: 20px;
  background: #2a2a2a;
  border: 1px solid #444;
  border-radius: 3px;
  color: #ccc;
  font-size: 0.7rem;
  font-family: inherit;
  padding: 0 4px;
  outline: none;
  text-align: center;
  appearance: textfield;
  -moz-appearance: textfield;
}

.buffer-input:focus {
  border-color: #6ea8fe;
  background: #333;
}

.filter-panel {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: 100;
  background: #1e1e1e;
  border: 1px solid #444;
  border-radius: 4px;
  padding: 4px 0;
  min-width: 120px;
  margin-top: 2px;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  cursor: pointer;
  color: #ccc;
  font-size: 0.7rem;
  white-space: nowrap;
}

.filter-item:hover {
  background: #333;
  color: #fff;
}

.filter-item input[type="checkbox"] {
  width: 11px;
  height: 11px;
  accent-color: #6ea8fe;
  margin: 0;
}

.form-check-input {
  width: 12px;
  height: 12px;
}
</style>
