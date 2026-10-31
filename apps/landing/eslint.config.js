// Correctness checks for the site: the frontmatter of each page and the node
// scripts that name the stories and take the screenshots. Formatting is not
// checked: it is applied manually, and a formatting rule would rewrite files.

import js from '@eslint/js'
import globals from 'globals'
import astro from 'eslint-plugin-astro'
import tseslint from 'typescript-eslint'
import { createStrictRules, createStrictScriptRules } from '../../tools/lint-layers/base-eslint.mjs'

/** A parameter that is named and unused is part of a required signature; a leading underscore marks it. */
const unused = [
  'error',
  { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
]

export default tseslint.config(
  { ignores: ['dist/**', '.astro/**', 'node_modules/**'] },

  js.configs.recommended,
  // The frontmatter is TypeScript, and nothing else in this package is.
  ...tseslint.configs.recommended.map((one) => ({ ...one, files: ['**/*.ts', '**/*.astro'] })),
  astro.configs.recommended,

  {
    files: ['**/*.astro'],
    rules: {
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': unused,
    },
  },

  {
    files: ['**/*.mjs'],
    languageOptions: { globals: globals.node },
    rules: { 'no-unused-vars': unused, ...createStrictScriptRules() },
  },

  {
    files: ['**/*.ts'],
    languageOptions: {
      parserOptions: { projectService: true, tsconfigRootDir: import.meta.dirname },
    },
  },

  {
    files: ['**/*.astro'],
    languageOptions: {
      parserOptions: { project: true, tsconfigRootDir: import.meta.dirname },
    },
  },

  { files: ['**/*.ts', '**/*.astro'], rules: createStrictRules() },
)
