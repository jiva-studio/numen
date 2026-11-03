// The repo's rules for TypeScript, and the rules that make a test that cannot fail fail the lint:
// a step behind a condition, an expectation behind a condition, a test with no expectation, a
// fixed wait, a forced click, and an `.or(...)` locator that matches whichever draws first.

import globals from 'globals'
import tseslint from 'typescript-eslint'
import playwright from 'eslint-plugin-playwright'
import { createBaseConfig } from '../../../tools/lint-layers/base-eslint.mjs'

const specs = ['tests/**/*.spec.ts', 'flashcards/**/*.spec.ts']

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, {
    // Every file here is test code: it reads the clock and the dice for names no run repeats.
    portIgnores: ['**/*.ts'],
    extraIgnores: ['playwright-report/**', 'test-results/**'],
    // A module with a .d.mts beside it is left out of scripts/tsconfig.json, which takes the declaration.
    allowDefaultProject: ['scripts/shards.mjs', 'scripts/stand-ins.mjs'],
  }),

  {
    files: ['**/*.{ts,mjs}'],
    languageOptions: { globals: { ...globals.node, ...globals.browser } },
  },

  {
    // The runners print to the terminal that started them.
    files: ['scripts/**'],
    rules: { 'no-console': 'off' },
  },

  {
    ...playwright.configs['flat/recommended'],
    files: specs,
  },

  {
    files: specs,
    rules: {
      'playwright/no-conditional-in-test': 'error',
      'playwright/no-conditional-expect': 'error',
      // A helper named expect<Something> holds the expectations of the tests that call it.
      'playwright/expect-expect': ['error', { assertFunctionPatterns: ['^expect[A-Z]'] }],
      'playwright/no-skipped-test': 'error',
      'playwright/no-focused-test': 'error',
      'playwright/no-wait-for-timeout': 'error',
      'playwright/no-force-option': 'error',
      'playwright/no-useless-not': 'error',
      'playwright/valid-expect': 'error',
      'playwright/prefer-web-first-assertions': 'error',
      'no-restricted-syntax': [
        'error',
        {
          selector: 'CallExpression[callee.property.name="or"]',
          message:
            'a locator joined with .or() passes on whichever draws first — name the one outcome the test is about',
        },
      ],
    },
  },
)
