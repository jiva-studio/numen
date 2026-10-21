import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { getPluginAliases } from '../vite.shared'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: getPluginAliases([
      '@',
      '@plugins',
      '@numen/plugin-sdk/testing',
      '@numen/plugin-sdk',
      '@numen/ui/styles.css',
      '@numen/ui',
    ]),
  },
  test: {
    environment: 'jsdom',
  },
})
