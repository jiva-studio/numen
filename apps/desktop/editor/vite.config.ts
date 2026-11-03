import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import {
  createHostModuleEntries,
  getEntryFileName,
  hostModuleInputs,
} from '@numen/plugin-host/vite'
import { LIGHTNING_CSS, createWindowAliases, createWindowBuild } from '@numen/shell/vite'

export default defineConfig({
  plugins: [vue(), createHostModuleEntries()],
  css: { lightningcss: LIGHTNING_CSS },
  // `@/` is the window's own source. A path leaving its own folder is written
  // through it, so what layer is reached is read off the import itself.
  resolve: {
    alias: createWindowAliases(import.meta.url, {
      '@numen/plugin-manager': '../../../libs/plugin-manager/src/index.ts',
    }),
  },
  // Output goes into the Go package that serves it: embedded files must be
  // located next to the embedding code.
  build: createWindowBuild(import.meta.url, '../../../libs/core/adapter/window/editor/pages/app', {
    inputs: hostModuleInputs,
    getEntryFileName,
  }),
})
