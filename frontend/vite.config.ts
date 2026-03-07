import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../backend/internal/adapter/dashboard/dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/-/api': 'http://localhost:3001',
      '/-/events': 'http://localhost:3001',
      '/webhook': 'http://localhost:3001',
      '/health': 'http://localhost:3001',
      '/gitea': 'http://localhost:3001',
      '/-/locks': 'http://localhost:3001',
    },
  },
})
