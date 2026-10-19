import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // Anfragen an /api gehen an das Backend (make dev)
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
