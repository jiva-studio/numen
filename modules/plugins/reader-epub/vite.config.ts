import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { createPluginBuildConfig } from '../vite.shared'

export default defineConfig(
  createPluginBuildConfig({
    root: import.meta.dirname,
    plugins: [vue()],
    aliases: ['@', '@plugins', '@numen/plugin-sdk', '@numen/protocol', '@numen/ui', '@numen/wire'],
    external: ['vue', '@numen/plugin-sdk', '@numen/ui'],
  }),
)
