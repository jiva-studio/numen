import { accessSync, constants } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'

const src = fileURLToPath(new URL('./src', import.meta.url))

const CANDIDATES = [
  'google-chrome-stable',
  'google-chrome',
  'chromium',
  'chromium-browser',
]

/** A browser already on the machine; undefined uses Playwright's own. */
function systemChrome(): string | undefined {
  const explicit = process.env['CHROME_PATH']
  if (explicit) return explicit

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
    projects: [
      {
        plugins: [vue()],
        resolve: { alias: { '@': src } },
        test: {
          name: 'unit',
          environment: 'jsdom',
          include: ['src/**/*.test.ts'],
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
          browser: {
            enabled: true,
            headless: true,
            provider: 'playwright',
            instances: [
              {
                browser: 'chromium',
                ...(chrome ? { launch: { executablePath: chrome } } : {}),
              },
            ],
          },
        },
      },
    ],
  },
})
