import { createApp } from 'vue'

import 'bootstrap/dist/css/bootstrap.css'
import 'bootstrap-icons/font/bootstrap-icons.css'
import 'bootstrap'
import './style.css'
import '@imengyu/vue3-context-menu/lib/vue3-context-menu.css'
import ContextMenu from '@imengyu/vue3-context-menu'

import App from './App.vue'
import { logger } from './utils/logger'
import { GetLogLevel } from '../wailsjs/go/main/App'

const level = await GetLogLevel()
if (level) {
  logger.setLevel(level as 'trace' | 'debug' | 'info' | 'warn' | 'error')
}

createApp(App).use(ContextMenu).mount('#app')
