import { fileURLToPath, URL } from 'node:url'
import type { StorybookConfig } from '@storybook/vue3-vite'

/** Where a window's own screens are drawn, beside the components they use. */
const APPS = [
  '../../../apps/desktop/ui/src/**/*.stories.ts',
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
    alias: {
      ...(Array.isArray(config.resolve?.alias) ? {} : config.resolve?.alias),
      '@numen/ui': fileURLToPath(new URL('../src/index.ts', import.meta.url)),
    },
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
