// Correctness and boundaries for the window, on the rules `@numen/ui` already
// states. Nothing here is about how the code looks: the formatting is done by
// hand and a rule with an opinion about it would start rewriting files nobody
// asked it to.
//
// The library's ban on importing the domain is the one rule that stops at this
// package: a window is the thing that knows what a vault is.

import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import { createBaseConfig, lifetimes } from '../../../tools/lint-layers/base-eslint.mjs'
import { createLayerElements, createLayerPolicies } from '../../../tools/lint-layers/layers.mjs'

const layerOrder = ['shared', 'entities', 'features', 'widgets', 'pages', 'app']
const sliceLayers = ['entities', 'features', 'widgets', 'pages']

// A plugin is an element of its own, and no policy reaches it: the window
// loads a plugin over HTTP and imports nothing of it.
const pluginElement = { type: 'plugin', pattern: '../../../plugins/*', capture: ['name'] }

const layerElements = createLayerElements({
  order: layerOrder,
  sliceLayers,
  extra: [{ type: 'testing', pattern: 'src/testing' }, pluginElement],
})

const layerPolicies = createLayerPolicies({
  order: layerOrder,
  sliceLayers,
  indexOnly: true,
  extra: [
    { allow: { to: { element: { type: 'testing' } } } },
    { from: { element: { type: 'testing' } }, allow: { to: { element: { type: layerOrder } } } },
  ],
})

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
        { category: 'public', pattern: 'src/*/*/index.ts' },
        { category: 'entry', pattern: 'src/main.ts' },
        { category: 'test', pattern: 'src/**/*.test.ts' },
        { category: 'story', pattern: 'src/**/*.stories.ts' },
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
      'src/testing/**',
      // The ports' own defaults, which is where the browser is allowed in.
      'src/entities/note/model/notes.ts',
      'src/app/window/useWindowKinds.ts',
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

  {
    files: ['src/testing/**', '.storybook/**'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',
      '@typescript-eslint/no-floating-promises': 'off',
      '@typescript-eslint/no-misused-promises': 'off',
      '@typescript-eslint/await-thenable': 'off',
      'no-console': 'off',
      'require-yield': 'off',
    },
  },
)
