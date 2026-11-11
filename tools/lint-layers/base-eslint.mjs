import js from '@eslint/js'
import globals from 'globals'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'
import prettier from 'eslint-config-prettier'
import numenRules from './numen-rules.mjs'

/** The variable a name refers to from a node, or null. */
function findScopeVariable(context, node, name) {
  for (let scope = context.sourceCode.getScope(node); scope; scope = scope.upper) {
    const variable = scope.set.get(name)
    if (variable) return variable
  }
  return null
}

const noMockOfSubjectSliceRule = {
  meta: {
    type: 'problem',
    docs: {
      description:
        'Forbid vi.mock or vi.spyOn of a module or value in the same slice as the test: a test that replaces its own slice verifies the replacement',
    },
    messages: {
      mocksOwnSlice:
        'vi.mock("{{specifier}}") replaces a module of the slice "{{slice}}" that this test verifies. Use the real module and fake only what lies outside the slice (a transport, a port).',
      spiesOnOwnSlice:
        'vi.spyOn({{target}}) spies on a value from the slice "{{slice}}" that this test verifies. Use the real module and fake only what lies outside the slice (a transport, a port).',
    },
    schema: [],
  },
  create(context) {
    const SLICED_LAYERS = new Set(['entities', 'features', 'pages', 'widgets', 'app', 'shared'])
    const filename = (context.filename ?? '').replaceAll('\\', '/')
    const root = /^(.*\/(?:src|frontend))\//.exec(filename)
    if (!root) return {}
    const rootDir = root[1]
    const slice = getSlice(filename)

    function getSlice(file) {
      const parts = file.slice(rootDir.length + 1).split('/')
      const length = SLICED_LAYERS.has(parts[0]) ? 2 : 1
      return parts.length > length ? parts.slice(0, length).join('/') : '.'
    }

    function resolve(specifier) {
      if (specifier.startsWith('@/')) return `${rootDir}/${specifier.slice(2)}`
      if (!specifier.startsWith('.')) return null
      const parts = `${filename.slice(0, filename.lastIndexOf('/'))}/${specifier}`.split('/')
      const out = []
      for (const part of parts) {
        if (part === '..') out.pop()
        else if (part !== '.') out.push(part)
      }
      return out.join('/')
    }

    function isOwnSliceSpecifier(specifier) {
      if (typeof specifier !== 'string') return false
      const target = resolve(specifier)
      if (!target || !target.startsWith(`${rootDir}/`)) return false
      return getSlice(`${target}/x`) === slice || getSlice(target) === slice
    }

    function getRootIdentifier(node) {
      let current = node
      while (current?.type === 'MemberExpression') current = current.object
      return current?.type === 'Identifier' ? current : null
    }

    function isSpySubjectImport(def) {
      const specifier = def.parent?.source?.value
      if (!isOwnSliceSpecifier(specifier)) return false
      if (def.node?.type === 'ImportNamespaceSpecifier') return true
      const target = resolve(specifier)
      if (!target) return false
      const targetStem = target.slice(target.lastIndexOf('/') + 1).replace(/\.[a-z]+$/i, '')
      const fileBase = filename.slice(filename.lastIndexOf('/') + 1)
      return targetStem === 'index' || fileBase.startsWith(`${targetStem}.`)
    }

    function resolveFunctionBody(identifier) {
      const fnVar = findScopeVariable(context, identifier, identifier.name)
      for (const fnDef of fnVar?.defs ?? []) {
        if (fnDef.type === 'FunctionName') return fnDef.node.body
        if (
          fnDef.type === 'Variable' &&
          (fnDef.node.init?.type === 'ArrowFunctionExpression' ||
            fnDef.node.init?.type === 'FunctionExpression')
        ) {
          return fnDef.node.init.body
        }
      }
      return null
    }

    function findObjectPropertyInit(node, propName) {
      if (!node || typeof node !== 'object') return null
      if (node.type === 'AwaitExpression') return findObjectPropertyInit(node.argument, propName)
      if (node.type === 'ObjectExpression') {
        for (const prop of node.properties) {
          if (prop.type === 'Property' && prop.key?.name === propName) return prop.value
        }
        return null
      }
      if (node.type === 'BlockStatement') {
        for (const stmt of node.body) {
          if (stmt.type === 'ReturnStatement') {
            const found = findObjectPropertyInit(stmt.argument, propName)
            if (found) return found
          }
        }
      }
      return null
    }

    function isExpressionFromOwnSlice(expr, seen) {
      if (!expr || typeof expr !== 'object') return false
      const unwrapped = expr.type === 'AwaitExpression' ? expr.argument : expr
      const rootId =
        unwrapped?.type === 'CallExpression'
          ? getRootIdentifier(unwrapped.callee)
          : getRootIdentifier(unwrapped)
      if (!rootId) return false
      if (isFromOwnSlice(rootId, seen)) return true
      const body = resolveFunctionBody(rootId)
      return Boolean(body && returnsOwnSlice(body, seen))
    }

    function isFromOwnSlice(identifier, seen = new Set()) {
      const variable = findScopeVariable(context, identifier, identifier.name)
      if (!variable || seen.has(variable)) return false
      seen.add(variable)
      for (const def of variable.defs) {
        if (def.type === 'ImportBinding') {
          if (isSpySubjectImport(def)) return true
          continue
        }
        if (def.type !== 'Variable') continue
        if (def.node.id.type === 'Identifier') {
          if (isExpressionFromOwnSlice(def.node.init, seen)) return true
          continue
        }
        if (def.node.id.type === 'ObjectPattern') {
          const prop = def.node.id.properties.find(
            (p) => p.type === 'Property' && p.value?.name === identifier.name,
          )
          const propName = prop?.key?.name
          const unwrapped =
            def.node.init?.type === 'AwaitExpression' ? def.node.init.argument : def.node.init
          const calleeId =
            unwrapped?.type === 'CallExpression' ? getRootIdentifier(unwrapped.callee) : null
          const body = calleeId ? resolveFunctionBody(calleeId) : unwrapped
          const propInit = propName && body ? findObjectPropertyInit(body, propName) : null
          if (propInit && isExpressionFromOwnSlice(propInit, seen)) return true
        }
      }
      return false
    }

    function returnsOwnSlice(node, seen) {
      if (!node || typeof node !== 'object') return false
      if (node.type === 'CallExpression' || node.type === 'AwaitExpression') {
        return isExpressionFromOwnSlice(node, new Set(seen))
      }
      if (node.type === 'BlockStatement') {
        return node.body.some(
          (stmt) => stmt.type === 'ReturnStatement' && returnsOwnSlice(stmt.argument, seen),
        )
      }
      return false
    }

    return {
      'CallExpression[callee.object.name="vi"][callee.property.name=/^(mock|doMock)$/]'(node) {
        const specifier = node.arguments[0]?.value
        if (isOwnSliceSpecifier(specifier)) {
          context.report({ node, messageId: 'mocksOwnSlice', data: { specifier, slice } })
        }
      },
      'CallExpression[callee.object.name="vi"][callee.property.name="spyOn"]'(node) {
        const target = node.arguments[0]
        const rootId = getRootIdentifier(target)
        if (rootId && isFromOwnSlice(rootId)) {
          context.report({
            node,
            messageId: 'spiesOnOwnSlice',
            data: { target: context.sourceCode.getText(target), slice },
          })
        }
      },
    }
  },
}

const numenPlugin = {
  ...numenRules,
  rules: {
    ...numenRules.rules,
    'no-mock-of-subject-slice': noMockOfSubjectSliceRule,
  },
}

/** The globals a pure core is not allowed to reach for, and the port for each. */
export const lifetimes = [
  { name: 'requestAnimationFrame', port: 'the schedule a Clock carries' },
  { name: 'cancelAnimationFrame', port: 'the cancel a Clock carries' },
  { name: 'matchMedia', port: 'a prop, or CSS where the browser already knows' },
]

/**
 * Selectors that ban the wall clock (`Date.now()`, `new Date()`) and the monotonic clock
 * (`performance.now()`). When `clockFunction` names a function declaration, the wall clock is
 * allowed inside that declaration only; `monotonicFunction` does the same for `performance.now()`.
 *
 * @param {string | undefined} clockFunction Name of the one function that reads the wall clock
 * @param {string | undefined} monotonicFunction Name of the one function that reads the monotonic clock
 * @returns {Array<{ selector: string, message: string }>}
 */
export function createClockSelectors(clockFunction, monotonicFunction) {
  const inside = (name) => (name ? `:not(FunctionDeclaration[id.name="${name}"] *)` : '')
  const message = 'the clock is a port — take the now a Clock carries'
  return [
    { selector: `MemberExpression[object.name="Date"][property.name="now"]${inside(clockFunction)}`, message },
    { selector: `NewExpression[callee.name="Date"][arguments.length=0]${inside(clockFunction)}`, message },
    {
      selector: `MemberExpression[object.name="performance"][property.name="now"]${inside(monotonicFunction)}`,
      message,
    },
  ]
}

/**
 * Selectors that ban scheduling with the machine's timers. A pure module takes the schedule it
 * runs on from its caller.
 *
 * @returns {Array<{ selector: string, message: string }>}
 */
export function createTimerSelectors() {
  const message = 'a timer has a lifetime a test must control — take the schedule a Clock carries'
  return [
    { selector: 'CallExpression[callee.name=/^(setTimeout|setInterval)$/]', message },
    { selector: 'CallExpression[callee.property.name=/^(setTimeout|setInterval)$/]', message },
  ]
}

/**
 * The selectors every non-test source file obeys: the clocks, randomness and measuring the window.
 *
 * @param {{ clockFunction?: string, monotonicFunction?: string }} options The sanctioned clock functions
 * @returns {Array<{ selector: string, message: string }>}
 */
export function createDeterminismSelectors({ clockFunction, monotonicFunction } = {}) {
  return [
    ...createClockSelectors(clockFunction, monotonicFunction),
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
  ]
}

/**
 * The block for the pure segments (`lib/`, `domain/`): the determinism selectors and the timers.
 * A package that sets `no-restricted-syntax` for its sources adds this block after its own.
 *
 * @param {{ clockFunction?: string, monotonicFunction?: string }} options The sanctioned clock functions
 * @returns {object}
 */
export function createPureSegmentConfig(options = {}) {
  return {
    files: ['**/lib/**/*.ts', '**/domain/**/*.ts'],
    ignores: ['**/*.test.ts', '**/*.stories.ts', '**/fixtures/**', '**/testing/**', '.storybook/**', ...(options.portIgnores || [])],
    rules: {
      'no-restricted-syntax': ['error', ...createDeterminismSelectors(options), ...createTimerSelectors()],
    },
  }
}

/**
 * The block that bans, in the sources of `files`, the globals with a lifetime a test must hold
 * still and the calls that read the clock, randomness or the window.
 *
 * @param {{ files: string[], ignores?: string[], clockFunction?: string, monotonicFunction?: string }} options
 * @returns {object}
 */
export function createPortConfig({ files, ignores = [], ...clockFunctions }) {
  return {
    files,
    ignores,
    rules: {
      'no-restricted-globals': [
        'error',
        ...lifetimes.map(({ name, port }) => ({
          name,
          message: `${name} has a lifetime a test must hold still — take ${port}`,
        })),
      ],
      'no-restricted-syntax': ['error', ...createDeterminismSelectors(clockFunctions)],
    },
  }
}

/**
 * The rules every package enforces on every file, tests included. A promise is awaited, returned
 * or given an error channel; `void` never discards a value; no block or function body is empty.
 *
 * @param {{ isTyped?: boolean }} options `isTyped` adds the rules that need type information
 * @returns {Record<string, unknown>}
 */
export function createStrictRules({ isTyped = true } = {}) {
  return {
    'no-void': 'error',
    'no-empty': ['error', { allowEmptyCatch: false }],
    'no-empty-function': 'off',
    '@typescript-eslint/no-empty-function': 'error',
    '@typescript-eslint/consistent-type-imports': 'error',
    '@typescript-eslint/no-non-null-assertion': 'error',
    ...(isTyped
      ? {
          '@typescript-eslint/no-unnecessary-condition': 'error',
          '@typescript-eslint/no-unnecessary-type-assertion': 'error',
          '@typescript-eslint/require-await': 'error',
          '@typescript-eslint/unbound-method': 'error',
          '@typescript-eslint/no-floating-promises': ['error', { ignoreVoid: false }],
          '@typescript-eslint/no-misused-promises': 'error',
          '@typescript-eslint/await-thenable': 'error',
          '@typescript-eslint/switch-exhaustiveness-check': [
            'error',
            { considerDefaultExhaustiveForUnions: true },
          ],
        }
      : {}),
  }
}

/**
 * The strict rules for plain JavaScript files, which carry no TypeScript plugin and no type
 * information.
 *
 * @returns {Record<string, unknown>}
 */
export function createStrictScriptRules() {
  return {
    'no-void': 'error',
    'no-empty': ['error', { allowEmptyCatch: false }],
    'no-empty-function': 'error',
  }
}

export { numenPlugin }

/** The rule is shared with configurations that do not build on createBaseConfig. */
export const maxComposableReturnProperties = numenPlugin.rules['max-composable-return-properties']

/**
 * Returns the unified base ESLint flat configuration array for Vue 3 + TypeScript projects.
 *
 * @param {string} dirname The import.meta.dirname of the package
 * @param {object} options Optional overrides; `clockFunction` and `monotonicFunction` name the one function allowed to read each clock; `defaultProject` replaces the files linted outside tsconfig.json
 * @returns {Array}
 */
export function createBaseConfig(dirname, options = {}) {
  const allowDefault = options.defaultProject ?? [
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
        ...createStrictRules({ isTyped: useProjectService }),
        '@typescript-eslint/no-unused-vars': [
          'error',
          { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
        ],
        'vue/multi-word-component-names': 'off',
        'max-depth': ['error', 3],
        complexity: ['error', 10],
        'max-lines': ['error', { max: 350, skipBlankLines: true, skipComments: true }],
        'max-params': ['error', 4],
        'max-lines-per-function': ['error', { max: 80, skipBlankLines: true, skipComments: true, IIFEs: true }],
        'max-statements': ['error', 50],
        'numen/max-composable-return-properties': ['error', { max: 8 }],
        'numen/enforce-vue-reactivity-in-composable': 'error',
        'numen/no-double-type-assertion': 'error',
        'numen/no-unknown': 'error',
        'numen/no-comment-only-catch': 'error',
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
        'numen/vue-script-sections': 'error',
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
          ...createDeterminismSelectors(options),
        ],
      },
    },

    createPureSegmentConfig(options),

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
        'numen/pure-segment': 'error',
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
      files: ['**/*.test.ts', '**/*.stories.ts', '.storybook/**'],
      rules: {
        '@typescript-eslint/no-explicit-any': 'off',
        '@typescript-eslint/no-non-null-assertion': 'off',
        '@typescript-eslint/no-unnecessary-condition': 'off',
        '@typescript-eslint/no-unnecessary-type-assertion': 'off',
        '@typescript-eslint/require-await': 'off',
        '@typescript-eslint/unbound-method': 'off',
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
        'numen/no-unknown': 'off',
        'numen/no-comment-only-catch': 'off',
        '@typescript-eslint/consistent-type-assertions': 'off',
        'max-depth': 'off',
        complexity: 'off',
        'max-lines': 'off',
        'max-params': 'off',
        'max-lines-per-function': 'off',
        'max-statements': 'off',
      },
    },

    {
      files: ['**/*.spec.ts', '**/testing/**', '**/fixtures/**'],
      rules: {
        'max-lines': 'off',
        'numen/max-composable-return-properties': 'off',
        'max-lines-per-function': 'off',
        'max-statements': 'off',
      },
    },

    {
      files: ['**/*.test.ts', '**/*.spec.ts', '**/*.stories.ts', '**/testing/**/*.ts', '**/fixtures/**/*.ts'],
      rules: { 'numen/no-style-declaration-read': 'error' },
    },

    {
      files: ['**/*.test.ts'],
      rules: { 'numen/no-mock-of-subject-slice': 'error' },
    },

    prettier,
  ]
}

/**
 * The flat configuration of a TypeScript library without Vue (`libs/core/client`,
 * `libs/plugin/loader`): the strict rules, the size limits, the package boundary and the
 * determinism of its sources.
 *
 * @param {string} dirname The import.meta.dirname of the package
 * @returns {Array}
 */
export function createPlainLibraryConfig(dirname) {
  const sources = { files: ['**/*.ts'], ignores: ['**/*.test.ts', '**/*.spec.ts'] }
  return [
    { ignores: ['dist/**', 'coverage/**', 'node_modules/**'] },
    js.configs.recommended,
    ...tseslint.configs.recommended,
    {
      plugins: { numen: numenPlugin },
      languageOptions: {
        globals: globals.browser,
        parserOptions: {
          projectService: { allowDefaultProject: ['eslint.config.js'] },
          tsconfigRootDir: dirname,
        },
      },
      rules: {
        'no-console': 'error',
        ...createStrictRules(),
        '@typescript-eslint/no-unused-vars': [
          'error',
          { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
        ],
      },
    },
    {
      ...sources,
      rules: {
        'numen/no-unknown': 'error',
        'max-lines': ['error', { max: 350, skipBlankLines: true, skipComments: true }],
        'max-lines-per-function': ['error', { max: 60, skipBlankLines: true, skipComments: true }],
        complexity: ['error', 10],
        'max-depth': ['error', 3],
        'max-params': ['error', 3],
        'no-nested-ternary': 'error',
        'no-restricted-imports': [
          'error',
          {
            patterns: [
              {
                group: ['**/libs/**', '**/apps/**'],
                message:
                  'reach libraries and other applications by their package names (@numen/*), never by relative paths out of the module',
              },
            ],
          },
        ],
        'no-restricted-syntax': [
          'error',
          ...createClockSelectors(),
          {
            selector: 'MemberExpression[object.name="Math"][property.name="random"]',
            message: 'the same input gives the same numbers on any machine on any day',
          },
        ],
      },
    },
    {
      files: ['**/*.test.ts', '**/*.spec.ts'],
      rules: {
        'numen/no-style-declaration-read': 'error',
        'numen/no-mock-of-subject-slice': 'error',
        '@typescript-eslint/no-explicit-any': 'off',
        '@typescript-eslint/no-non-null-assertion': 'off',
        '@typescript-eslint/no-unnecessary-condition': 'off',
        '@typescript-eslint/no-unnecessary-type-assertion': 'off',
        '@typescript-eslint/require-await': 'off',
        '@typescript-eslint/unbound-method': 'off',
        'no-console': 'off',
      },
    },
    prettier,
  ]
}

/**
 * The recommended rules of `eslint-plugin-storybook` for story files, with `csf-component` off: the link from a
 * story to its component is the import graph (`public-component-has-story`), and a screen story has no single
 * component.
 *
 * @param {{ configs: Record<string, Array> }} storybook The `eslint-plugin-storybook` module, imported by the package that declares it
 * @returns {Array}
 */
export function createStoryConfig(storybook) {
  return [...storybook.configs['flat/recommended'], { files: ['**/*.stories.ts'], rules: { 'storybook/csf-component': 'off' } }]
}
