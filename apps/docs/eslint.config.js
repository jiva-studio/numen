// Correctness rules for the manual: the frontmatter of each component, the
// collection it is built from, and the node scripts that write the pages and
// take the screenshots. Formatting is done manually, so no rule here concerns
// code style.

import js from '@eslint/js'
import globals from 'globals'
import astro from 'eslint-plugin-astro'
import tseslint from 'typescript-eslint'

/** An unused parameter is a required signature; a leading underscore marks it. */
const unused = [
  'error',
  { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
]

export default tseslint.config(
  { ignores: ['dist/**', '.astro/**', 'node_modules/**'] },

  js.configs.recommended,
  // The frontmatter and the collection are TypeScript, and nothing else here is.
  ...tseslint.configs.recommended.map((one) => ({ ...one, files: ['**/*.ts', '**/*.astro'] })),
  astro.configs.recommended,

  {
    files: ['**/*.ts', '**/*.astro'],
    rules: {
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': unused,
    },
  },

  {
    files: ['**/*.mjs'],
    languageOptions: { globals: globals.node },
    rules: { 'no-unused-vars': unused },
  },

)
