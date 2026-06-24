import { resolve } from 'path'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  resolve: {
    alias: {
      '@renderer': resolve('src'),
      '@wailsjs': resolve('wailsjs')
    }
  },
  plugins: [vue()]
})
