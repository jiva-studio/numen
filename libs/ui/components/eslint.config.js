// Correctness and boundaries. Nothing here concerns code style: formatting is
// manual, and a style rule would rewrite files.
//
// Three of the four groups below are the module's boundary made executable. A
// component has no knowledge of the domain, anything with a lifetime is a port,
// and a component that locates its own children uses a template ref.

import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import storybook from 'eslint-plugin-storybook'
import {
  createBaseConfig,
  createPortConfig,
  createPureSegmentConfig,
  createStoryConfig,
} from '../../../tools/lint-layers/base-eslint.mjs'
import {
  createDomainImports,
  createLibraryLayerConfigs,
  createOwnImportConfig,
} from '../../../tools/lint-layers/layers.mjs'

/** The one function declaration that reads the wall clock. */
const clockFunction = 'getNow'

/** The one function declaration that reads the monotonic clock. */
const monotonicFunction = 'getMonotonicNow'

const clockFunctions = { clockFunction, monotonicFunction }

/** The tests that read the wire's fixtures from `libs/core/protocol/testdata` by relative path. */
const testdataReaders = []

/** The files that stand outside the layers. */
const outsideLayers = [
  'src/**/*.stories.ts',
  'src/**/fixtures/**',
  'src/vite.ts',
  'src/vite.test.ts',
  ...testdataReaders,
]

const domainImports = createDomainImports({
  message:
    'a component knows nothing about the domain or the wire — take props in a drawing vocabulary and emit opaque identifiers',
  direction: 'the dependency runs apps → libs/ui/components, never back',
})

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, {
    ...clockFunctions,
    defaultProject: ['eslint.config.js'],
  }),

  // UI layer boundaries and domain isolation
  ...createLibraryLayerConfigs({
    boundaries,
    order: ['shared', 'features', 'screens'],
    sliceLayers: ['features'],
    outsideLayers,
    domainImports,
  }),

  createOwnImportConfig({
    layer: 'screens',
    patterns: domainImports,
    group: ['@/screens/**'],
    message: 'a screen imports its own files by relative path',
    outsideLayers,
  }),

  // Anything with a lifetime a test must hold still is a port. The
  // pure core computes what is drawn from its props alone, so it may not ask
  // the machine what time it is, what the window measures, or what comes next.
  createPortConfig({
    files: ['src/**/*.ts'],
    ignores: ['src/**/*.test.ts', 'src/**/*.stories.ts', 'src/**/fixtures/**'],
    ...clockFunctions,
  }),

  createPureSegmentConfig(clockFunctions),

  ...createStoryConfig(storybook),
)
