import tseslint from 'typescript-eslint'
import {
  createBaseConfig,
  createNoUnknownContractConfig,
} from '../../../tools/lint-layers/base-eslint.mjs'

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, {
    extraIgnores: [
      'eslint.config.js',
      'vitest.config.ts',
      'stories/vitest.config.ts',
      'stories/.storybook/**',
    ],
  }),
  createNoUnknownContractConfig(['src/**/*.ts']),
)
