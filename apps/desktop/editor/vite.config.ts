import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import * as hostModules from '@numen/plugin-loader/vite'
import { createWindowConfig } from '@numen/workbench/vite'

// `@/` is the window's own source. A path leaving its own folder is written
// through it, so what layer is reached is read off the import itself.
export default defineConfig(
  createWindowConfig(import.meta.url, {
    plugins: [vue()],
    hostModules,
    outDir: '../../../libs/core/kernel/adapter/window/editor/pages/app',
    own: {
      '@numen/plugin-manager/testing': '../../../libs/plugin/manager/src/testing/index.ts',
      '@numen/plugin-manager': '../../../libs/plugin/manager/src/index.ts',
    },
  }),
)
