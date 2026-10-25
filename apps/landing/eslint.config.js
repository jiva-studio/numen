// Correctness for the site: the frontmatter of each page and the node scripts
// that name the stories and take the pictures. Nothing here is about how the
// code looks: the formatting is done by hand and a rule with an opinion about
// it would start rewriting files nobody asked it to.

import js from '@eslint/js'
import globals from 'globals'
import astro from 'eslint-plugin-astro'
import tseslint from 'typescript-eslint'

/** A parameter named and unused is a signature to honour; a leading underscore says so. */
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
    rules: { 'no-unused-vars': unused },
  },
)
