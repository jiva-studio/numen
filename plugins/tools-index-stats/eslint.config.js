import js from '@eslint/js'
import tseslint from 'typescript-eslint'
import pluginVue from 'eslint-plugin-vue'
import prettier from 'eslint-config-prettier'

const contract = ['plugin-sdk', 'ui']

export default [
  {
    ignores: ['dist/**', 'coverage/**', 'node_modules/**', 'stories/.storybook/**'],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...pluginVue.configs['flat/recommended'],
  {
    files: ['**/*.{ts,vue}'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
        projectService: { allowDefaultProject: ['vite.config.ts', 'vitest.config.ts'] },
        tsconfigRootDir: import.meta.dirname,
        extraFileExtensions: ['.vue'],
      },
    },
    rules: {
      // The strict set: every promise is awaited, returned or given an error channel; no
      // statement discards a value with `void`; no block or function body is empty.
      'no-void': 'error',
      'no-empty': ['error', { allowEmptyCatch: false }],
      'no-empty-function': 'off',
      '@typescript-eslint/no-empty-function': 'error',
      '@typescript-eslint/no-floating-promises': ['error', { ignoreVoid: false }],
      '@typescript-eslint/no-misused-promises': 'error',
      '@typescript-eslint/await-thenable': 'error',
      '@typescript-eslint/switch-exhaustiveness-check': ['error', { considerDefaultExhaustiveForUnions: true }],
      complexity: ['error', 12],
      'max-depth': ['error', 3],
      'max-params': ['error', 4],
      'no-console': 'error',
      'no-undef': 'off',
      'vue/multi-word-component-names': 'off',
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['@/**', '@/pages/**', '@/shared/**', '@/entities/**', '@/features/**', '@/widgets/**', '@/app/**'],
              message: 'plugins must not import host internals (@/*); use @numen/plugin-sdk or @numen/ui instead',
            },
            {
              group: ['@plugins', '@plugins/**'],
              message: 'a plugin names no other plugin; share through @numen/plugin-sdk',
            },
            {
              group: ['@numen/*', ...contract.flatMap((name) => [`!@numen/${name}`, `!@numen/${name}/**`])],
              message: 'a plugin names only @numen/plugin-sdk, @numen/ui and its own directory',
            },
          ],
        },
      ],
    },
  },
  {
    files: ['**/*.{ts,vue}'],
    ignores: ['**/*.test.ts', '**/*.stories.ts', 'stories/.storybook/**', '**/testing/**'],
    rules: {
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
    files: ['**/lib/**/*.ts'],
    ignores: ['**/*.test.ts', '**/*.stories.ts'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            { name: 'vue', message: 'lib/ is pure TypeScript and must not depend on Vue reactivity' },
            { name: 'pinia', message: 'lib/ is pure TypeScript and must not depend on Pinia' },
          ],
          patterns: [
            { group: ['@vue/*'], message: 'lib/ is pure TypeScript and must not depend on @vue/*' },
            {
              group: ['../ui/*', '../ui/**', './ui/*', '../model/*', '../model/**', './model/*', '../api/*', '../api/**', './api/*'],
              message: 'lib/ is the lowest segment and cannot import from api/, model/, or ui/',
            },
          ],
        },
      ],
      'no-restricted-globals': [
        'error',
        { name: 'window', message: 'lib/ is pure TypeScript and must not access DOM window' },
        { name: 'document', message: 'lib/ is pure TypeScript and must not access DOM document' },
        { name: 'navigator', message: 'lib/ is pure TypeScript and must not access browser navigator' },
        { name: 'localStorage', message: 'lib/ is pure TypeScript and must not access localStorage' },
        { name: 'sessionStorage', message: 'lib/ is pure TypeScript and must not access sessionStorage' },
      ],
    },
  },
  {
    files: ['**/model/**/*.ts'],
    ignores: ['**/*.test.ts', '**/*.stories.ts'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [{ group: ['../ui/*', '../ui/**', './ui/*'], message: 'model/ must not import upward from ui/' }],
        },
      ],
      'no-restricted-globals': [
        'error',
        { name: 'window', message: 'model/ must not access window directly — use a platform/DOM port' },
        { name: 'document', message: 'model/ must not access document directly — use a platform/DOM port' },
        { name: 'navigator', message: 'model/ must not access navigator directly — use a platform/DOM port' },
      ],
    },
  },
  {
    files: ['**/*.vue'],
    ignores: ['**/*.stories.ts', 'stories/.storybook/**'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['**/api/wire*', './api/wire*', '../api/wire*', '@connectrpc/*', '@bufbuild/*'],
              message: '.vue components are humble views and must talk to model/composables, never directly to wire/RPC',
            },
          ],
        },
      ],
      'no-restricted-globals': [
        'error',
        { name: 'window', message: '.vue components must not access window directly — delegate to a composable or port' },
        { name: 'document', message: '.vue components must not access document directly — use template refs or a composable' },
        { name: 'navigator', message: '.vue components must not access navigator directly — delegate to a composable or port' },
      ],
    },
  },
  {
    files: ['**/*.test.ts', '**/*.stories.ts', 'stories/.storybook/**'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      'no-console': 'off',
      'no-restricted-globals': 'off',
      'require-yield': 'off',
      'max-lines': 'off',
      'max-depth': 'off',
      complexity: 'off',
      'max-params': 'off',
    },
  },
  // Configuration scripts are outside the TypeScript project.
  {
    files: ['**/*.js'],
    ...tseslint.configs.disableTypeChecked,
  },
  prettier,
]
