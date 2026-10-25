import type { StorybookConfig } from '@storybook/vue3-vite'

/**
 * The screens this window draws, staged where they are written. They reach the
 * library the way the window does, through its build.
 */
const config: StorybookConfig = {
  stories: ['../src/**/*.stories.ts'],
  addons: ['@storybook/addon-vitest'],
  framework: { name: '@storybook/vue3-vite', options: {} },
}

export default config
