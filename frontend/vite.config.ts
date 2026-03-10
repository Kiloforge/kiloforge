/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/',
  build: {
    outDir: '../backend/internal/adapter/dashboard/dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:4001',
      '/events': 'http://localhost:4001',
      '/ws': 'http://localhost:4001',
      '/webhook': 'http://localhost:4001',
      '/health': 'http://localhost:4001',
      '/gitea': 'http://localhost:4001',
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.ts',
    css: { modules: { classNameStrategy: 'non-scoped' } },
    exclude: ['e2e/**', 'node_modules/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'text-summary'],
      exclude: ['e2e/**', 'src/test/**', '**/*.d.ts'],
    },
  },
})
