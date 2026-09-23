import { accessSync, constants } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath, URL } from 'node:url'
import { coverageConfigDefaults, defineConfig } from 'vitest/config'
import type { BrowserCommand } from 'vitest/node'
import { playwright } from '@vitest/browser-playwright'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'

const src = fileURLToPath(new URL('./src', import.meta.url))

/** A place on the page the browser drives the pointer to. */
interface Position {
  readonly x: number
  readonly y: number
}

/**
 * The pointer put down at one place, carried to another and lifted, by the
 * browser itself. What a drag leaves behind — a selection, a capture — is then
 * the browser's own.
 */
const sweep: BrowserCommand<[from: Position, to: Position]> = async ({ page }, from, to) => {
  await page.mouse.move(from.x, from.y)
  await page.mouse.down()
  await page.mouse.move(to.x, to.y)
  await page.mouse.up()
}

declare module 'vitest/browser' {
  interface BrowserCommands {
    sweep: (from: Position, to: Position) => Promise<void>
  }
}

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
    // The floor the unit tests stand on. Each number is where the suite is
    // today, so a change may only raise it.
    coverage: {
      include: ['src/**'],
      // A story is the corpus a test run draws, not code under test.
      exclude: [...coverageConfigDefaults.exclude, '**/*.stories.ts'],
      reporter: ['text-summary'],
      thresholds: { statements: 89, branches: 84, functions: 90, lines: 91 },
    },
    projects: [
      {
        plugins: [vue()],
        resolve: { alias: { '@': src } },
        test: {
          name: 'unit',
          environment: 'jsdom',
          include: ['src/**/*.test.ts', '.storybook/**/*.test.ts'],
          testTimeout: 30_000,
        },
      },
      {
        plugins: [
          vue(),
          tailwindcss(),
          storybookTest({
            configDir: fileURLToPath(new URL('./.storybook', import.meta.url)),
          }),
        ],
        resolve: { alias: { '@': src } },
        test: {
          name: 'stories',
          // Every story is drawn twice, and the frames of all of them come off
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
            commands: { sweep },
          },
        },
      },
    ],
  },
})
