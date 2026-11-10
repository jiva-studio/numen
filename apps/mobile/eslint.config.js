// Correctness and boundaries for the phone, on the rules `@numen/ui` already
// defines. Nothing here concerns code style: formatting is manual, and a style
// rule would rewrite files.
//
// The library's ban on importing the domain is the one rule that does not apply
// to this package: an application is where the vault concept belongs.

import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import { createBaseConfig, createPortConfig } from '../../tools/lint-layers/base-eslint.mjs'
import {
  createLayerConfig,
  createLayerElements,
  createLayerPolicies,
} from '../../tools/lint-layers/layers.mjs'

const layerOrder = ['shared', 'entities', 'features', 'pages', 'app']
const sliceLayers = ['entities', 'features', 'pages']

const layerElements = createLayerElements({ order: layerOrder, sliceLayers })
const layerPolicies = createLayerPolicies({ order: layerOrder, sliceLayers, indexOnly: true })

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, {
    extraIgnores: ['android/**'],
    // The `*.config.ts` files are in tsconfig.json; only this file is outside it.
    defaultProject: ['eslint.config.js'],
  }),

  // FSD layer hierarchy and boundaries enforcement
  createLayerConfig({
    boundaries,
    elements: layerElements,
    policies: layerPolicies,
    categories: [
      { category: 'public', pattern: 'src/*/*/index.ts' },
      { category: 'entry', pattern: 'src/main.ts' },
    ],
  }),

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
  createPortConfig({ files: ['src/**/*.ts'], ignores: ['src/**/*.test.ts'] }),
)
