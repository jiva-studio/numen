import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  build: {
    lib: {
      entry: resolve(import.meta.dirname, 'frontend/index.ts'),
      formats: ['es'],
      fileName: () => 'index.js',
      cssFileName: 'styles',
    },
    outDir: 'dist',
    emptyOutDir: false,
    rollupOptions: {
      external: [
        'vue',
        '@numen/plugin-sdk',
        '@numen/protocol',
        '@numen/wire',
        '@connectrpc/connect',
        '@connectrpc/connect-web',
      ],
    },
  },
})
