import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const API_TARGET = process.env.VITE_IM_API_PROXY_TARGET ?? 'http://localhost:8901'
const WS_TARGET = process.env.VITE_IM_WS_PROXY_TARGET ?? 'ws://localhost:8902'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/im/api': {
        target: API_TARGET,
        changeOrigin: true,
      },
      '/im/ws': {
        target: WS_TARGET,
        ws: true,
        changeOrigin: true,
      },
    },
  },
})
