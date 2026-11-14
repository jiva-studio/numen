import tseslint from 'typescript-eslint'
import { createBaseConfig } from '../../../tools/lint-layers/base-eslint.mjs'

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, {
    extraIgnores: [
      'eslint.config.js',
      'vitest.config.ts',
      'stories/vitest.config.ts',
      'stories/.storybook/**',
    ],
  }),

  // The generated messages open with the generator's own disable comment, which they do not need.
  {
    files: ['src/**/*_pb.ts'],
    linterOptions: { reportUnusedDisableDirectives: 'off' },
  },
)
