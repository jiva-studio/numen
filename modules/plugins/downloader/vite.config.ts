import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { createPluginBuildConfig } from '../vite.shared'

export default defineConfig(
  createPluginBuildConfig({
    root: import.meta.dirname,
    plugins: [vue()],
    aliases: [],
    external: [
      'vue',
      '@numen/plugin-sdk',
      '@numen/protocol',
      '@numen/wire',
      '@connectrpc/connect',
      '@connectrpc/connect-web',
    ],
  }),
)
