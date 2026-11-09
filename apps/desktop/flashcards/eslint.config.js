// Correctness and boundary rules for the flashcards window, based on the rules
// `@numen/ui` already defines. No formatting rules are enabled: formatting is
// done manually, and a formatting rule would rewrite files.
//
// The library's ban on importing the protocol does not apply to this package:
// the shell requests the plugins of its target from the window.

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

const layerElements = createLayerElements({ order: layerOrder, sliceLayers })
const layerPolicies = createLayerPolicies({ order: layerOrder, sliceLayers, indexOnly: true })

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, {
    extraIgnores: ['eslint.config.js', 'vitest.config.ts'],
  }),

  // FSD Layer Hierarchy and boundaries enforcement
  createLayerConfig({
    boundaries,
    elements: layerElements,
    policies: layerPolicies,
    categories: [
      { category: 'public', pattern: 'src/*/*/index.ts' },
      { category: 'entry', pattern: 'src/main.ts' },
    ],
  }),

  // Anything with a lifetime a test must hold still is a port. The pure core
  // computes what is drawn from what it is given, so it may not ask the machine
  // what time it is, what the window measures, or what comes next.
  createPortConfig({ files: ['src/**/*.ts'], ignores: ['src/**/*.test.ts'] }),
)
