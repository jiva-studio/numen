import { defineConfig } from 'astro/config'
import starlight from '@astrojs/starlight'

/**
 * The manual. Static output, published on its own host, and built from the
 * same faces and colours as the page the product is read about on.
 */
export default defineConfig({
  site: 'https://docs.numen.md',
  integrations: [
    starlight({
      title: 'numen',
      description: 'How to keep notes in numen: writing, linking, finding, and the agent.',
      logo: { src: './src/assets/mark.svg', alt: '' },
      favicon: '/favicon.svg',
      head: [
        { tag: 'link', attrs: { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' } },
      ],
      customCss: [
        // Both axes: a heading set large is drawn with the letterforms cut for
        // that size. The face's default is the cut for text, and a text cut
        // enlarged is what reads as uneven.
        '@fontsource-variable/newsreader/standard.css',
        '@fontsource-variable/newsreader/standard-italic.css',
        '@fontsource/ibm-plex-sans/400.css',
        '@fontsource/ibm-plex-sans/500.css',
        '@fontsource/ibm-plex-sans/600.css',
        '@fontsource/ibm-plex-mono/400.css',
        '@fontsource/ibm-plex-mono/500.css',
        './src/styles/manual.css',
      ],
      // The manual is read, not contributed to: the repository is private, and
      // a link into it would lead nowhere.
      editLink: undefined,
      lastUpdated: false,
      sidebar: [
        {
          label: 'Start',
          items: [
            { slug: 'install' },
            { slug: 'vault' },
            { slug: 'window' },
          ],
        },
        {
          label: 'Notes',
          items: [
            { slug: 'writing' },
            { slug: 'links' },
            { slug: 'plex' },
            { slug: 'finding' },
            { slug: 'documents' },
          ],
        },
        {
          label: 'The agent',
          items: [{ slug: 'agent' }, { slug: 'connect' }],
        },
        {
          label: 'Settings',
          items: [
            { slug: 'settings' },
            { slug: 'themes' },
            { slug: 'meaning' },
            { slug: 'reading' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { slug: 'commands' },
            { slug: 'keyboard' },
            { slug: 'reference' },
            { slug: 'starting' },
            { slug: 'cli' },
          ],
        },
        { slug: 'trouble' },
        { label: 'Forum', link: 'https://forum.numen.md' },
        { label: 'Download numen', link: 'https://numen.md/#get' },
      ],
    }),
  ],
})
