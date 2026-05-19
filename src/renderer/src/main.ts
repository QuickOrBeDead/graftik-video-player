import { createApp } from 'vue'

import 'bootstrap/dist/css/bootstrap.css'
import 'bootstrap-icons/font/bootstrap-icons.css'
import 'bootstrap'
import './style.css'
import '@imengyu/vue3-context-menu/lib/vue3-context-menu.css'
import ContextMenu from '@imengyu/vue3-context-menu'

import { createRouter, createWebHashHistory } from 'vue-router'

import App from './App.vue'
import Main from './components/Main.vue'
import Playlists from './components/Playlists.vue'
import NewPlaylist from './components/NewPlaylist.vue'

const routes = [
  { path: '/', component: Main },
  { path: '/playlists', component: Playlists },
  { path: '/add-playlist', component: NewPlaylist }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes: routes
})

createApp(App).use(router).use(ContextMenu).mount('#app')
