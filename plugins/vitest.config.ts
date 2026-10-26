/**
 * The stories of every plugin, drawn in a real browser as one corpus.
 *
 * A plugin's unit tests run under its own `vitest.config.ts`. The stories of
 * all of them are collected here, against the components' build, the way a
 * plugin reaches them in the window.
 */
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import { getStoryBrowser } from '../libs/ui/core/.storybook/browser'
import { getPluginAliases } from './vite.shared'
import { createStoryUtilitiesPlugin } from './vite.styles'

const alias = getPluginAliases([
  '@plugins',
  '@numen/plugin-sdk/testing',
  '@numen/plugin-sdk',
  '@numen/protocol/numen/v1',
  '@numen/protocol',
  '@numen/ui/styles.css',
  '@numen/ui',
  '@numen/wire',
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
