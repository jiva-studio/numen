// Correctness and boundaries. Nothing here is about how the code looks: the
// formatting is done by hand and a rule with an opinion about it would start
// rewriting files nobody asked it to.
//
// Three of the four groups below are the module's boundary made executable. A
// component knows nothing about the domain, anything with a lifetime is a port,
// and a component finding its own children takes a template ref.

import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import { createBaseConfig, lifetimes } from '../../../tools/lint-layers/base-eslint.mjs'
import { createLayerElements, createLayerPolicies } from '../../../tools/lint-layers/layers.mjs'

const layerOrder = ['shared', 'features', 'screens']
const sliceLayers = ['features']

const layerElements = createLayerElements({ order: layerOrder, sliceLayers })
const layerPolicies = createLayerPolicies({ order: layerOrder, sliceLayers, indexOnly: true })

/** The tests that read the wire's fixtures from `libs/protocol/testdata` by relative path. */
const testdataReaders = []

/** The files that stand outside the layers. */
const outsideLayers = [
  'src/**/*.stories.ts',
  'src/**/fixtures/**',
  'src/vite.ts',
  'src/vite.test.ts',
  ...testdataReaders,
]

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
      'a component knows nothing about the domain or the wire — take props in a drawing vocabulary and emit opaque identifiers',
  },
  {
    group: ['**/apps/**', '**/libs/**'],
    message: 'the dependency runs apps → libs/ui/core, never back',
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

  // UI layer boundaries and domain isolation
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

  {
    files: ['src/screens/**/*.{ts,vue}'],
    ignores: ['src/**/*.test.ts', 'src/**/*.spec.ts', ...outsideLayers],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            ...domainImports,
            {
              group: ['@/screens/**'],
              message: 'a screen imports its own files by relative path',
            },
          ],
        },
      ],
    },
  },

  // Anything with a lifetime a test must hold still is a port. The
  // pure core computes what is drawn from its props alone, so it may not ask
  // the machine what time it is, what the window measures, or what comes next.
  {
    files: ['src/**/*.ts'],
    ignores: [
      'src/**/*.test.ts',
      'src/**/*.stories.ts',
      'src/**/fixtures/**',
      // The port itself.
      'src/shared/lib/time/clock.ts',
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
    files: ['vitest.setup.ts'],
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
