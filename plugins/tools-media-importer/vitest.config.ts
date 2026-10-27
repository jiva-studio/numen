import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { getPluginAliases } from '@numen/plugin-sdk/vite'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: getPluginAliases(['@numen/plugin-sdk/testing', '@numen/plugin-sdk']),
  },
  test: {
    environment: 'jsdom',
  },
})
