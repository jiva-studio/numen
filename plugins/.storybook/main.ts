import type { StorybookConfig } from '@storybook/vue3-vite'

/** The stories every plugin writes beside its frontend. */
const config: StorybookConfig = {
  stories: ['../*/frontend/**/*.stories.ts'],
  addons: ['@storybook/addon-vitest'],
  framework: { name: '@storybook/vue3-vite', options: {} },
}

export default config
