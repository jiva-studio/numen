import { resolve } from 'path'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@numen/protocol': resolve(import.meta.dirname, '../../libs/protocol/index.ts'),
      '@numen/plugin-sdk/testing': resolve(
        import.meta.dirname,
        '../../libs/plugin-sdk-ts/src/testing/mockContext.ts',
      ),
      '@numen/plugin-sdk': resolve(import.meta.dirname, '../../libs/plugin-sdk-ts/src/index.ts'),
      '@numen/wire': resolve(import.meta.dirname, '../../libs/wire/index.ts'),
    },
  },
  test: {
    environment: 'jsdom',
  },
})
