// Correctness and boundaries for the flashcards window, on the rules
// `@numen/ui` already states. Nothing here is about how the code looks: the
// formatting is done by hand and a rule with an opinion about it would start
// rewriting files nobody asked it to.
//
// The library's ban on importing the domain is the one rule that stops at this
// package: a window is the thing that knows what a vault is. Which way the
// modules may point is dependency-cruiser's answer, given by `npm run check`
// in modules/tools/depgraph.

import js from '@eslint/js'
import globals from 'globals'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'
import prettier from 'eslint-config-prettier'

/** The globals a pure core is not allowed to reach for, and the port for each. */
const lifetimes = [
  { name: 'requestAnimationFrame', port: 'the schedule a Clock carries' },
  { name: 'cancelAnimationFrame', port: 'the cancel a Clock carries' },
  { name: 'matchMedia', port: 'a prop, or CSS where the browser already knows' },
]

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
          allowDefaultProject: [
            'eslint.config.js',
            'vitest.config.ts',
            '.storybook/main.ts',
            '.storybook/preview.ts',
          ],
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

      // A screen is named after the one thing it draws, and is addressed in
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
      'vue/max-lines-per-block': ['error', { template: 100, script: 300, skipBlankLines: true }],
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

  // Two hundred and fifty lines in a handwritten file, and a function whose
  // branches a reader cannot hold at once is two functions. A test and a story
  // are shaped by what they are describing and are not held to either.
  {
    files: ['src/**/*.{ts,vue}'],
    ignores: ['src/**/*.test.ts', 'src/**/*.stories.ts'],
    rules: {
      'max-lines': ['error', { max: 250, skipBlankLines: false, skipComments: false }],
      complexity: ['error', 10],
      'max-depth': ['error', 3],
    },
  },

  // Anything with a lifetime a test must hold still is a port. The pure core
  // computes what is drawn from what it is given, so it may not ask the machine
  // what time it is, what the window measures, or what comes next.
  //
  // The humble view is not held to this — it is the half that talks to the
  // browser — and neither are the stories and the tests, which stand outside a
  // screen on purpose.
  {
    files: ['src/**/*.ts'],
    ignores: [
      'src/**/*.test.ts',
      'src/**/*.stories.ts',
      // The port's own default, which is where the browser is allowed in.
      'src/pages/session/model/session.ts',
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
  {
    files: ['**/*.test.ts', '**/*.stories.ts', '.storybook/**'],
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
