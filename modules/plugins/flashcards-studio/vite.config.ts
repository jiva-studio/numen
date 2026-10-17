import { fileURLToPath, URL } from 'node:url'
import { resolve } from 'node:path'
import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'

function ensureStylesPlugin(): Plugin {
  return {
    name: 'ensure-styles',
    apply: 'build',
    generateBundle(_options, bundle) {
      if (!bundle['styles.css']) {
        this.emitFile({
          type: 'asset',
          fileName: 'styles.css',
          source: '',
        })
      }
    },
  }
}

export default defineConfig({
  plugins: [vue(), ensureStylesPlugin()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('../../apps/desktop/editor/src', import.meta.url)),
      '@plugins': fileURLToPath(new URL('..', import.meta.url)),
      '@numen/plugin-sdk': fileURLToPath(new URL('../../libs/plugin-sdk/src/index.ts', import.meta.url)),
      '@numen/ui': fileURLToPath(new URL('../../libs/ui/dist/index.js', import.meta.url)),
      '@numen/wire': fileURLToPath(new URL('../../libs/wire/index.ts', import.meta.url)),
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    lib: {
      entry: resolve(import.meta.dirname, 'frontend/index.ts'),
      formats: ['es'],
      fileName: () => 'index.js',
      cssFileName: 'styles',
    },
    rollupOptions: {
      external: ['vue', '@numen/plugin-sdk', '@numen/ui', '@numen/wire'],
      output: {
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) return 'styles.css'
          return assetInfo.name || 'asset'
        },
      },
    },
  },
})
