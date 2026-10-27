import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { getPluginAliases } from '../vite.shared'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: getPluginAliases([
      '@numen/plugin-sdk/testing',
      '@numen/plugin-sdk',
      '@numen/protocol/numen/v1',
      '@numen/protocol',
      '@numen/ui/styles.css',
      '@numen/ui',
    ]),
  },
  test: {
    environment: 'jsdom',
  },
})
