import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import dts from 'vite-plugin-dts'

export default defineConfig({
  plugins: [vue(), dts({ include: ['src'], rollupTypes: true })],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  build: {
    lib: {
      entry: fileURLToPath(new URL('./src/index.ts', import.meta.url)),
      name: 'NumenUI',
      fileName: 'numen-ui',
    },
    // Vue is the host application's, not ours; two copies of it in one page
    // is a broken application, not a large bundle.
    rollupOptions: { external: ['vue'], output: { globals: { vue: 'Vue' } } },
  },
})
