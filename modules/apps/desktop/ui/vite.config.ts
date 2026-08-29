import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  // Built into the Go package that serves it: what is embedded has to sit
  // beside the code that embeds it.
  // `assets` is where the vault's own files are asked for, so the window's
  // built pieces are filed apart from them.
  build: {
    outDir: '../../../libs/core/adapter/webui/pages/app',
    emptyOutDir: true,
    assetsDir: 'built',
  },
})
