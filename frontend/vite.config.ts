/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/',
  build: {
    outDir: '../backend/internal/adapter/dashboard/dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/react-dom/') || id.includes('node_modules/react/') || id.includes('node_modules/scheduler/')) {
            return 'vendor-react';
          }
          if (id.includes('node_modules/react-router')) {
            return 'vendor-router';
          }
          if (id.includes('node_modules/@tanstack/')) {
            return 'vendor-query';
          }
          if (id.includes('node_modules/react-markdown') || id.includes('node_modules/remark-') || id.includes('node_modules/rehype-') || id.includes('node_modules/unified') || id.includes('node_modules/mdast-') || id.includes('node_modules/hast-') || id.includes('node_modules/micromark') || id.includes('node_modules/unist-')) {
            return 'vendor-markdown';
          }
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:39517',
      '/events': 'http://localhost:39517',
      '/ws': 'http://localhost:39517',
      '/webhook': 'http://localhost:39517',
      '/health': 'http://localhost:39517',
      '/gitea': 'http://localhost:39517',
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
