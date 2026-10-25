import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  // `@/` is the window's own source. A path leaving its own folder is written
  // through it, so what layer is reached is read off the import itself.
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@plugins': fileURLToPath(new URL('../../../plugins', import.meta.url)),
      '@numen/plugin-sdk': fileURLToPath(
        new URL('../../../libs/plugin-sdk-ts/src/index.ts', import.meta.url),
      ),
      '@numen/protocol': fileURLToPath(new URL('../../../libs/protocol/index.ts', import.meta.url)),
      '@numen/wire': fileURLToPath(new URL('../../../libs/wire/index.ts', import.meta.url)),
      '@numen/ui/styles.css': fileURLToPath(
        new URL('../../../libs/ui/dist/numen-ui.css', import.meta.url),
      ),
      '@numen/ui': fileURLToPath(new URL('../../../libs/ui/dist/index.js', import.meta.url)),
    },
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
