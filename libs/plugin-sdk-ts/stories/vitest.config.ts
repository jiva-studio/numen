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
import { getStoryBrowser } from '../../ui/core/.storybook/browser'
import { createStoryUtilitiesPlugin, getPluginAliases } from '../src/vite'

const alias = getPluginAliases([
  '@plugins',
  '@numen/plugin-sdk/testing',
  '@numen/plugin-sdk',
  '@numen/protocol/numen/v1',
  '@numen/protocol',
  '@numen/ui/styles.css',
  '@numen/ui',
])

export default defineConfig({
  plugins: [
    vue(),
    createStoryUtilitiesPlugin(),
    storybookTest({ configDir: fileURLToPath(new URL('./.storybook', import.meta.url)) }),
  ],
  resolve: { alias },
  test: {
    name: 'stories',
    // Every story is drawn twice, and the frames of all of them come off one
    // machine.
    testTimeout: 30_000,
    browser: getStoryBrowser(),
  },
})
