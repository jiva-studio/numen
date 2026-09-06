// Correctness for the manual: the frontmatter of each component, the collection
// it is built from, and the node scripts that write the pages and take the
// pictures. Nothing here is about how the code looks: the formatting is done by
// hand and a rule with an opinion about it would start rewriting files nobody
// asked it to.

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

  // The camera drives a browser, and what it asks the page to wait for is
  // written here and evaluated there.
  {
    files: ['tools/shoot.mjs'],
    languageOptions: { globals: { ...globals.node, ...globals.browser } },
  },
)
