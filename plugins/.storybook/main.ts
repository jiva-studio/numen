import type { StorybookConfig } from '@storybook/vue3-vite'
import vue from '@vitejs/plugin-vue'
import { getPluginAliases } from '../vite.shared.ts'
import { createStoryUtilitiesPlugin } from '../vite.styles.ts'

/**
 * A story drawn for a picture is served by `storybook dev`, which reads no
 * Vite config of the plugins: the Vue compiler and the paths a plugin's imports
 * resolve to are given here, as `vitest.config.ts` gives them to a test run.
 */
const drawing = !process.env['VITEST']

const reaching: NonNullable<StorybookConfig['viteFinal']> = (config) => ({
  ...config,
  plugins: [...(config.plugins ?? []), vue(), createStoryUtilitiesPlugin()],
  resolve: {
    ...config.resolve,
    alias: getPluginAliases([
      '@plugins',
      '@numen/plugin-sdk/testing',
      '@numen/plugin-sdk',
      '@numen/protocol/numen/v1',
      '@numen/protocol',
      '@numen/ui/styles.css',
      '@numen/ui',
      '@numen/wire',
    ]),
  },
})

/** The stories every plugin writes beside its frontend. */
const config: StorybookConfig = {
  stories: ['../*/frontend/**/*.stories.ts'],
  addons: ['@storybook/addon-vitest'],
  framework: { name: '@storybook/vue3-vite', options: {} },
  ...(drawing ? { viteFinal: reaching } : {}),
}

export default config
