import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  // Built into the Go package that serves it: what is embedded has to sit
  // beside the code that embeds it.
  build: {
    outDir: '../../../libs/core/adapter/reviewui/pages/app',
    emptyOutDir: true,
    assetsDir: 'built',
  },
  // A test runs in a document, because the client this page is built on reads
  // the address the window was served from.
  test: {
    environment: 'jsdom',
  },
})
