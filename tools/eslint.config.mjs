// Correctness and strictness for the node scripts under tools/. Formatting is not checked.

import js from '@eslint/js'
import globals from 'globals'
import { createStrictScriptRules } from './lint-layers/base-eslint.mjs'

/** A parameter that is named and unused is part of a required signature; a leading underscore marks it. */
const unused = [
  'error',
  { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
]

export default [
  { ignores: ['**/node_modules/**'] },

  js.configs.recommended,

  {
    files: ['**/*.mjs', '**/*.js'],
    languageOptions: { globals: globals.node },
    rules: { 'no-unused-vars': unused, ...createStrictScriptRules() },
  },
]
