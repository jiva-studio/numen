import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

const alias = {
  '@': fileURLToPath(new URL('../../apps/desktop/editor/src', import.meta.url)),
  '@plugins': fileURLToPath(new URL('..', import.meta.url)),
  '@numen/plugin-sdk/testing': fileURLToPath(
    new URL('../../libs/plugin-sdk/src/testing/index.ts', import.meta.url),
  ),
  '@numen/plugin-sdk': fileURLToPath(
    new URL('../../libs/plugin-sdk/src/index.ts', import.meta.url),
  ),
  '@numen/ui/styles.css': fileURLToPath(
    new URL('../../libs/ui/dist/numen-ui.css', import.meta.url),
  ),
  '@numen/ui': fileURLToPath(new URL('../../libs/ui/dist/index.js', import.meta.url)),
  '@numen/wire': fileURLToPath(new URL('../../libs/wire/index.ts', import.meta.url)),
}

export default defineConfig({
  plugins: [vue()],
  resolve: { alias },
  test: {
    environment: 'jsdom',
  },
})
