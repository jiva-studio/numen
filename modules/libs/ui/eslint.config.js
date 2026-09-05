// Correctness and boundaries. Nothing here is about how the code looks: the
// formatting is done by hand and a rule with an opinion about it would start
// rewriting files nobody asked it to.
//
// Three of the four groups below are the module's boundary made executable. A
// component knows nothing about the domain, anything with a lifetime is a port,
// and a component finding its own children takes a template ref.

import js from '@eslint/js'
import globals from 'globals'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'

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
        // This file is the one the build's tsconfig does not name.
        projectService: { allowDefaultProject: ['eslint.config.js'] },
        tsconfigRootDir: import.meta.dirname,
        extraFileExtensions: ['.vue'],
      },
    },
    rules: {
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

      // Button, Card and Menu are what the field calls these, and a single-file
      // component is addressed in PascalCase, so it meets no HTML element. What
      // a name here must not do is take a word the barrel already exports as a
      // type: that is the collision, and no rule refuses it.
      'vue/multi-word-component-names': 'off',

      // The one place it fires is the regular expression that strips control
      // characters out of a URL, where they are the subject.
      'no-control-regex': 'off',
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
    },
  },

  // Nothing in this module imports anything that knows what a vault
  // is — not in the components, not in the stories, not in the fixtures. A
  // fixture taken from the domain is how the dependency comes back in through
  // the door marked "tests", so the rule covers every file the module holds.
  {
    files: ['**/*.{ts,vue}'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['@numen/protocol', '@numen/protocol/*', '@numen/editor', '@numen/wire'],
              message:
                'a component knows nothing about the domain or the wire — take props in a drawing vocabulary and emit opaque identifiers',
            },
            {
              group: ['**/apps/**'],
              message: 'the dependency runs apps → libs/ui, never back',
            },
          ],
        },
      ],
    },
  },

  // Anything with a lifetime a test must hold still is a port. The
  // pure core computes what is drawn from its props alone, so it may not ask
  // the machine what time it is, what the window measures, or what comes next.
  //
  // The humble view is not held to this — it is the half that talks to the
  // browser — and neither are the stories and the fixtures, which stand
  // outside a component on purpose.
  {
    files: ['src/**/*.ts'],
    ignores: [
      'src/**/*.test.ts',
      'src/**/*.stories.ts',
      'src/**/fixtures/**',
      // The port itself, and the one function that turns an element into the
      // rectangle the pure core reasons about.
      'src/lib/clock.ts',
      'src/workspace/drop.ts',
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

  // A component finding its own children takes a template ref. A
  // story stands outside the component and is allowed to reach in.
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
  // The promise rules are off here rather than obeyed: a play function is
  // written as a run of interactions and four hundred of them do not await
  // what they start. That is a backlog, not a licence — it is worth an
  // afternoon and then this block gets shorter. In the components the same
  // rules are on and the tree is clean.
  {
    files: ['**/*.test.ts', '**/*.stories.ts', 'vitest.setup.ts', '.storybook/**'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',
      '@typescript-eslint/no-floating-promises': 'off',
      '@typescript-eslint/no-misused-promises': 'off',
      '@typescript-eslint/await-thenable': 'off',
      // A test double for a stream that fails before it yields anything.
      'require-yield': 'off',
    },
  },
)
