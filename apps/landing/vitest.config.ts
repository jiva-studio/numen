import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { getStoryBrowser } from '../../libs/ui/core/.storybook/browser'

export default defineConfig({
  plugins: [vue()],
  test: {
    name: 'browser',
    include: ['src/**/*.test.ts'],
    browser: getStoryBrowser(),
  },
})
