import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { createPluginBuildConfig } from '@numen/plugin-sdk/vite'

export default defineConfig(
  createPluginBuildConfig({
    root: import.meta.dirname,
    plugins: [vue()],
    aliases: ['@plugins', '@numen/plugin-sdk', '@numen/ui'],
  }),
)
