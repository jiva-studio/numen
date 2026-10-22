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
const SECTION_ORDER = {
  props: 1,
  'props & emits': 1,
  'props and emits': 1,
  events: 2,
  emits: 2,
  state: 3,
  hooks: 4,
  lifecycle: 4,
  handlers: 5,
  helpers: 6,
}
const bannerRegex = /^-{3,}\s*([A-Za-z &]+?)\s*-{3,}$/

/**
 * Custom rules for Numen project conventions:
 * 1. max-inline-props: Enforce extracting large props interfaces (> 6 fields) into *.types.ts.
 * 2. vue-script-sections: Enforce section banners and canonical order (AGENTS.md rule 4).
 * 3. max-script-lines: Enforce max lines of executable code in <script> (ignoring comments/banners).
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

    'vue-script-sections': {
      meta: {
        type: 'layout',
        docs: {
          description: 'Enforce section structure and canonical order in Vue SFC script block',
        },
        messages: {
          missingSections:
            'Script has {{count}} lines of code but no section banners. Group script into sections: Props -> Events -> State -> Hooks -> Handlers -> Helpers.',
          unknownSection:
            'Unknown section banner "{{name}}". Canonical sections: Props, Events, Props & Emits, State, Hooks, Handlers, Helpers.',
          outOfOrder:
            'Section "{{name}}" should not appear after "{{prevName}}". Canonical order: Props -> Events -> State -> Hooks -> Handlers -> Helpers.',
        },
        schema: [
          {
            type: 'object',
            properties: {
              minCodeLines: { type: 'integer', minimum: 1 },
            },
            additionalProperties: false,
          },
        ],
      },
      create(context) {
        const minCodeLines = context.options[0]?.minCodeLines ?? 35
        return {
          Program() {
            const docFrag = context.sourceCode.parserServices.getDocumentFragment?.()
            const script = docFrag?.children?.find(
              (c) => c.type === 'VElement' && c.name === 'script',
            )
            if (!script) return

            const scriptStart = script.loc.start.line
            const scriptEnd = script.loc.end.line

            const comments = context.sourceCode
              .getAllComments()
              .filter((c) => c.loc.start.line >= scriptStart && c.loc.end.line <= scriptEnd)

            const scriptText = context.sourceCode.getText(script)
            const stripped = scriptText
              .replace(/<script[\s\S]*?>/, '')
              .replace(/<\/script>/, '')
              .replace(/\/\*[\s\S]*?\*\//g, '')
            const codeLines = stripped.split('\n').filter((l) => {
              const t = l.trim()
              return t.length > 0 && !t.startsWith('//')
            }).length

            const banners = []
            for (const comment of comments) {
              const text = comment.value.trim()
              const match = text.match(bannerRegex)
              if (match) {
                banners.push({
                  node: comment,
                  name: match[1].trim(),
                  line: comment.loc.start.line,
                })
              }
            }

            if (codeLines > minCodeLines && banners.length === 0) {
              context.report({
                node: script,
                messageId: 'missingSections',
                data: { count: codeLines },
              })
            }

            let lastRank = 0
            let lastName = ''
            for (const b of banners) {
              const rank = SECTION_ORDER[b.name.toLowerCase()]
              if (!rank) {
                context.report({
                  node: b.node,
                  messageId: 'unknownSection',
                  data: { name: b.name },
                })
                continue
              }
              if (rank < lastRank) {
                context.report({
                  node: b.node,
                  messageId: 'outOfOrder',
                  data: { name: b.name, prevName: lastName },
                })
              }
              lastRank = rank
              lastName = b.name
            }
          },
        }
      },
    },

    'max-script-lines': {
      meta: {
        type: 'problem',
        docs: {
          description:
            'Enforce maximum lines of executable code in <script> (excluding comments and blanks)',
        },
        messages: {
          tooManyLines:
            'Script has {{count}} executable code lines (max: {{max}}). Decompose into composables or sub-components.',
        },
        schema: [
          {
            type: 'object',
            properties: {
              max: { type: 'integer', minimum: 1 },
            },
            additionalProperties: false,
          },
        ],
      },
      create(context) {
        const max = context.options[0]?.max ?? 100
        return {
          Program() {
            const docFrag = context.sourceCode.parserServices.getDocumentFragment?.()
            const script = docFrag?.children?.find(
              (c) => c.type === 'VElement' && c.name === 'script',
            )
            if (!script) return

            const scriptText = context.sourceCode.getText(script)
            const stripped = scriptText
              .replace(/<script[\s\S]*?>/, '')
              .replace(/<\/script>/, '')
              .replace(/\/\*[\s\S]*?\*\//g, '')
            const codeLines = stripped.split('\n').filter((l) => {
              const t = l.trim()
              return t.length > 0 && !t.startsWith('//')
            }).length

            if (codeLines > max) {
              context.report({
                node: script,
                messageId: 'tooManyLines',
                data: { count: codeLines, max },
              })
            }
          },
        }
      },
    },

    'boolean-prop-naming': {
      meta: {
        type: 'suggestion',
        docs: {
          description:
            'Enforce boolean props to use is/has/can prefix according to AGENTS.md rule 6',
        },
        messages: {
          invalidBooleanPrefix:
            'Boolean prop "{{name}}" must use prefix is/has/can (e.g. is{{capitalized}}).',
        },
      },
      create(context) {
        return {
          'CallExpression[callee.name="defineProps"] > TSTypeParameterInstantiation > TSTypeLiteral > TSPropertySignature'(
            node,
          ) {
            const name = node.key?.name || node.key?.value
            if (!name) return

            const typeAnnotation = node.typeAnnotation?.typeAnnotation
            if (!typeAnnotation) return

            let isBoolean = typeAnnotation.type === 'TSBooleanKeyword'
            if (typeAnnotation.type === 'TSUnionType') {
              isBoolean = typeAnnotation.types.some((t) => t.type === 'TSBooleanKeyword')
            }

            if (isBoolean && !/^(is|has|can)[A-Z]/.test(name)) {
              const capitalized = name.charAt(0).toUpperCase() + name.slice(1)
              context.report({
                node: node.key,
                messageId: 'invalidBooleanPrefix',
                data: { name, capitalized },
              })
            }
          },
        }
      },
    },

    'handler-naming': {
      meta: {
        type: 'suggestion',
        docs: {
          description:
            'Enforce event handler naming on<Action> and forbid handle... prefix (AGENTS.md rule 3)',
        },
        messages: {
          invalidHandlerName:
            'Event handler "{{name}}" must start with "on" in PascalCase (e.g. on{{capitalized}}). Patterns like "handle..." are forbidden by AGENTS.md rule 3.',
        },
      },
      create(context) {
        let handlersStart = -1
        let handlersEnd = Infinity

        return {
          Program() {
            const docFrag = context.sourceCode.parserServices.getDocumentFragment?.()
            const script = docFrag?.children?.find(
              (c) => c.type === 'VElement' && c.name === 'script',
            )
            if (!script) return

            const comments = context.sourceCode
              .getAllComments()
              .filter(
                (c) =>
                  c.loc.start.line >= script.loc.start.line &&
                  c.loc.end.line <= script.loc.end.line,
              )

            for (let i = 0; i < comments.length; i++) {
              const c = comments[i]
              const match = c.value.trim().match(bannerRegex)
              if (match && match[1].trim().toLowerCase() === 'handlers') {
                handlersStart = c.loc.end.line
                for (let j = i + 1; j < comments.length; j++) {
                  const nextMatch = comments[j].value.trim().match(bannerRegex)
                  if (nextMatch) {
                    handlersEnd = comments[j].loc.start.line
                    break
                  }
                }
                if (handlersEnd === Infinity) {
                  handlersEnd = script.loc.end.line
                }
                break
              }
            }
          },

          FunctionDeclaration(node) {
            if (!node.id || handlersStart === -1) return
            const line = node.loc.start.line
            if (line > handlersStart && line < handlersEnd) {
              const name = node.id.name
              if (!/^on[A-Z]/.test(name)) {
                const cleaned = name.replace(/^handle/, '')
                const capitalized = cleaned.charAt(0).toUpperCase() + cleaned.slice(1)
                context.report({
                  node: node.id,
                  messageId: 'invalidHandlerName',
                  data: { name, capitalized },
                })
              }
            }
          },

          VariableDeclarator(node) {
            if (!node.id || handlersStart === -1) return
            const isFn =
              node.init?.type === 'ArrowFunctionExpression' ||
              node.init?.type === 'FunctionExpression'
            if (!isFn) return
            const line = node.loc.start.line
            if (line > handlersStart && line < handlersEnd) {
              const name = node.id.name
              if (!/^on[A-Z]/.test(name)) {
                const cleaned = name.replace(/^handle/, '')
                const capitalized = cleaned.charAt(0).toUpperCase() + cleaned.slice(1)
                context.report({
                  node: node.id,
                  messageId: 'invalidHandlerName',
                  data: { name, capitalized },
                })
              }
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

  const useProjectService = options.useProjectService !== false

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
          ...(useProjectService
            ? {
                projectService: {
                  allowDefaultProject: allowDefault,
                },
                tsconfigRootDir: dirname,
              }
            : {}),
          extraFileExtensions: ['.vue'],
        },
      },
      rules: {
        'no-console': 'error',
        ...(useProjectService
          ? {
              '@typescript-eslint/no-floating-promises': 'error',
              '@typescript-eslint/no-misused-promises': 'error',
              '@typescript-eslint/await-thenable': 'error',
            }
          : {}),
        '@typescript-eslint/no-unused-vars': [
          'error',
          { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
        ],
        'vue/multi-word-component-names': 'off',
        'max-depth': ['error', 3],
        complexity: ['error', 12],
        'max-params': ['error', 4],
      },
    },

    {
      files: ['**/*.vue'],
      languageOptions: {
        parserOptions: {
          parser: tseslint.parser,
          ...(useProjectService
            ? {
                projectService: true,
                tsconfigRootDir: dirname,
              }
            : {}),
          extraFileExtensions: ['.vue'],
        },
      },
      rules: {
        'numen/max-inline-props': ['error', { max: 6 }],
        'numen/vue-script-sections': ['error', { minCodeLines: 35 }],
        'numen/max-script-lines': ['error', { max: 100 }],
        'numen/boolean-prop-naming': 'error',
        'numen/handler-naming': 'error',
        'vue/max-lines-per-block': [
          'error',
          {
            template: 140,
            skipBlankLines: true,
          },
        ],
        'vue/define-props-declaration': ['error', 'type-based'],
        'vue/define-emits-declaration': ['error', 'type-based'],
        'vue/require-explicit-emits': 'error',
        'vue/no-undef-components': 'error',
        'vue/attribute-hyphenation': ['error', 'always'],
        'vue/custom-event-name-casing': ['error', 'kebab-case', { ignores: ['update:modelValue'] }],
        'vue/v-on-event-hyphenation': ['error', 'always'],
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
        'vue/no-side-effects-in-computed-properties': 'error',
        'vue/no-async-in-computed-properties': 'error',
        'vue/no-ref-as-operand': 'error',
        'vue/no-useless-v-bind': 'error',
        'vue/no-lone-template': 'error',
        'vue/no-useless-mustaches': 'error',
        'vue/prefer-true-attribute-shorthand': 'error',
        'vue/prefer-use-template-ref': 'error',
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
        'numen/max-script-lines': 'off',
        'numen/vue-script-sections': 'off',
        'numen/boolean-prop-naming': 'off',
        'numen/handler-naming': 'off',
        'max-depth': 'off',
        complexity: 'off',
        'max-params': 'off',
      },
    },

    prettier,
  ]
}
