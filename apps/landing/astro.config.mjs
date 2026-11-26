import { resolve } from 'node:path'
import { defineConfig } from 'astro/config'
import vue from '@astrojs/vue'
import rehypeSanitize from 'rehype-sanitize'

const root = resolve(import.meta.dirname, '../..')

/**
 * Static output: the whole site is files, and the page has nothing to ask a
 * server at the moment it is read.
 *
 * `site` is where those files are published, and canonical URLs are built
 * from it.
 *
 * Markdown is sanitized because the plugin catalog renders the READMEs and
 * release notes of plugins published outside this repository.
 *
 * Vue draws the filter bar of the plugin catalog from `@numen/ui`, which is reached through its
 * build; the main page has no framework on it.
 */
export default defineConfig({
  site: 'https://numen.md',
  build: { format: 'file' },
  markdown: { rehypePlugins: [rehypeSanitize] },
  integrations: [vue()],
  vite: {
    resolve: {
      dedupe: ['vue'],
      alias: {
        '@numen/ui/styles.css': resolve(root, 'libs/ui/components/dist/numen-ui.css'),
        '@numen/ui': resolve(root, 'libs/ui/components/dist/index.js'),
      },
    },
  },
})
