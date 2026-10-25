/**
 * How the phone's tests are run.
 *
 * A test runs in a document, so a component can be mounted and asked what it
 * drew. The build in `vite.config.ts` writes the page the WebView loads and has
 * nothing to say about that.
 */
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    name: 'unit',
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    // A test here waits on an import out of one install shared by every
    // package, which the default five seconds does not cover on a cold cache.
    testTimeout: 30_000,
  },
})
