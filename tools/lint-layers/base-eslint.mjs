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

    'max-composable-return-properties': {
      meta: {
        type: 'problem',
        docs: {
          description:
            'Enforce maximum number of properties returned from a composable to avoid God-composables',
        },
        messages: {
          tooManyReturnedProperties:
            'Composable "{{name}}" returns {{count}} properties (max: {{max}}). Decompose into smaller focused composables.',
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
        const max = context.options[0]?.max ?? 8
        const filename = context.filename || ''

        // Check if file is a composable file (e.g. use*.ts) or function is use*
        function checkReturnObject(node, fnName) {
          if (!fnName || !/^use[A-Z]/.test(fnName)) return
          let count = 0
          for (const prop of node.properties) {
            if (prop.type === 'Property') {
              count++
            } else if (prop.type === 'SpreadElement') {
              // Spreading another object/state counts as multiple properties (at least 2)
              count += 2
            }
          }
          if (count > max) {
            context.report({
              node,
              messageId: 'tooManyReturnedProperties',
              data: { name: fnName, count, max },
            })
          }
        }

        return {
          'FunctionDeclaration:exit'(node) {
            const fnName = node.id?.name
            if (!fnName || !/^use[A-Z]/.test(fnName)) return
            const body = node.body
            if (!body || body.type !== 'BlockStatement') return
            for (const stmt of body.body) {
              if (
                stmt.type === 'ReturnStatement' &&
                stmt.argument?.type === 'ObjectExpression'
              ) {
                checkReturnObject(stmt.argument, fnName)
              }
            }
          },

          'VariableDeclarator:exit'(node) {
            const fnName = node.id?.name
            if (!fnName || !/^use[A-Z]/.test(fnName)) return
            const init = node.init
            if (!init) return
            if (
              init.type === 'ArrowFunctionExpression' ||
              init.type === 'FunctionExpression'
            ) {
              if (init.body.type === 'ObjectExpression') {
                checkReturnObject(init.body, fnName)
              } else if (init.body.type === 'BlockStatement') {
                for (const stmt of init.body.body) {
                  if (
                    stmt.type === 'ReturnStatement' &&
                    stmt.argument?.type === 'ObjectExpression'
                  ) {
                    checkReturnObject(stmt.argument, fnName)
                  }
                }
              }
            }
          },
        }
      },
    },

    'enforce-vue-reactivity-in-composable': {
      meta: {
        type: 'suggestion',
        docs: {
          description:
            'Enforce that any use* function contains Vue reactivity primitives, or rename to create*',
        },
        messages: {
          missingReactivity:
            'Function "{{name}}" starts with "use" but contains no Vue reactivity primitives (ref, computed, watch, etc.). Rename to create{{suffix}} or pure service.',
        },
      },
      create(context) {
        const REACTIVITY_NAMES = new Set([
          'ref',
          'shallowRef',
          'computed',
          'watch',
          'watchEffect',
          'watchSyncEffect',
          'watchPostEffect',
          'reactive',
          'shallowReactive',
          'toRef',
          'toRefs',
          'toValue',
          'onMounted',
          'onUnmounted',
          'onBeforeMount',
          'onBeforeUnmount',
          'onUpdated',
          'onBeforeUpdate',
          'onScopeDispose',
          'provide',
          'inject',
          'useTemplateRef',
        ])

        function checkFunction(node, name) {
          if (!name || !/^use[A-Z]/.test(name)) return
          let hasReactivity = false

          // Traverse descendants to find reactivity calls or other use* calls
          function inspect(n) {
            if (!n || hasReactivity) return
            if (n.type === 'CallExpression') {
              let calleeName = ''
              if (n.callee.type === 'Identifier') {
                calleeName = n.callee.name
              } else if (
                n.callee.type === 'MemberExpression' &&
                n.callee.property.type === 'Identifier'
              ) {
                calleeName = n.callee.property.name
              }
              if (
                REACTIVITY_NAMES.has(calleeName) ||
                /^use[A-Z]/.test(calleeName)
              ) {
                hasReactivity = true
                return
              }
            }
            for (const key of Object.keys(n)) {
              if (key === 'parent') continue
              const child = n[key]
              if (Array.isArray(child)) {
                for (const c of child) {
                  if (c && typeof c === 'object' && c.type) inspect(c)
                }
              } else if (child && typeof child === 'object' && child.type) {
                inspect(child)
              }
            }
          }

          inspect(node.body)

          if (!hasReactivity) {
            const suffix = name.slice(3)
            context.report({
              node: node.id || node,
              messageId: 'missingReactivity',
              data: { name, suffix },
            })
          }
        }

        return {
          FunctionDeclaration(node) {
            if (node.id) checkFunction(node, node.id.name)
          },
          VariableDeclarator(node) {
            if (
              node.id?.type === 'Identifier' &&
              (node.init?.type === 'ArrowFunctionExpression' ||
                node.init?.type === 'FunctionExpression')
            ) {
              checkFunction(node.init, node.id.name)
            }
          },
        }
      },
    },

    'no-double-type-assertion': {
      meta: {
        type: 'problem',
        docs: {
          description:
            'Forbid fake double type assertions (e.g. as unknown as T, as any as T) to prevent AI type-safety bypasses',
        },
        messages: {
          forbiddenDoubleAssertion:
            'Double type assertion ("as unknown as T" / "as any as T") is forbidden. Fix the underlying type signature instead of forcing a cast.',
        },
      },
      create(context) {
        return {
          TSAsExpression(node) {
            if (node.expression?.type === 'TSAsExpression') {
              const innerType = node.expression.typeAnnotation?.type
              if (
                innerType === 'TSUnknownKeyword' ||
                innerType === 'TSAnyKeyword'
              ) {
                context.report({
                  node,
                  messageId: 'forbiddenDoubleAssertion',
                })
              }
            }
          },
        }
      },
    },

    'thin-computed': {
      meta: {
        type: 'problem',
        docs: {
          description:
            'Enforce that computed properties in Vue components remain thin projections without loops, sorting, or multi-step business logic',
        },
        messages: {
          tooManyStatements:
            'computed() has {{count}} statements (max: {{max}}). Extract data transformations and business logic into a pure module.',
          forbiddenBranch:
            'Branching ({{kind}}) inside computed() in a .vue file is forbidden. Move conditional business logic into a composable or pure module.',
          forbiddenLoop:
            'Loops inside computed() in a .vue file are forbidden. Move iteration and data transformation into a pure module.',
          forbiddenTransform:
            'Array .{{method}}() inside computed() in a .vue file is forbidden. Move sorting, filtering, and reduction into a pure module.',
        },
        schema: [
          {
            type: 'object',
            properties: {
              maxStatements: { type: 'integer', minimum: 1 },
            },
            additionalProperties: false,
          },
        ],
      },
      create(context) {
        const maxStatements = context.options[0]?.maxStatements ?? 3
        const LOOP_TYPES = new Set([
          'ForStatement',
          'ForInStatement',
          'ForOfStatement',
          'WhileStatement',
          'DoWhileStatement',
        ])
        const FORBIDDEN_ARRAY_METHODS = new Set(['sort', 'reduce', 'some', 'every'])

        function inspectBody(node, state) {
          if (!node) return
          if (node.type === 'SwitchStatement') {
            context.report({
              node,
              messageId: 'forbiddenBranch',
              data: { kind: 'switch' },
            })
            return
          }
          if (node.type === 'IfStatement') {
            state.ifCount += 1
            if (state.ifCount > 1) {
              context.report({
                node,
                messageId: 'forbiddenBranch',
                data: { kind: 'multiple if statements' },
              })
            }
          }
          if (LOOP_TYPES.has(node.type)) {
            context.report({
              node,
              messageId: 'forbiddenLoop',
            })
            return
          }
          if (
            node.type === 'CallExpression' &&
            node.callee?.type === 'MemberExpression' &&
            node.callee.property?.type === 'Identifier' &&
            FORBIDDEN_ARRAY_METHODS.has(node.callee.property.name)
          ) {
            context.report({
              node: node.callee.property,
              messageId: 'forbiddenTransform',
              data: { method: node.callee.property.name },
            })
          }
          for (const key of Object.keys(node)) {
            if (key === 'parent') continue
            const child = node[key]
            if (Array.isArray(child)) {
              for (const item of child) {
                if (item && typeof item === 'object' && item.type) inspectBody(item, state)
              }
            } else if (child && typeof child === 'object' && child.type) {
              inspectBody(child, state)
            }
          }
        }

        return {
          'CallExpression[callee.name="computed"]'(node) {
            const arg = node.arguments?.[0]
            if (!arg) return
            const fn =
              arg.type === 'ArrowFunctionExpression' || arg.type === 'FunctionExpression'
                ? arg
                : null
            if (!fn) return
            if (fn.body?.type === 'BlockStatement') {
              const count = fn.body.body.length
              if (count > maxStatements) {
                context.report({
                  node,
                  messageId: 'tooManyStatements',
                  data: { count, max: maxStatements },
                })
              }
            }
            inspectBody(fn.body, { ifCount: 0 })
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
        'numen/max-composable-return-properties': ['error', { max: 8 }],
        'numen/enforce-vue-reactivity-in-composable': 'error',
        'numen/no-double-type-assertion': 'error',
        '@typescript-eslint/no-explicit-any': 'error',
        '@typescript-eslint/consistent-type-assertions': [
          'error',
          {
            assertionStyle: 'as',
            objectLiteralTypeAssertions: 'never',
          },
        ],
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
        'numen/thin-computed': ['error', { maxStatements: 3 }],
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
        'vue/no-watch-after-await': 'error',
        'vue/no-lifecycle-after-await': 'error',
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
      files: ['**/*.{ts,vue}'],
      ignores: [
        '**/*.test.ts',
        '**/*.stories.ts',
        '.storybook/**',
        '**/testing/**',
        '**/clock.ts',
        '**/drop.ts',
        '**/dom.ts',
        'src/entities/note/model/notes.ts',
        'src/app/window/useWindowKinds.ts',
        ...(options.portIgnores || []),
      ],
      rules: {
        'no-restricted-globals': [
          'error',
          ...lifetimes.map((l) => ({
            name: l.name,
            message: `reach through ${l.port}`,
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
              { group: ['../ui/*', '../ui/**', './ui/*', '../model/*', '../model/**', './model/*', '../api/*', '../api/**', './api/*'], message: 'lib/ is the lowest segment and cannot import from api/, model/, or ui/' },
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
            patterns: [
              { group: ['../ui/*', '../ui/**', './ui/*'], message: 'model/ must not import upward from ui/' },
            ],
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
      ignores: ['**/*.stories.ts', '.storybook/**'],
      rules: {
        'no-restricted-imports': [
          'error',
          {
            patterns: [
              { group: ['**/api/wire*', './api/wire*', '../api/wire*', '@connectrpc/*', '@bufbuild/*'], message: '.vue components are humble views and must talk to model/composables, never directly to wire/RPC' },
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
      files: ['**/safe.ts', '**/safe.stories.ts'],
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
        'numen/thin-computed': 'off',
        'numen/vue-script-sections': 'off',
        'numen/boolean-prop-naming': 'off',
        'numen/handler-naming': 'off',
        'numen/max-composable-return-properties': 'off',
        'numen/enforce-vue-reactivity-in-composable': 'off',
        'numen/no-double-type-assertion': 'off',
        '@typescript-eslint/consistent-type-assertions': 'off',
        'max-depth': 'off',
        complexity: 'off',
        'max-params': 'off',
      },
    },

    prettier,
  ]
}
