// Correctness and boundaries, as in `libs/ui/components`. The shell imports components
// through `@numen/ui` and no other first-party package.

import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import storybook from 'eslint-plugin-storybook'
import {
  createBaseConfig,
  createPortConfig,
  createStoryConfig,
} from '../../../tools/lint-layers/base-eslint.mjs'
import {
  createDomainImports,
  createLibraryLayerConfigs,
} from '../../../tools/lint-layers/layers.mjs'

/** The files that stand outside the layers. */
const outsideLayers = [
  'src/**/*.stories.ts',
  'src/**/fixtures/**',
  'src/vite.ts',
  'src/vite.test.ts',
]

const domainImports = createDomainImports({
  message:
    'the shell knows the components and nothing of the domain or the wire — take props in a drawing vocabulary and emit opaque identifiers',
  direction: 'the dependency runs apps → libs/ui/workbench → libs/ui/components, never back',
})

export default tseslint.config(
  ...createBaseConfig(import.meta.dirname, { defaultProject: ['eslint.config.js'] }),

  ...createLibraryLayerConfigs({
    boundaries,
    order: ['shared', 'features', 'screens'],
    sliceLayers: ['features'],
    outsideLayers,
    domainImports,
  }),

  // Anything with a lifetime a test must hold still is a port.
  createPortConfig({
    files: ['src/**/*.ts'],
    ignores: ['src/**/*.test.ts', 'src/**/*.stories.ts', 'src/**/fixtures/**'],
  }),

  ...createStoryConfig(storybook),
)
