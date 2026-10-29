// Correctness and boundary rules for the flashcards window, based on the rules
// `@numen/ui` already defines. No formatting rules are enabled: formatting is
// done by hand, and a formatting rule would rewrite files.
//
// The library's ban on importing the protocol does not apply to this package:
// the shell requests the plugins of its target from the window.

import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import { createBaseConfig, lifetimes } from '../../../tools/lint-layers/base-eslint.mjs'
import { createLayerElements, createLayerPolicies } from '../../../tools/lint-layers/layers.mjs'

const layerOrder = ['shared', 'entities', 'features', 'widgets', 'pages', 'app']
const sliceLayers = ['entities', 'features', 'widgets', 'pages']

const layerElements = createLayerElements({ order: layerOrder, sliceLayers })
const layerPolicies = createLayerPolicies({ order: layerOrder, sliceLayers, indexOnly: true })

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, {
    extraIgnores: ['eslint.config.js', 'vitest.config.ts'],
  }),

  // FSD Layer Hierarchy and boundaries enforcement
  {
    files: ['src/**/*.{ts,vue}'],
    plugins: { boundaries },
    settings: {
      'import/resolver': { typescript: { alwaysTryTypes: true } },
      'boundaries/elements': layerElements,
      'boundaries/files': [
        { category: 'public', pattern: 'src/*/*/index.ts' },
        { category: 'entry', pattern: 'src/main.ts' },
      ],
    },
    rules: {
      'boundaries/no-unknown-files': 'error',
      'boundaries/dependencies': ['error', { default: 'disallow', policies: layerPolicies }],
    },
  },

  // Anything with a lifetime a test must hold still is a port. The pure core
  // computes what is drawn from what it is given, so it may not ask the machine
  // what time it is, what the window measures, or what comes next.
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
