// Correctness and boundaries for the phone, on the rules `@numen/ui` already
// defines. Nothing here concerns code style: formatting is manual, and a style
// rule would rewrite files.
//
// The library's ban on importing the domain is the one rule that does not apply
// to this package: an application is where the vault concept belongs.

import tseslint from 'typescript-eslint'
import { createBaseConfig, lifetimes } from '../../tools/lint-layers/base-eslint.mjs'

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, {
    extraIgnores: ['android/**'],
  }),

  // The `*.config.ts` files are in tsconfig.json; only this file is outside it.
  {
    languageOptions: {
      parserOptions: {
        projectService: {
          allowDefaultProject: ['eslint.config.js'],
        },
      },
    },
  },

  // Limits on complexity and file length for handwritten sources.
  {
    files: ['src/**/*.{ts,vue}'],
    ignores: ['src/**/*.test.ts'],
    rules: {
      'max-lines': ['error', { max: 250, skipBlankLines: false, skipComments: false }],
      complexity: ['error', 10],
      'max-depth': ['error', 3],
    },
  },

  // Anything with a lifetime a test must hold still is a port.
  {
    files: ['src/**/*.ts'],
    ignores: ['src/**/*.test.ts'],
    rules: {
      'no-restricted-globals': [
        'error',
        ...lifetimes.map(({ name, port }) => ({
          name,
          message: `${name} has a lifetime a test must hold still — take ${port}`,
        })),
      ],
      'no-restricted-syntax': [
        'error',
        {
          selector: 'MemberExpression[object.name="Date"][property.name="now"]',
          message: 'the clock is a port — take the now a Clock carries',
        },
        {
          selector: 'NewExpression[callee.name="Date"][arguments.length=0]',
          message: 'the clock is a port — take the now a Clock carries',
        },
        {
          selector: 'MemberExpression[object.name="Math"][property.name="random"]',
          message: 'the same input gives the same numbers on any machine on any day',
        },
        {
          selector: 'MemberExpression[property.name="getBoundingClientRect"]',
          message: 'what the window measures is a port — take it as a value',
        },
        {
          selector: 'MemberExpression[property.name="matchMedia"]',
          message: 'a reader who asks for less motion is answered in CSS',
        },
      ],
    },
  },
)
