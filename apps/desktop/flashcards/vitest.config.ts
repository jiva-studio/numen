/**
 * How the window's tests are run: in a document, so a component can be mounted and asked what it
 * drew. The build in `vite.config.ts` writes into the Go package's assets and has nothing to say
 * about it.
 */
import { fileURLToPath, URL } from 'node:url'
import { coverageConfigDefaults, defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

const alias = {
  '@': fileURLToPath(new URL('./src', import.meta.url)),
  '@numen/plugin-sdk': fileURLToPath(
    new URL('../../../libs/plugin-sdk-ts/src/index.ts', import.meta.url),
  ),
  '@numen/protocol': fileURLToPath(new URL('../../../libs/protocol/index.ts', import.meta.url)),
  '@numen/wire': fileURLToPath(new URL('../../../libs/wire/index.ts', import.meta.url)),
  '@numen/ui/styles.css': fileURLToPath(
    new URL('../../../libs/ui/core/dist/numen-ui.css', import.meta.url),
  ),
  '@numen/ui': fileURLToPath(new URL('../../../libs/ui/core/dist/index.js', import.meta.url)),
}

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
