import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Dev server proxies REST and WebSocket traffic to the Go backend so the
// browser sees a single origin (no CORS in development).
export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    port: 5173,
    proxy: {
      '/api/ws': {
        target: 'ws://backend:8080',
        ws: true,
      },
      '/api': {
        target: 'http://backend:8080',
        changeOrigin: true,
      },
    },
  },
})
