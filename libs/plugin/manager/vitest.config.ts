import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

/**
 * `@numen/ui` is imported through its build, the way a window does: run `npm run build` in
 * `libs/ui/components` first.
 */
const alias = {
  '@numen/ui/styles.css': fileURLToPath(
    new URL('../../ui/components/dist/numen-ui.css', import.meta.url),
  ),
  '@numen/ui': fileURLToPath(new URL('../../ui/components/dist/index.js', import.meta.url)),
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
