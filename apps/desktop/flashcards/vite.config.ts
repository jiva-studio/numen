import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import * as hostModules from '@numen/plugin-loader/vite'
import { createWindowConfig } from '@numen/workbench/vite'

// Built into the Go package that serves it: what is embedded has to sit
// beside the code that embeds it.
export default defineConfig(
  createWindowConfig(import.meta.url, {
    plugins: [vue()],
    hostModules,
    outDir: '../../../libs/core/kernel/adapter/window/shell/pages/app',
  }),
)
