import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { createPluginBuildConfig } from '../vite.shared'

export default defineConfig(
  createPluginBuildConfig({
    root: import.meta.dirname,
    plugins: [vue()],
    aliases: [
      '@numen/plugin-sdk',
      '@numen/ui',
      '@numen/protocol/numen/v1',
      '@numen/protocol',
    ],
  }),
)
