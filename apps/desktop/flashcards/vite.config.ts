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
  resolve: { alias: createWindowAliases(import.meta.url) },
  // Built into the Go package that serves it: what is embedded has to sit
  // beside the code that embeds it.
  build: createWindowBuild(import.meta.url, '../../../libs/core/adapter/window/shell/pages/app', {
    inputs: hostModuleInputs,
    getEntryFileName,
  }),
})
