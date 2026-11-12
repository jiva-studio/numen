import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@numen/core-client': fileURLToPath(
        new URL('../../libs/core/client/src/index.ts', import.meta.url),
      ),
    },
  },
  build: { outDir: 'dist' },
})
