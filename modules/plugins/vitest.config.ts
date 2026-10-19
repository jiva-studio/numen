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
import { getStoryBrowser } from '../libs/ui/.storybook/browser'

const alias = {
  '@': fileURLToPath(new URL('../apps/desktop/editor/src', import.meta.url)),
  '@plugins': fileURLToPath(new URL('.', import.meta.url)),
  '@numen/plugin-sdk/testing': fileURLToPath(
    new URL('../libs/plugin-sdk/src/testing/index.ts', import.meta.url),
  ),
  '@numen/plugin-sdk': fileURLToPath(new URL('../libs/plugin-sdk/src/index.ts', import.meta.url)),
  '@numen/protocol': fileURLToPath(new URL('../libs/protocol/index.ts', import.meta.url)),
  '@numen/wire': fileURLToPath(new URL('../libs/wire/index.ts', import.meta.url)),
  '@numen/ui/styles.css': fileURLToPath(new URL('../libs/ui/dist/numen-ui.css', import.meta.url)),
  '@numen/ui': fileURLToPath(new URL('../libs/ui/dist/index.js', import.meta.url)),
}

export default defineConfig({
  plugins: [
    vue(),
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
