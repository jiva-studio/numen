import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  // Built into the Go package that serves it: what is embedded has to sit
  // beside the code that embeds it.
  build: { outDir: '../internal/adapter/webui/pages', emptyOutDir: true },
  // In a browser the questions go to a Go process on this port; in the window
  // they are answered by the same handler, in the same binary.
  server: { proxy: { '/api': 'http://127.0.0.1:34115' } },
})
