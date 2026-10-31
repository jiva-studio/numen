import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

/**
 * `@numen/ui` is imported through its build, the way a window does: run `npm run build` in
 * `libs/ui/core` first.
 */
const alias = {
  '@numen/plugin-sdk/testing': fileURLToPath(
    new URL('../plugin-sdk-ts/src/testing/index.ts', import.meta.url),
  ),
  '@numen/plugin-sdk': fileURLToPath(new URL('../plugin-sdk-ts/src/index.ts', import.meta.url)),
  '@numen/protocol': fileURLToPath(new URL('../protocol/index.ts', import.meta.url)),
  '@numen/ui/styles.css': fileURLToPath(new URL('../ui/core/dist/numen-ui.css', import.meta.url)),
  '@numen/ui': fileURLToPath(new URL('../ui/core/dist/index.js', import.meta.url)),
}

export default defineConfig({
  plugins: [vue()],
  resolve: { alias },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    testTimeout: 30_000,
  },
})
