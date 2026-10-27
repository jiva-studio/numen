import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@numen/plugin-sdk': fileURLToPath(
        new URL('../../libs/plugin-sdk-ts/src/index.ts', import.meta.url),
      ),
    },
  },
  build: { outDir: 'dist' },
})
