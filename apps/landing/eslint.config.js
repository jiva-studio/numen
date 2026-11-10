// Correctness checks for the site: the frontmatter of each page, the demo
// window and the node scripts that name the stories and take the screenshots.
// Formatting is not checked: it is applied manually, and a formatting rule
// would rewrite files.

import js from '@eslint/js'
import globals from 'globals'
import astro from 'eslint-plugin-astro'
import tseslint from 'typescript-eslint'
import pluginVue from 'eslint-plugin-vue'
import { createStrictRules, createStrictScriptRules, numenPlugin } from '../../tools/lint-layers/base-eslint.mjs'

/** A parameter that is named and unused is part of a required signature; a leading underscore marks it. */
const unused = [
  'error',
  { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
]

const typed = ['**/*.ts', '**/*.vue', '**/*.astro']

export default tseslint.config(
  { ignores: ['dist/**', '.astro/**', 'node_modules/**'] },

  js.configs.recommended,
  // Everything but the node scripts is TypeScript.
  ...tseslint.configs.recommended.map((one) => ({ ...one, files: typed })),
  ...pluginVue.configs['flat/essential'],
  astro.configs.recommended,

  {
    files: typed,
    plugins: { numen: numenPlugin },
    languageOptions: {
      globals: globals.browser,
      parserOptions: { projectService: true, tsconfigRootDir: import.meta.dirname, extraFileExtensions: ['.vue'] },
    },
    rules: {
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': unused,
      'vue/multi-word-component-names': 'off',
      ...createStrictRules(),
      'numen/max-composable-return-properties': ['error', { max: 8 }],
    },
  },

  {
    files: typed,
    ignores: ['**/*.test.ts'],
    plugins: { numen: numenPlugin },
    rules: { 'numen/no-unknown': 'error' },
  },

  {
    files: ['**/*.vue'],
    languageOptions: { parserOptions: { parser: tseslint.parser } },
  },

  {
    files: ['**/*.astro'],
    languageOptions: {
      parserOptions: { projectService: false, project: true, tsconfigRootDir: import.meta.dirname },
    },
  },

  {
    files: ['**/*.mjs'],
    languageOptions: { globals: globals.node },
    rules: { 'no-unused-vars': unused, ...createStrictScriptRules() },
  },
)
