/**
 * How the window's tests are run: in a document, so a component can be mounted
 * and queried for what it drew. The build in `vite.config.ts` writes into the
 * Go package's assets and does not apply to it.
 */
import { coverageConfigDefaults, defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { createWindowAliases } from '@numen/workbench/vite'

const alias = createWindowAliases(import.meta.url, {
  '@numen/core-client/testing': '../../../libs/core/client/src/testing/index.ts',
})

export default defineConfig({
  plugins: [vue()],
  resolve: { alias },
  test: {
    coverage: {
      include: ['src/**'],
      exclude: [...coverageConfigDefaults.exclude],
      reporter: ['text-summary'],
    },
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    testTimeout: 30_000,
  },
})
