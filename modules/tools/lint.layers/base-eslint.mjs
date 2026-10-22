import js from '@eslint/js'
import globals from 'globals'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'
import prettier from 'eslint-config-prettier'

/** The globals a pure core is not allowed to reach for, and the port for each. */
export const lifetimes = [
  { name: 'requestAnimationFrame', port: 'the schedule a Clock carries' },
  { name: 'cancelAnimationFrame', port: 'the cancel a Clock carries' },
  { name: 'matchMedia', port: 'a prop, or CSS where the browser already knows' },
]

/**
 * Custom rule: Enforce extracting large props interfaces into a separate *.types.ts file.
 */
const numenPlugin = {
  rules: {
    'max-inline-props': {
      meta: {
        type: 'problem',
        docs: {
          description:
            'Enforce extracting large props interfaces (> 6 fields) into a separate types file',
        },
        schema: [
          {
            type: 'object',
            properties: {
              max: {
                type: 'integer',
                minimum: 1,
              },
            },
            additionalProperties: false,
          },
        ],
        messages: {
          tooManyProps:
            'defineProps has {{count}} inline fields (max: {{max}}). Extract props into a separate *.types.ts file.',
        },
      },
      create(context) {
        const max = context.options[0]?.max ?? 6
        return {
          'CallExpression[callee.name="defineProps"] > TSTypeParameterInstantiation > TSTypeLiteral'(
            node,
          ) {
            const count = node.members?.length || 0
            if (count > max) {
              context.report({
                node,
                messageId: 'tooManyProps',
                data: { count, max },
              })
            }
          },
        }
      },
    },
  },
}

/**
 * Returns the unified base ESLint flat configuration array for Vue 3 + TypeScript projects.
 *
 * @param {string} dirname The import.meta.dirname of the package
 * @param {object} options Optional overrides
 * @returns {Array}
 */
export function createBaseConfig(dirname, options = {}) {
  const allowDefault = [
    'eslint.config.js',
    'vitest.config.ts',
    '.storybook/*.ts',
    ...(options.allowDefaultProject || []),
  ]

  return [
    {
      ignores: [
        'dist/**',
        'storybook-static/**',
        'coverage/**',
        'node_modules/**',
        ...(options.extraIgnores || []),
      ],
    },

    js.configs.recommended,
    ...tseslint.configs.recommended,
    ...pluginVue.configs['flat/essential'],

    {
      plugins: {
        numen: numenPlugin,
      },
      languageOptions: {
        globals: globals.browser,
        parserOptions: {
          projectService: {
            allowDefaultProject: allowDefault,
          },
          tsconfigRootDir: dirname,
          extraFileExtensions: ['.vue'],
        },
      },
      rules: {
        'no-console': 'error',
        '@typescript-eslint/no-floating-promises': 'error',
        '@typescript-eslint/no-misused-promises': 'error',
        '@typescript-eslint/await-thenable': 'error',
        '@typescript-eslint/no-unused-vars': [
          'error',
          { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
        ],
        'vue/multi-word-component-names': 'off',
      },
    },

    {
      files: ['**/*.vue'],
      languageOptions: {
        parserOptions: {
          parser: tseslint.parser,
          projectService: true,
          tsconfigRootDir: dirname,
          extraFileExtensions: ['.vue'],
        },
      },
      rules: {
        'numen/max-inline-props': ['error', { max: 6 }],
        'vue/max-lines-per-block': [
          'error',
          {
            script: 120,
            template: 140,
            skipBlankLines: true,
          },
        ],
        'vue/define-props-declaration': ['error', 'type-based'],
        'vue/require-explicit-emits': 'error',
        'vue/no-undef-components': 'error',
        'vue/attribute-hyphenation': ['error', 'always'],
        'vue/custom-event-name-casing': ['error', 'kebab-case'],
        'vue/no-unused-refs': 'error',
        'vue/component-api-style': ['error', ['script-setup', 'composition']],
        'vue/component-name-in-template-casing': ['error', 'PascalCase'],
        'vue/padding-line-between-blocks': ['error', 'always'],
        'vue/max-template-depth': ['error', { maxDepth: 4 }],
        'vue/block-order': ['error', { order: ['script', 'template', 'style'] }],
        'vue/define-macros-order': [
          'error',
          {
            order: ['defineOptions', 'defineProps', 'defineModel', 'defineEmits', 'defineSlots'],
          },
        ],
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

    {
      files: ['src/**/*.{ts,vue}'],
      ignores: ['src/**/clock.ts', 'src/**/drop.ts', ...(options.portIgnores || [])],
      rules: {
        'no-restricted-globals': [
          'error',
          ...lifetimes.map((l) => ({
            name: l.name,
            message: `reach through ${l.port}`,
          })),
        ],
      },
    },

    {
      files: ['src/**/safe.ts', 'src/**/safe.stories.ts'],
      rules: {
        'no-control-regex': 'off',
      },
    },

    {
      files: ['**/*.test.ts', '**/*.stories.ts', '.storybook/**'],
      rules: {
        '@typescript-eslint/no-explicit-any': 'off',
        '@typescript-eslint/no-non-null-assertion': 'off',
        '@typescript-eslint/no-floating-promises': 'off',
        '@typescript-eslint/no-misused-promises': 'off',
        '@typescript-eslint/await-thenable': 'off',
        'no-console': 'off',
        'no-restricted-globals': 'off',
        'require-yield': 'off',
        'vue/max-template-depth': 'off',
        'vue/max-lines-per-block': 'off',
      },
    },

    prettier,
  ]
}
