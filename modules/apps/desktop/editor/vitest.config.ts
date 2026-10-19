/**
 * How the window's tests are run.
 *
 * A unit test runs in a document, so a component can be mounted and asked what
 * it drew. A story runs in a real browser, which is the only thing that can
 * answer what a browser decides: what is clipped, where a drag lands, what a
 * blend came to. The build in `vite.config.ts` writes into the Go package's
 * assets and has nothing to say about either.
 */
import { fileURLToPath, URL } from 'node:url'
import { coverageConfigDefaults, defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import { getStoryBrowser } from '../../../libs/ui/.storybook/browser'

const alias = {
  '@': fileURLToPath(new URL('./src', import.meta.url)),
  '@plugins': fileURLToPath(new URL('../../../plugins', import.meta.url)),
  '@numen/plugin-sdk': fileURLToPath(
    new URL('../../../libs/plugin-sdk/src/index.ts', import.meta.url),
  ),
  '@numen/protocol': fileURLToPath(new URL('../../../libs/protocol/index.ts', import.meta.url)),
  '@numen/wire': fileURLToPath(new URL('../../../libs/wire/index.ts', import.meta.url)),
  '@numen/ui/styles.css': fileURLToPath(
    new URL('../../../libs/ui/dist/numen-ui.css', import.meta.url),
  ),
  '@numen/ui': fileURLToPath(new URL('../../../libs/ui/dist/index.js', import.meta.url)),
}

export default defineConfig({
  resolve: { alias },
  test: {
    // The floor the tests stand on. Each number is where the suite is today,
    // so a change may only raise it.
    coverage: {
      include: ['src/**'],
      // A story is the corpus a test run draws, not code under test.
      exclude: [...coverageConfigDefaults.exclude, '**/*.stories.ts'],
      reporter: ['text-summary'],
      thresholds: { statements: 90, branches: 85, functions: 83, lines: 91 },
    },
    projects: [
      {
        plugins: [vue()],
        resolve: { alias },
        test: {
          name: 'unit',
          environment: 'jsdom',
          include: ['src/**/*.test.ts'],
          setupFiles: ['./src/testing/noLayout.ts'],
          testTimeout: 30_000,
        },
      },
      {
        plugins: [
          vue(),
          storybookTest({
            configDir: fileURLToPath(new URL('./.storybook', import.meta.url)),
          }),
        ],
        resolve: { alias },
        test: {
          name: 'stories',
          // Every story is drawn twice, and the frames of all of them come off
          // one machine.
          testTimeout: 30_000,
          browser: getStoryBrowser(),
        },
      },
    ],
  },
})
