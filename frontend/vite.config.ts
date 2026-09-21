import { fileURLToPath, URL } from 'node:url'

import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

// The Compose stack publishes the API on BWIM_API_PORT (8081 by default), so
// the dev proxy has to follow the same variable or every request from
// `npm run dev` lands on a closed port.
const apiTarget = `http://localhost:${process.env.BWIM_API_PORT ?? '8081'}`

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: '0.0.0.0',
    port: Number(process.env.BWIM_DEV_PORT ?? 6001),
    proxy: {
      '/api': apiTarget,
      '/health': apiTarget,
      '/ready': apiTarget,
    },
  },
  test: {
    environment: 'jsdom',
  },
})
