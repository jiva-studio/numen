import globals from 'globals'
import tseslint from 'typescript-eslint'
import { createPlainLibraryConfig } from '../../../tools/lint-layers/base-eslint.mjs'

export default tseslint.config(
  ...createPlainLibraryConfig(import.meta.dirname),

  // The generated messages are linted under every rule but the file length, which follows the schema. Every
  // file opens with the generator's own disable comment, which most files do not need.
  {
    files: ['src/**/*_pb.ts'],
    linterOptions: { reportUnusedDisableDirectives: 'off' },
    rules: { 'max-lines': 'off', 'numen/no-unknown': 'off' },
  },

  // The generators are node scripts outside the TypeScript project.
  {
    files: ['**/*.mjs'],
    ...tseslint.configs.disableTypeChecked,
    languageOptions: {
      ...tseslint.configs.disableTypeChecked.languageOptions,
      globals: globals.node,
    },
  },
)
