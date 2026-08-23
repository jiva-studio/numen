/**
 * How the window's tests are run.
 *
 * A test runs in a document, so a component can be mounted and asked what it
 * drew. The build in `vite.config.ts` writes into the Go package's assets and
 * has nothing to say about that.
 */
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    name: 'unit',
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
  },
})
