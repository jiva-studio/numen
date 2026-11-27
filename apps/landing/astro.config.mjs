import { resolve } from 'node:path'
import { defineConfig } from 'astro/config'

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
    resolve: {
      alias: {
        '@numen/ui/styles.css': resolve(root, 'libs/ui/components/dist/numen-ui.css'),
      },
    },
  },
})
