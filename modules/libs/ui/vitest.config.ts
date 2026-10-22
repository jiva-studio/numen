import { fileURLToPath, URL } from 'node:url'
import { coverageConfigDefaults, defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import { getStoryBrowser } from './.storybook/browser'

const src = fileURLToPath(new URL('./src', import.meta.url))

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
          include: ['src*.test.ts', '.storybook*.test.ts'],
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
          browser: getStoryBrowser(),
        },
      },
    ],
  },
})
