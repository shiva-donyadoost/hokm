mport { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Dev server proxies REST and WebSocket traffic to the Go backend so the
// browser sees a single origin (no CORS in development).
export default defineConfig({
  plugins: [react()],
  build: {
    // Inline gameplay SFX as data: URLs so Howler never fetches /assets/audio/*
    // over the network (IDM and similar download managers intercept those).
    // Largest SFX today ~25KB; 64KB leaves headroom for replacements.
    assetsInlineLimit: 65536,
  },
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
