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
import { accessSync, constants } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath, URL } from 'node:url'
import { coverageConfigDefaults, defineConfig } from 'vitest/config'
import { playwright } from '@vitest/browser-playwright'
import vue from '@vitejs/plugin-vue'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'

const CANDIDATES = ['google-chrome-stable', 'google-chrome', 'chromium', 'chromium-browser']

/**
 * A browser already on the machine; undefined uses Playwright's own.
 *
 * A run under `CI` takes the pinned build unless it was pointed at one, so the
 * version the stories are drawn in is the version the workflow fetched.
 */
function systemChrome(): string | undefined {
  const explicit = process.env['CHROME_PATH']
  if (explicit) return explicit
  if (process.env['CI']) return undefined

  for (const directory of (process.env['PATH'] ?? '').split(':')) {
    if (!directory) continue
    for (const name of CANDIDATES) {
      const candidate = join(directory, name)
      try {
        accessSync(candidate, constants.X_OK)
        return candidate
      } catch {
        continue
      }
    }
  }
  return undefined
}

const chrome = systemChrome()

export default defineConfig({
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
        test: {
          name: 'unit',
          environment: 'jsdom',
          include: ['src/**/*.test.ts'],
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
        test: {
          name: 'stories',
          // Every story is drawn twice, and the frames of both of them come off
          // one machine.
          testTimeout: 30_000,
          browser: {
            enabled: true,
            headless: true,
            provider: playwright(),
            // The window is WebKit on a mac and on Linux, and Chromium on
            // Windows. Every story is rendered in both.
            instances: [
              {
                browser: 'chromium',
                ...(chrome
                  ? { provider: playwright({ launchOptions: { executablePath: chrome } }) }
                  : {}),
              },
              { browser: 'webkit' },
            ],
          },
        },
      },
    ],
  },
})
