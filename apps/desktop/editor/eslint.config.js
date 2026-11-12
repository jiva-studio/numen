// Correctness and boundaries for the window, on the rules `@numen/ui` already
// states. Nothing here concerns code style: formatting is done manually and a
// formatting rule would start rewriting files.
//
// The library's ban on importing the domain is the one rule that stops at this
// package: a window is the layer that depends on the vault domain.

import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import { createBaseConfig, createPortConfig } from '../../../tools/lint-layers/base-eslint.mjs'
import {
  createLayerConfig,
  createLayerElements,
  createLayerPolicies,
} from '../../../tools/lint-layers/layers.mjs'

const layerOrder = ['shared', 'entities', 'features', 'pages', 'app']
const sliceLayers = ['entities', 'features', 'pages']

// A plugin is an element of its own, and no policy applies to it: the window
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
    // Test support reaches a slice through its index, as the code it supports does.
    ...layerOrder.map((type) => ({
      from: { element: { type: 'testing' } },
      allow: {
        to: {
          element: { type },
          ...(sliceLayers.includes(type) ? { file: { categories: 'public' } } : {}),
        },
      },
    })),
  ],
})

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, { allowDefaultProject: ['scripts/*.mjs'] }),

  // FSD Layer Hierarchy and boundaries enforcement
  createLayerConfig({
    boundaries,
    elements: layerElements,
    policies: layerPolicies,
    categories: [
      { category: 'public', pattern: 'src/*/*/index.ts' },
      { category: 'entry', pattern: 'src/main.ts' },
      { category: 'test', pattern: 'src/**/*.test.ts' },
      { category: 'story', pattern: 'src/**/*.stories.ts' },
    ],
  }),

  // Anything with a lifetime a test must hold still is a port. The pure core
  // computes what is drawn from what it is given, so it may not ask the machine
  // what time it is, what the window measures, or what comes next.
  createPortConfig({
    files: ['src/**/*.ts'],
    ignores: ['src/**/*.test.ts', 'src/**/*.stories.ts', 'src/testing/**'],
  }),

  {
    files: ['src/testing/**', '.storybook/**'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',
      'no-console': 'off',
      'require-yield': 'off',
    },
  },
)
