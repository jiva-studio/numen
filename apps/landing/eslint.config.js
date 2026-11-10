// Correctness checks for the site on the shared base: the frontmatter of each page, the demo
// window and the node scripts that name the stories and take the screenshots.
// Formatting is not checked: it is applied manually, and a formatting rule
// would rewrite files.

import globals from 'globals'
import astro from 'eslint-plugin-astro'
import tseslint from 'typescript-eslint'
import { createBaseConfig, createStrictScriptRules } from '../../tools/lint-layers/base-eslint.mjs'

/** A parameter that is named and unused is part of a required signature; a leading underscore marks it. */
const unused = [
  'error',
  { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
]

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, {
    extraIgnores: ['.astro/**'],
    defaultProject: ['eslint.config.js'],
  }),

  astro.configs.recommended,

  {
    files: ['**/*.astro'],
    languageOptions: {
      parserOptions: { projectService: false, project: true, tsconfigRootDir: import.meta.dirname },
    },
  },

  // The node scripts are plain JavaScript outside tsconfig.json: no type information.
  {
    ...tseslint.configs.disableTypeChecked,
    files: ['**/*.mjs'],
    languageOptions: {
      ...tseslint.configs.disableTypeChecked.languageOptions,
      globals: globals.node,
      parserOptions: { projectService: false },
    },
    rules: {
      ...tseslint.configs.disableTypeChecked.rules,
      '@typescript-eslint/no-unused-vars': 'off',
      'no-unused-vars': unused,
      ...createStrictScriptRules(),
    },
  },
)
