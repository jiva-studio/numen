// Correctness and boundaries for the flashcards window, on the rules
// `@numen/ui` already states. Nothing here is about how the code looks: the
// formatting is done by hand and a rule with an opinion about it would start
// rewriting files nobody asked it to.
//
// The library's ban on importing the domain is the one rule that stops at this
// package: a window is the thing that knows what a vault is.

import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import { createBaseConfig, lifetimes } from '../../../tools/lint-layers/base-eslint.mjs'
import { createLayerElements, createLayerPolicies } from '../../../tools/lint-layers/layers.mjs'

const layerOrder = ['shared', 'entities', 'features', 'widgets', 'pages', 'app']
const sliceLayers = ['entities', 'features', 'widgets', 'pages']

const layerElements = createLayerElements({ order: layerOrder, sliceLayers })
const layerPolicies = createLayerPolicies({ order: layerOrder, sliceLayers })

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname),

  // FSD Layer Hierarchy and boundaries enforcement
  {
    files: ['src/**/*.{ts,vue}'],
    plugins: { boundaries },
    settings: {
      'import/resolver': { typescript: { alwaysTryTypes: true } },
      'boundaries/elements': layerElements,
      'boundaries/files': [
        { category: 'crossImport', pattern: 'src/*/*/@x/*' },
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
    ignores: [
      'src/**/*.test.ts',
      'src/**/*.stories.ts',
      // A port's own default, which is where the browser is allowed in.
      'src/pages/session/model/session.ts',
      'src/pages/decks/model/reviewDays.ts',
    ],
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
