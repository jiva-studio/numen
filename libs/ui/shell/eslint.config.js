// Correctness and boundaries, as in `libs/ui/core`. The shell imports components
// through `@numen/ui` and no other first-party package.

import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import { createBaseConfig, createPortConfig } from '../../../tools/lint-layers/base-eslint.mjs'
import {
  createLayerConfig,
  createLayerElements,
  createLayerPolicies,
  createOwnImportConfig,
} from '../../../tools/lint-layers/layers.mjs'

const layerOrder = ['shared', 'features', 'screens']
const sliceLayers = ['features']

const layerElements = createLayerElements({ order: layerOrder, sliceLayers })
const layerPolicies = createLayerPolicies({ order: layerOrder, sliceLayers, indexOnly: true })

/** The files that stand outside the layers. */
const outsideLayers = ['src/**/*.stories.ts', 'src/**/fixtures/**', 'src/vite.ts']

const domainImports = [
  {
    group: [
      '@numen/*',
      '@numen/*/**',
      '!@numen/ui',
      // The same packages named by the path the install put them at.
      '**/node_modules/@numen/**',
    ],
    message:
      'the shell knows the components and nothing of the domain or the wire — take props in a drawing vocabulary and emit opaque identifiers',
  },
  {
    group: ['**/apps/**', '**/libs/**'],
    message: 'the dependency runs apps → libs/ui/shell → libs/ui/core, never back',
  },
]

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, { defaultProject: ['eslint.config.js'] }),

  createLayerConfig({
    boundaries,
    elements: layerElements,
    policies: layerPolicies,
    categories: [
      { category: 'public', pattern: 'src/features/*/index.ts' },
      { category: 'entry', pattern: 'src/index.ts' },
    ],
    ignores: outsideLayers,
    rules: { 'no-restricted-imports': ['error', { patterns: domainImports }] },
  }),

  createOwnImportConfig({
    layer: 'features',
    patterns: domainImports,
    group: ['@/features/**', '@/screens/**'],
    message: 'a slice imports its own files by relative path and no other slice',
    outsideLayers,
  }),

  // Anything with a lifetime a test must hold still is a port.
  createPortConfig({
    files: ['src/**/*.ts'],
    ignores: ['src/**/*.test.ts', 'src/**/*.stories.ts', 'src/**/fixtures/**'],
  }),
)
