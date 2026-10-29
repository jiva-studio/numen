/**
 * How the phone's tests are run.
 *
 * A test runs in a document, so a component can be mounted and its rendered
 * output inspected. The build in `vite.config.ts` writes the page the WebView
 * loads and does not provide that.
 */
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@numen/plugin-sdk': fileURLToPath(
        new URL('../../libs/plugin-sdk-ts/src/index.ts', import.meta.url),
      ),
    },
  },
  test: {
    name: 'unit',
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    // A test here waits on an import out of one install shared by every
    // package, which the default five seconds does not cover on a cold cache.
    testTimeout: 30_000,
  },
})
