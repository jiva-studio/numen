import { defineConfig } from 'astro/config'

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
})
