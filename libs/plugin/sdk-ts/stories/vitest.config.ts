/**
 * The stories of every plugin, rendered in a real browser as one corpus.
 *
 * A plugin's unit tests run under its own `vitest.config.ts`. The stories of
 * all plugins are collected here, against the components' build, as a plugin
 * consumes them in the window.
 */
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import { getStoryBrowser } from '../../../ui/components/.storybook/browser'
import { createStoryUtilitiesPlugin } from './.storybook/utilities'

export default defineConfig({
  plugins: [
    vue(),
    createStoryUtilitiesPlugin(fileURLToPath(new URL('../../../../plugins', import.meta.url))),
    storybookTest({ configDir: fileURLToPath(new URL('./.storybook', import.meta.url)) }),
  ],
  test: {
    name: 'stories',
    // Every story is drawn twice, and the frames of all of them come off one
    // machine.
    testTimeout: 30_000,
    browser: getStoryBrowser(),
  },
})
