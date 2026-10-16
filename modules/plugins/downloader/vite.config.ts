import { existsSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'

function ensureStylesPlugin(): Plugin {
  return {
    name: 'ensure-styles',
    apply: 'build',
    closeBundle() {
      const distDir = resolve(import.meta.dirname, 'dist')
      const cssPath = resolve(distDir, 'styles.css')
      if (!existsSync(cssPath)) {
        writeFileSync(cssPath, '')
      }
    },
  }
}

export default defineConfig({
  plugins: [vue(), ensureStylesPlugin()],
  build: {
    lib: {
      entry: resolve(import.meta.dirname, 'frontend/index.ts'),
      formats: ['es'],
      fileName: () => 'index.js',
      cssFileName: 'styles',
    },
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      external: [
        'vue',
        '@numen/plugin-sdk',
        '@numen/protocol',
        '@numen/wire',
        '@connectrpc/connect',
        '@connectrpc/connect-web',
      ],
      output: {
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) return 'styles.css'
          return assetInfo.name || 'asset'
        },
      },
    },
  },
})
