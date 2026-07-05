<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { logger, type LogLevel, type LogEntry } from '@renderer/utils/logger'

const emit = defineEmits<{ close: [] }>()

const filterLevel = ref<LogLevel | 'all'>('all')
const visible = ref(true)
const autoScroll = ref(true)
const containerRef = ref<HTMLDivElement>()
const expandedAttrs = ref(new Set<number>())

function toggleAttrs(id: number) {
  const s = expandedAttrs.value
  if (s.has(id)) s.delete(id)
  else s.add(id)
  expandedAttrs.value = new Set(s)
}

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
  const all = logger.getBuffer()
  if (filterLevel.value === 'all') return all
  return all.filter(e => e.level === filterLevel.value || shouldInclude(e.level))
})

function shouldInclude(level: LogLevel): boolean {
  if (filterLevel.value === 'all') return true
  const priorities: Record<string, number> = { trace: 0, debug: 1, info: 2, warn: 3, error: 4 }
  return priorities[level] >= priorities[filterLevel.value]
}

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

let unsub: () => void

onMounted(() => {
  unsub = logger.subscribe(() => {
    if (autoScroll.value && containerRef.value) {
      nextTick(() => {
        containerRef.value!.scrollTop = containerRef.value!.scrollHeight
      })
    }
  })
})

onUnmounted(() => {
  if (unsub) unsub()
})

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
}

function onFilterChange() {
  const next: Record<string, LogLevel | 'all'> = {
    all: 'error',
    error: 'warn',
    warn: 'info',
    info: 'debug',
    debug: 'trace',
    trace: 'all',
  }
  filterLevel.value = next[filterLevel.value] || 'all'
}
</script>

<template>
  <div v-if="visible" class="dev-console" @click.stop>
    <div class="dev-console-header">
      <div class="d-flex align-items-center gap-2">
        <i class="bi bi-terminal-fill"></i>
        <span class="fw-semibold small">Dev Console</span>
        <span class="badge bg-secondary">{{ entries.length }} entries</span>
      </div>
      <div class="d-flex align-items-center gap-1">
        <button
          class="btn btn-sm border-0 text-white-50"
          :class="{ 'text-white': filterLevel !== 'all' }"
          @click="onFilterChange"
          :title="'Filter: ' + filterLevel"
        >
          <i class="bi bi-funnel"></i>
          <span class="ms-1 small">{{ filterLevel }}</span>
        </button>
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
    <div ref="containerRef" class="dev-console-body" @scroll="autoScroll = false">
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
          <span v-if="entry.source" class="entry-source text-white-50">({{ entry.source }})</span>
          <button
            v-if="entry.attrs && Object.keys(entry.attrs).length > 0"
            class="entry-attrs-toggle"
            @click="toggleAttrs(entry.id)"
            :title="expandedAttrs.has(entry.id) ? 'Hide attrs' : 'Show attrs'"
          >
            <i :class="expandedAttrs.has(entry.id) ? 'bi bi-chevron-down' : 'bi bi-chevron-right'"></i>
            <span class="attrs-count">{{ Object.keys(entry.attrs).length }}</span>
          </button>
        </div>
        <div v-if="entry.attrs && expandedAttrs.has(entry.id)" class="entry-attrs">
          <div v-for="(val, key) in entry.attrs" :key="key" class="entry-attr">
            <span class="attr-key">{{ key }}:</span>
            <span class="attr-val">{{ formatAttrValue(val) }}</span>
          </div>
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

.entry-attrs-toggle {
  background: none;
  border: none;
  color: #888;
  cursor: pointer;
  padding: 0 4px;
  display: flex;
  align-items: center;
  gap: 3px;
  font-size: 0.7rem;
  flex-shrink: 0;
}

.entry-attrs-toggle:hover {
  color: #fff;
}

.attrs-count {
  font-size: 0.6rem;
  background: #444;
  color: #bbb;
  border-radius: 8px;
  padding: 0 5px;
  line-height: 1.4;
}

.entry-attrs {
  padding: 2px 12px 4px 112px;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.entry-attr {
  display: flex;
  gap: 6px;
  font-size: 0.7rem;
  line-height: 1.5;
}

.attr-key {
  color: #6ea8fe;
  flex-shrink: 0;
  min-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.attr-val {
  color: #aaa;
  overflow: hidden;
  text-overflow: ellipsis;
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
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entry-trace .entry-message { color: #666; }
.entry-debug .entry-message { color: #999; }
.entry-warn .entry-message { color: #ffc107; }
.entry-error .entry-message { color: #ff6b6b; }

.dev-console-footer {
  padding: 4px 12px;
  background: #1a1a1a;
  border-top: 1px solid #333;
  flex-shrink: 0;
}

.cursor-pointer {
  cursor: pointer;
}

.form-check-input {
  width: 12px;
  height: 12px;
}
</style>
