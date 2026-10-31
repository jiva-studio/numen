// Correctness and boundaries for the plugin manager. It imports components from `@numen/ui` and
// the plugin contract from `@numen/plugin-sdk`, and no other first-party package.

import tseslint from 'typescript-eslint'
import { createBaseConfig } from '../../tools/lint-layers/base-eslint.mjs'

const firstPartyImports = [
  {
    group: [
      '@numen/*',
      '@numen/*/**',
      '!@numen/ui',
      '!@numen/ui/**',
      '!@numen/plugin-sdk',
      '!@numen/plugin-sdk/**',
      // The same packages named by the path the install put them at.
      '**/node_modules/@numen/**',
    ],
    message:
      'the plugin manager knows the components and the plugin contract — take everything else by props and answer with events',
  },
  {
    group: ['**/apps/**', '**/libs/**'],
    message: 'the dependency runs apps → libs/plugin-manager → libs/ui and libs/plugin-sdk-ts',
  },
]

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname),

  {
    languageOptions: {
      parserOptions: {
        projectService: { allowDefaultProject: ['eslint.config.js'] },
      },
    },
  },

  {
    files: ['src/**/*.{ts,vue}'],
    rules: {
      'no-restricted-imports': ['error', { patterns: firstPartyImports }],
    },
  },
)
