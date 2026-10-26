import { join } from 'node:path'
import { fileURLToPath, URL } from 'node:url'
import type { StorybookConfig } from '@storybook/vue3-vite'

const EDITOR_SRC = fileURLToPath(new URL('../../../../apps/desktop/editor/src', import.meta.url))
const FLASHCARDS_SRC = fileURLToPath(
  new URL('../../../../apps/desktop/flashcards/src', import.meta.url),
)
const SHELL_SRC = fileURLToPath(new URL('../src', import.meta.url))

/** Where a window's own screens are drawn, beside the chrome they stand in. */
const APPS = [
  '../../../../apps/desktop/editor/src/**/*.stories.ts',
  '../../../../apps/desktop/flashcards/src/**/*.stories.ts',
]

/**
 * A screen of an application is staged for a picture to be taken of it, and is
 * drawn here and run nowhere: the corpus a test run draws is this module's own
 * stories, which are served out of this module.
 */
const drawing = !process.env['VITEST']

/**
 * How a story reaches its imports: `@/` from the source of the module it is
 * written in, `@numen/ui` through its build, and `@numen/shell` from this source.
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
          this: {
            resolve: (
              id: string,
              importer?: string,
              options?: { skipSelf?: boolean },
            ) => Promise<unknown>
          },
          source: string,
          importer: string | undefined,
        ) {
          if (!importer) return null
          const target = importer.includes('/apps/desktop/editor/')
            ? join(EDITOR_SRC, source)
            : importer.includes('/apps/desktop/flashcards/')
              ? join(FLASHCARDS_SRC, source)
              : join(SHELL_SRC, source)
          return this.resolve(target, importer, { skipSelf: true })
        },
      },
      {
        find: /^@numen\/ui\/styles\.css$/,
        replacement: fileURLToPath(new URL('../../core/dist/numen-ui.css', import.meta.url)),
      },
      {
        find: /^@numen\/ui$/,
        replacement: fileURLToPath(new URL('../../core/dist/index.js', import.meta.url)),
      },
      {
        find: /^@numen\/shell\/styles\.css$/,
        replacement: fileURLToPath(new URL('../src/styles.css', import.meta.url)),
      },
      {
        find: /^@numen\/shell$/,
        replacement: fileURLToPath(new URL('../src/index.ts', import.meta.url)),
      },
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
