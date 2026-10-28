// Correctness and boundaries, as in `libs/ui/core`. The shell reaches the
// components through `@numen/ui` and nothing else of ours.

import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import { createBaseConfig, lifetimes } from '../../../tools/lint-layers/base-eslint.mjs'
import { createLayerElements, createLayerPolicies } from '../../../tools/lint-layers/layers.mjs'

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
  ...createBaseConfig(import.meta.dirname),

  {
    languageOptions: {
      parserOptions: {
        projectService: {
          allowDefaultProject: ['eslint.config.js'],
        },
      },
    },
  },

  {
    files: ['src/**/*.{ts,vue}'],
    ignores: outsideLayers,
    plugins: { boundaries },
    settings: {
      'import/resolver': { typescript: { alwaysTryTypes: true } },
      'boundaries/elements': layerElements,
      'boundaries/files': [
        { category: 'public', pattern: 'src/features/*/index.ts' },
        { category: 'entry', pattern: 'src/index.ts' },
      ],
    },
    rules: {
      'boundaries/no-unknown-files': 'error',
      'boundaries/dependencies': ['error', { default: 'disallow', policies: layerPolicies }],
      'no-restricted-imports': ['error', { patterns: domainImports }],
    },
  },

  {
    files: ['src/features/**/*.{ts,vue}'],
    ignores: ['src/**/*.test.ts', 'src/**/*.spec.ts', ...outsideLayers],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            ...domainImports,
            {
              group: ['@/features/**', '@/screens/**'],
              message: 'a slice imports its own files by relative path and no other slice',
            },
          ],
        },
      ],
    },
  },

  // Anything with a lifetime a test must hold still is a port.
  {
    files: ['src/**/*.ts'],
    ignores: [
      'src/**/*.test.ts',
      'src/**/*.stories.ts',
      'src/**/fixtures/**',
      // The one function that turns an element into the rectangle the pure
      // core reasons about.
      'src/features/workspace/lib/drop.ts',
      'src/shared/lib/dom.ts',
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
