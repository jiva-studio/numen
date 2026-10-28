import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import {
  createHostModuleEntries,
  getEntryFileName,
  hostModuleInputs,
} from '@numen/plugin-host/vite'
import { LIGHTNING_CSS } from '@numen/shell/vite'

export default defineConfig({
  plugins: [vue(), createHostModuleEntries()],
  css: { lightningcss: LIGHTNING_CSS },
  // `@/` is the window's own source. A path leaving its own folder is written
  // through it, so what layer is reached is read off the import itself.
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@numen/plugin-host': fileURLToPath(
        new URL('../../../libs/plugin-host-ts/src/index.ts', import.meta.url),
      ),
      '@numen/plugin-sdk': fileURLToPath(
        new URL('../../../libs/plugin-sdk-ts/src/index.ts', import.meta.url),
      ),
      '@numen/protocol': fileURLToPath(new URL('../../../libs/protocol/index.ts', import.meta.url)),
      '@numen/ui/styles.css': fileURLToPath(
        new URL('../../../libs/ui/core/dist/numen-ui.css', import.meta.url),
      ),
      '@numen/ui': fileURLToPath(new URL('../../../libs/ui/core/dist/index.js', import.meta.url)),
      '@numen/shell/styles.css': fileURLToPath(
        new URL('../../../libs/ui/shell/dist/numen-shell.css', import.meta.url),
      ),
      '@numen/shell': fileURLToPath(
        new URL('../../../libs/ui/shell/dist/index.js', import.meta.url),
      ),
    },
  },
  // Built into the Go package that serves it: what is embedded has to sit
  // beside the code that embeds it.
  build: {
    outDir: '../../../libs/core/adapter/window/shell/pages/app',
    emptyOutDir: true,
    assetsDir: 'built',
    rollupOptions: {
      input: {
        index: fileURLToPath(new URL('./index.html', import.meta.url)),
        ...hostModuleInputs,
      },
      // A host module keeps every export, and the page and the plugins share its one instance.
      preserveEntrySignatures: 'strict',
      output: { entryFileNames: getEntryFileName },
    },
  },
})
