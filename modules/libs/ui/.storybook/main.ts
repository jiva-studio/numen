import { join } from 'node:path'
import { fileURLToPath, URL } from 'node:url'
import type { StorybookConfig } from '@storybook/vue3-vite'

const EDITOR_SRC = fileURLToPath(new URL('../../../apps/desktop/editor/src', import.meta.url))
const FLASHCARDS_SRC = fileURLToPath(new URL('../../../apps/desktop/flashcards/src', import.meta.url))
const UI_SRC = fileURLToPath(new URL('../src', import.meta.url))

/** Where a window's own screens are drawn, beside the components they use. */
const APPS = [
  '../../../apps/desktop/editor/src/**/*.stories.ts',
  '../../../apps/desktop/flashcards/src/**/*.stories.ts',
]

/**
 * A screen of an application is staged for a picture to be taken of it, and is
 * drawn here and run nowhere: the corpus a test run draws is this module's own
 * stories, which are served out of this module.
 */
const drawing = !process.env['VITEST']

/**
 * How a screen of an application reaches this module: from this source, and
 * against the one copy of Vue the page holds. It asks for the built module,
 * and two copies of Vue in one page is a broken page.
 */
const reaching: NonNullable<StorybookConfig['viteFinal']> = (config) => ({
  ...config,
  resolve: {
    ...config.resolve,
    alias: [
      {
        find: /^@\/(.*)/,
        replacement: '$1',
        async customResolver(
          this: { resolve: (id: string, importer?: string, options?: { skipSelf?: boolean }) => Promise<unknown> },
          source: string,
          importer: string | undefined,
        ) {
          if (!importer) return null
          const target = importer.includes('/apps/desktop/editor/')
            ? join(EDITOR_SRC, source)
            : importer.includes('/apps/desktop/flashcards/')
              ? join(FLASHCARDS_SRC, source)
              : join(UI_SRC, source)
          return this.resolve(target, importer, { skipSelf: true })
        },
      },
      { find: '@numen/ui', replacement: fileURLToPath(new URL('../src/index.ts', import.meta.url)) },
    ] as never,
    dedupe: [...(config.resolve?.dedupe ?? []), 'vue', '@lucide/vue'],
  },
})

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.ts', ...(drawing ? APPS : [])],
  addons: ['@storybook/addon-docs', '@storybook/addon-vitest'],
  framework: { name: '@storybook/vue3-vite', options: {} },
  ...(drawing ? { viteFinal: reaching } : {}),
}

export default config
