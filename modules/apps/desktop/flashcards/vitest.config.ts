/**
 * How the window's tests are run.
 *
 * A unit test runs in a document, because the client this page is built on
 * reads the address the window was served from. A story runs in a real
 * browser, which is the only thing that can answer what a browser decides:
 * what is clipped, what a blend came to, where a line broke. The build in
 * `vite.config.ts` writes into the Go package's assets and has nothing to say
 * about either.
 */
import { fileURLToPath, URL } from 'node:url'
import { coverageConfigDefaults, defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import { getStoryBrowser } from '../../../libs/ui/.storybook/browser'

/** `@/` is the window's own source, as `vite.config.ts` and `tsconfig.json` say. */
const alias = { '@': fileURLToPath(new URL('./src', import.meta.url)) }

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
      thresholds: { statements: 83, branches: 80, functions: 78, lines: 85 },
    },
    projects: [
      {
        plugins: [vue()],
        resolve: { alias },
        test: {
          name: 'unit',
          environment: 'jsdom',
          include: ['src*.test.ts'],
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
          // Every story is drawn twice, and the frames of both of them come off
          // one machine.
          testTimeout: 30_000,
          browser: getStoryBrowser(),
        },
      },
    ],
  },
})
