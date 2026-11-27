import { resolve } from 'node:path'
import { defineConfig } from 'astro/config'
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
 */
export default defineConfig({
  site: 'https://numen.md',
  build: { format: 'file' },
  markdown: { rehypePlugins: [rehypeSanitize] },
  vite: {
    resolve: {
      alias: {
        '@numen/ui/styles.css': resolve(root, 'libs/ui/components/dist/numen-ui.css'),
      },
    },
  },
})
