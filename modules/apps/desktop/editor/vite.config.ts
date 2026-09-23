import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  // `@/` is the window's own source. A path leaving its own folder is written
  // through it, so what layer is reached is read off the import itself.
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  // Built into the Go package that serves it: what is embedded has to sit
  // beside the code that embeds it.
  // `assets` is where the vault's own files are asked for, so the window's
  // built pieces are filed apart from them.
  build: {
    outDir: '../../../libs/core/adapter/window/editor/pages/app',
    emptyOutDir: true,
    assetsDir: 'built',
  },
})
