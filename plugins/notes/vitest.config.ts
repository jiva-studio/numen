import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { getPluginAliases } from '@numen/plugin-sdk/vite'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: getPluginAliases([
      '@plugins',
      '@numen/plugin-sdk/testing',
      '@numen/plugin-sdk',
      '@numen/ui/styles.css',
      '@numen/ui',
    ]),
  },
  test: {
    environment: 'jsdom',
    testTimeout: 30_000,
  },
})
