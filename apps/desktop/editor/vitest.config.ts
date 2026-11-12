/**
 * How the window's tests are run.
 *
 * A unit test runs in a document, so a component can be mounted and queried for
 * what it drew. A story runs in a real browser, the only environment that can
 * determine browser behavior: what is clipped, where a drag lands, what a
 * blend produces. The build in `vite.config.ts` writes into the Go package's
 * assets and does not apply to either.
 */
import { fileURLToPath, URL } from 'node:url'
import { coverageConfigDefaults, defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { createWindowAliases } from '@numen/workbench/vite'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import { getStoryBrowser } from '../../../libs/ui/components/.storybook/browser'

const alias = createWindowAliases(import.meta.url, {
  '@numen/core-client/testing': '../../../libs/core/client/src/testing/index.ts',
  '@numen/plugin-manager/testing': '../../../libs/plugin/manager/src/testing/index.ts',
  '@numen/plugin-manager': '../../../libs/plugin/manager/src/index.ts',
})

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
