import { resolve } from 'node:path'
import { defineConfig } from 'astro/config'
import vue from '@vitejs/plugin-vue'

const root = resolve(import.meta.dirname, '../..')

/**
 * Static output: the whole site is files, and the page has nothing to ask a
 * server at the moment it is read.
 *
 * `site` is where those files are published, and canonical URLs are built
 * from it.
 */
export default defineConfig({
  site: 'https://numen.md',
  build: { format: 'file' },
  vite: {
    plugins: [vue()],
    resolve: {
      dedupe: ['vue'],
      alias: {
        vue: resolve(root, 'node_modules/vue/dist/vue.runtime.esm-bundler.js'),
        '@demo-window': resolve(import.meta.dirname, 'src/components/demo/ui/DemoWindow.vue'),
        '@numen/ui/styles.css': resolve(root, 'libs/ui/components/dist/numen-ui.css'),
        '@numen/ui': resolve(root, 'libs/ui/components/dist/index.js'),
        '@numen/workbench/styles.css': resolve(root, 'libs/ui/workbench/dist/numen-shell.css'),
        '@numen/workbench': resolve(root, 'libs/ui/workbench/dist/index.js'),
      },
    },
  },
})
