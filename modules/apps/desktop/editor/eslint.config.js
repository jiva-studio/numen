// Correctness and boundaries for the window, on the rules `@numen/ui` already
// states. Nothing here is about how the code looks: the formatting is done by
// hand and a rule with an opinion about it would start rewriting files nobody
// asked it to.
//
// The library's ban on importing the domain is the one rule that stops at this
// package: a window is the thing that knows what a vault is.

import js from '@eslint/js'
import globals from 'globals'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import prettier from 'eslint-config-prettier'
import { createLayerElements, createLayerPolicies } from '../../../tools/lint/layers.mjs'

/** The globals a pure core is not allowed to reach for, and the port for each. */
const lifetimes = [
  { name: 'requestAnimationFrame', port: 'the schedule a Clock carries' },
  { name: 'cancelAnimationFrame', port: 'the cancel a Clock carries' },
  { name: 'matchMedia', port: 'a prop, or CSS where the browser already knows' },
]

const layerOrder = ['shared', 'entities', 'features', 'widgets', 'pages', 'app']
const sliceLayers = ['entities', 'features', 'widgets', 'pages']

const layerElements = createLayerElements({
  order: layerOrder,
  sliceLayers,
  extra: [{ type: 'testing', pattern: 'src/testing' }],
})

const layerPolicies = createLayerPolicies({
  order: layerOrder,
  sliceLayers,
  extra: [
    { allow: { to: { element: { type: 'testing' } } } },
    { from: { element: { type: 'testing' } }, allow: { to: { element: { type: layerOrder } } } },
  ],
})

export default tseslint.config(
  {
    ignores: ['dist/**', 'storybook-static/**', 'coverage/**', 'node_modules/**'],
  },

  js.configs.recommended,
  tseslint.configs.recommended,
  pluginVue.configs['flat/essential'],

  {
    languageOptions: {
      globals: globals.browser,
      parserOptions: {
        // The files the build's tsconfig does not name.
        projectService: {
          allowDefaultProject: ['eslint.config.js', 'vitest.config.ts', '.storybook/*.ts'],
        },
        tsconfigRootDir: import.meta.dirname,
        extraFileExtensions: ['.vue'],
      },
    },
    rules: {
      // Nothing here logs. What a person can act on is said in the window, on
      // the task list, or on the stream whoever started the process reads. A
      // message that exists only on the console is one the window decided not
      // to say.
      'no-console': 'error',

      // A promise nobody waits for finishes somewhere nobody is looking.
      '@typescript-eslint/no-floating-promises': 'error',
      '@typescript-eslint/no-misused-promises': 'error',
      '@typescript-eslint/await-thenable': 'error',

      // A parameter named and unused is a signature to honour; a leading
      // underscore says it is there on purpose.
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
      ],

      // A tab is named after the one thing it draws, and is addressed in
      // PascalCase, so it meets no HTML element. This window exports no barrel,
      // so the collision the rule guards against elsewhere cannot arise here.
      'vue/multi-word-component-names': 'off',
    },
  },

  // A single-file component is read by the Vue parser, which hands the script
  // on; without this the type-aware rules have no types for a .vue.
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
        extraFileExtensions: ['.vue'],
      },
    },
  },

  // A component's props are a contract with another module.
  {
    files: ['**/*.vue'],
    rules: {
      'vue/define-props-declaration': ['error', 'type-based'],
      'vue/require-explicit-emits': 'error',
      'vue/no-undef-components': 'error',
      'vue/attribute-hyphenation': ['error', 'always'],
      'vue/custom-event-name-casing': ['error', 'kebab-case'],
    },
  },

  // A component draws one thing, and its size is where that is checked. A
  // template past a hundred lines or four elements deep holds a second
  // component nobody has named; a script past three hundred holds work that
  // belongs in a `.ts` beside it, where a test reaches it without mounting
  // anything.
  //
  // The order of the blocks and of the macros is the one every component here
  // is already written in.
  {
    files: ['**/*.vue'],
    rules: {
      'vue/max-lines-per-block': [
        'error',
        { template: 120, script: 300, style: 80, skipBlankLines: true },
      ],
      'vue/max-template-depth': ['error', { maxDepth: 4 }],
      'vue/block-order': ['error', { order: ['script', 'template', 'style'] }],
      'vue/define-macros-order': [
        'error',
        { order: ['defineProps', 'defineModel', 'defineEmits', 'defineSlots'] },
      ],

      // A template says what is drawn. Every decision behind it is made in a
      // computed or in a named handler, which a test can call.
      'vue/no-restricted-syntax': [
        'error',
        {
          selector: 'VElement ConditionalExpression ConditionalExpression',
          message: 'a choice between three things is a computed',
        },
        {
          selector: 'VOnExpression LogicalExpression',
          message: 'a handler is a named function, and the guard goes inside it',
        },
        {
          selector:
            'VAttribute[directive=true][key.name.name="bind"][key.argument.name="style"] ObjectExpression',
          message:
            'an inline style object is a computed in <script>, not an object literal in the template',
        },
      ],

      // A computed is a projection of what the component was given.
      'no-restricted-syntax': [
        'error',
        {
          selector:
            'CallExpression[callee.name="computed"] :matches(ForStatement, ForOfStatement, ForInStatement, WhileStatement)',
          message: 'a loop over the domain is a pure function in a .ts, with a test of its own',
        },
      ],
    },
  },

  // Limits on complexity and file length for handwritten sources. A test, a spec
  // and a story are shaped by what they are describing and are not held to either.
  {
    files: ['src/**/*.{ts,vue}'],
    ignores: ['src/**/*.test.ts', 'src/**/*.spec.ts', 'src/**/*.stories.ts', 'src/testing/**'],
    rules: {
      'max-lines': ['error', { max: 350, skipBlankLines: true, skipComments: true }],
      'max-lines-per-function': ['error', { max: 60, skipBlankLines: true, skipComments: true }],
      complexity: ['error', 10],
      'max-depth': ['error', 3],
      'max-params': ['error', 3],
      'no-nested-ternary': 'error',
    },
  },

  // Package boundaries: no reaching out of the module via relative paths, and
  // no plugin named. The window meets its plugins through the loader, which
  // finds them by their manifests; the tests and stories that draw the composed
  // window are the files that name one.
  {
    files: ['src/**/*.{ts,vue}'],
    ignores: ['src/**/*.test.ts', 'src/**/*.spec.ts', 'src/**/*.stories.ts', 'src/testing/**'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['**/libs/**', '**/apps/**'],
              message:
                'reach libraries and other applications by their package names (@numen/*), never by relative paths out of the module',
            },
            {
              group: ['@plugins/**', '**/plugins/**'],
              message:
                'the window names no plugin: a plugin registers what it offers through @numen/plugin-sdk, and the loader finds it',
            },
          ],
        },
      ],
    },
  },

  // The layers of modules/apps/desktop/AGENTS.md: shared, entities, features,
  // widgets, pages, app, each reaching only what stands below it. A slice
  // reaches a sibling of its own layer only through the sibling's `@x`. Every
  // import is resolved to the file it names, so `../../pages/x` is held to the
  // same rule as `@/pages/x`.
  //
  // `src/testing` holds the doubles of every layer, reaches every layer, and is
  // reached by the tests of every layer. Tests and stories sit in their layer
  // and are held to the rule.
  //
  // A file under `src` that no element claims fails.
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
  //
  // The humble view is not held to this — it is the half that talks to the
  // browser — and neither are the stories, the tests and the fixtures, which
  // stand outside a tab on purpose.
  {
    files: ['src/**/*.ts'],
    ignores: [
      'src/**/*.test.ts',
      'src/**/*.stories.ts',
      'src/testing/**',
      // The ports' own defaults, which is where the browser is allowed in.
      'src/entities/note/model/notes.ts',
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

  // A component finding its own children takes a template ref. A story stands
  // outside the component and is allowed to reach in.
  {
    files: ['**/*.vue'],
    rules: {
      'no-restricted-properties': [
        'error',
        {
          object: 'document',
          property: 'querySelector',
          message: 'a component finding its own children takes a template ref',
        },
        {
          object: 'document',
          property: 'querySelectorAll',
          message: 'a component finding its own children takes a template ref',
        },
        {
          object: 'document',
          property: 'getElementById',
          message: 'a component finding its own children takes a template ref',
        },
      ],
    },
  },

  // A test and a story are code that ships to nobody, and both reach for the
  // browser on purpose.
  //
  // The promise rules are off: a play function is written as a run of
  // interactions and does not await what it starts. In the tabs the same rules
  // are on and the tree is clean.
  {
    files: ['**/*.test.ts', '**/*.stories.ts', 'src/testing/**', '.storybook/**'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',
      '@typescript-eslint/no-floating-promises': 'off',
      '@typescript-eslint/no-misused-promises': 'off',
      '@typescript-eslint/await-thenable': 'off',
      // A story standing alone may print what it measured.
      'no-console': 'off',
      // A test double for a stream that fails before it yields anything.
      'require-yield': 'off',
    },
  },

  prettier,
)
