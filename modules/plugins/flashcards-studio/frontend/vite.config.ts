import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    lib: {
      entry: './index.ts',
      name: 'FlashcardsStudio',
      fileName: 'index',
      formats: ['es'],
    },
    rollupOptions: {
      external: ['vue', '@numen/plugin-sdk', '@numen/ui', '@numen/wire'],
    },
  },
})
