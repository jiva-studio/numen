import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import * as hostModules from '@numen/plugin-host/vite'
import { createWindowConfig } from '@numen/shell/vite'

// `@/` is the window's own source. A path leaving its own folder is written
// through it, so what layer is reached is read off the import itself.
export default defineConfig(
  createWindowConfig(import.meta.url, {
    plugins: [vue()],
    hostModules,
    outDir: '../../../libs/core/adapter/window/editor/pages/app',
    own: { '@numen/plugin-manager': '../../../libs/plugin-manager/src/index.ts' },
  }),
)
