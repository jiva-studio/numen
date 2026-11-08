// The repo's rules for TypeScript, and the rule of the steps: a step says what happens and calls
// the person driver; how it is done on the screen stays in drivers/. A step with a locator, a key
// or a page in it is a scenario written in code.

import globals from 'globals'
import tseslint from 'typescript-eslint'
import { createBaseConfig } from '../../../tools/lint-layers/base-eslint.mjs'

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, {
    // The driver reads the clock and the dice for names no run repeats.
    portIgnores: ['**/*.ts'],
    allowDefaultProject: ['scripts/*.mjs', 'vitest.config.mjs'],
    extraIgnores: ['playwright-report/**', 'test-results/**', '.features-gen/**'],
  }),

  {
    files: ['**/*.{ts,mjs}'],
    languageOptions: { globals: { ...globals.node, ...globals.browser } },
  },

  {
    files: ['scripts/**'],
    rules: { 'no-console': 'off' },
  },

  {
    files: ['steps/**/*.ts'],
    rules: {
      'max-lines-per-function': ['error', { max: 6, skipBlankLines: true, skipComments: true }],
      'max-params': ['error', 5],
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['playwright/test', 'playwright', '@playwright/test', '**/tests/support/**'],
              message: 'a step reaches the screen through the person driver (drivers/)',
              allowTypeImports: true,
            },
          ],
        },
      ],
      'no-restricted-syntax': [
        'error',
        {
          selector:
            'CallExpression[callee.property.name=/^(locator|getBy[A-Za-z]+|click|dblclick|fill|press|type|check|uncheck|hover|goto|reload|waitFor[A-Za-z]*|evaluate|keyboard|mouse)$/]',
          message: 'a step does not drive the page: add a method to the person driver and call it',
        },
        {
          selector: 'Property[key.name=/^(page|context|browser|request)$/]',
          message: 'a step takes the person driver, not the page: destructure `person`',
        },
        {
          selector: 'MemberExpression[property.name=/^(page|keyboard|mouse)$/]',
          message: 'a step does not reach the page: call the person driver',
        },
      ],
    },
  },
)
