import { fileURLToPath, URL } from 'node:url'
import type { StorybookConfig } from '@storybook/vue3-vite'
import vue from '@vitejs/plugin-vue'
import { createStoryUtilitiesPlugin } from '../../src/viteStyles.ts'

/**
 * A story drawn for a picture is served by `storybook dev`, which reads no
 * Vite config of the plugins: the Vue compiler and the utilities of the plugins
 * are given here, as `vitest.config.ts` gives them to a test run.
 */
const drawing = !process.env['VITEST']

const reaching: NonNullable<StorybookConfig['viteFinal']> = (config) => ({
  ...config,
  plugins: [
    ...(config.plugins ?? []),
    vue(),
    createStoryUtilitiesPlugin(fileURLToPath(new URL('../../../../../plugins', import.meta.url))),
  ],
})

/** The stories every plugin writes beside its frontend. */
const config: StorybookConfig = {
  stories: ['../../../../../plugins/*/frontend/**/*.stories.ts'],
  addons: ['@storybook/addon-vitest'],
  framework: { name: '@storybook/vue3-vite', options: {} },
  ...(drawing ? { viteFinal: reaching } : {}),
}

export default config
