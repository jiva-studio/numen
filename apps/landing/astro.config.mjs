import { resolve } from 'node:path'
import { defineConfig } from 'astro/config'
import vue from '@vitejs/plugin-vue'
import { createStoryUtilitiesPlugin } from '../../libs/plugin-sdk-ts/src/vite.ts'

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
    plugins: [vue(), createStoryUtilitiesPlugin(resolve(root, 'plugins'))],
    resolve: {
      dedupe: ['vue'],
      alias: {
        vue: resolve(root, 'node_modules/vue/dist/vue.runtime.esm-bundler.js'),
        '@plugins': resolve(root, 'plugins'),
        '@numen/core-client': resolve(root, 'libs/core-client/src/index.ts'),
        '@numen/plugin-host': resolve(root, 'libs/plugin-host-ts/src/index.ts'),
        '@numen/plugin-sdk/testing': resolve(root, 'libs/plugin-sdk-ts/src/testing/index.ts'),
        '@numen/plugin-sdk': resolve(root, 'libs/plugin-sdk-ts/src/index.ts'),
        '@numen/protocol/numen/v1': resolve(root, 'libs/protocol/src/numen/v1'),
        '@numen/protocol': resolve(root, 'libs/protocol/index.ts'),
        '@numen/ui/styles.css': resolve(root, 'libs/ui/core/dist/numen-ui.css'),
        '@numen/ui': resolve(root, 'libs/ui/core/dist/index.js'),
        '@demo-window': resolve(import.meta.dirname, 'src/components/DemoWindow.vue'),
        '@editor-features': resolve(root, 'apps/desktop/editor/src/features'),
        '@numen/ui-shell/styles.css': resolve(root, 'libs/ui/shell/dist/numen-shell.css'),
        '@numen/ui-shell': resolve(root, 'libs/ui/shell/dist/index.js'),
      },
    },
  },
})
