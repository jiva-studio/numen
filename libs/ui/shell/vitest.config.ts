import { fileURLToPath, URL } from 'node:url'
import { coverageConfigDefaults, defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import { createLayeredStylesPlugin } from '@numen/ui/vite'
import { getStoryBrowser } from '../core/.storybook/browser'

const src = fileURLToPath(new URL('./src', import.meta.url))

/**
 * The shell reaches `@numen/ui` through its build, the way a window does: run
 * `npm run build` in `libs/ui/core` first.
 */
const alias = [
  { find: /^@\//, replacement: `${src}/` },
  {
    find: /^@numen\/ui\/styles\.css$/,
    replacement: fileURLToPath(new URL('../core/dist/numen-ui.css', import.meta.url)),
  },
  {
    find: /^@numen\/ui$/,
    replacement: fileURLToPath(new URL('../core/dist/index.js', import.meta.url)),
  },
]

export default defineConfig({
  test: {
    // The floor the unit tests stand on. Each number is where the suite is
    // today, so a change may only raise it.
    coverage: {
      include: ['src/**'],
      exclude: [...coverageConfigDefaults.exclude, '**/*.stories.ts', 'src/vite.ts'],
      reporter: ['text-summary'],
      thresholds: { statements: 95, branches: 91, functions: 94, lines: 95 },
    },
    projects: [
      {
        plugins: [vue()],
        resolve: { alias },
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
          tailwindcss(),
          createLayeredStylesPlugin(`${src}/styles.css`),
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
