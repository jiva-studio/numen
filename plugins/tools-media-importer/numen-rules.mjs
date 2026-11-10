// The component and discipline rules of the numen ESLint plugin. The file imports nothing, so
// plugins/.example/numen-rules.mjs is a byte-identical copy of it: a plugin configuration imports nothing outside its directory.

function stripBlockComments(text) {
  let result = ''
  let cursor = 0
  while (cursor < text.length) {
    const open = text.indexOf('/*', cursor)
    if (open === -1) {
      result += text.slice(cursor)
      break
    }
    result += text.slice(cursor, open)
    const close = text.indexOf('*/', open + 2)
    if (close === -1) break
    cursor = close + 2
  }
  return result
}

function getScriptBody(sourceCode, script) {
  if (script.startTag && script.endTag) {
    return sourceCode.text.slice(script.startTag.range[1], script.endTag.range[0])
  }
  const text = sourceCode.getText(script)
  const openEnd = text.indexOf('>')
  const closeStart = text.lastIndexOf('<')
  if (openEnd === -1 || closeStart <= openEnd) return text
  return text.slice(openEnd + 1, closeStart)
}

function countScriptCodeLines(sourceCode, script) {
  const stripped = stripBlockComments(getScriptBody(sourceCode, script))
  return stripped.split('\n').filter((l) => {
    const t = l.trim()
    return t.length > 0 && !t.startsWith('//')
  }).length
}

/** A function whose name says it builds a part of a window: a composable or a factory. */
const isPartName = (name) => /^(use|create)[A-Z]/.test(name)

/** The variable a name refers to from a node, or null. */
function findVariable(context, node, name) {
  for (let scope = context.sourceCode.getScope(node); scope; scope = scope.upper) {
    const variable = scope.set.get(name)
    if (variable) return variable
  }
  return null
}

/** True when a variable is bound to what a use* or create* call returns, whole or destructured. */
function isBoundToPart(variable) {
  const definition = variable?.defs[0]?.node
  if (definition?.type !== 'VariableDeclarator') return false
  const init = definition.init?.type === 'AwaitExpression' ? definition.init.argument : definition.init
  if (init?.type !== 'CallExpression') return false
  const callee = init.callee.type === 'MemberExpression' ? init.callee.property : init.callee
  return callee.type === 'Identifier' && isPartName(callee.name)
}

const maxComposableReturnProperties = {
  meta: {
    type: 'problem',
    docs: {
      description:
        'Enforce a maximum number of properties a composable (use*) or a factory (create*) returns, whether it returns an object literal, a call result or a typed value, and forbid groups that only bundle',
    },
    messages: {
      tooManyReturnedProperties:
        '"{{name}}" returns {{count}} properties (max: {{max}}). Decompose into smaller focused composables or factories, or group the result into named parts.',
      spreadInReturn:
        '"{{name}}" spreads an object into what it returns, which hides how many properties it returns. Name each property.',
      singleKeyGroup:
        '"{{name}}" returns the group "{{group}}" with one key. A group of one key adds a level and no meaning: return the key itself.',
      reexposedGroup:
        '"{{name}}" returns the group "{{group}}" that only re-exposes {{count}} members of the parts it calls. Let the consumer take the parts, or give the group a meaning of its own.',
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
    const services = context.sourceCode.parserServices
    const checker = services?.program?.getTypeChecker() ?? null

    // The object literal a returned expression is: the literal itself, or the literal a returned
    // identifier is declared with.
    function getReturnedObject(argument) {
      if (argument?.type === 'ObjectExpression') return argument
      if (argument?.type !== 'Identifier') return null
      const declarator = findVariable(context, argument, argument.name)?.defs[0]?.node
      return declarator?.type === 'VariableDeclarator' && declarator.init?.type === 'ObjectExpression'
        ? declarator.init
        : null
    }

    // The property count of a value that is no object literal, read from its type. A type declared in a
    // declaration file (a Ref, a Promise, an array) is not a bag of the function's own and counts nothing.
    function countTypeProperties(argument) {
      if (!checker) return 0
      const tsNode = services.esTreeNodeToTSNodeMap.get(argument)
      if (!tsNode) return 0
      const type = checker.getNonNullableType(checker.getTypeAtLocation(tsNode))
      const members = type.isUnion() ? type.types : [type]
      let most = 0
      for (const member of members) {
        const symbol = member.getSymbol()
        const declarations = symbol?.getDeclarations() ?? []
        const isOwn = declarations.length > 0 && declarations.every((one) => !one.getSourceFile().isDeclarationFile)
        if (!isOwn || checker.isArrayType(member) || checker.isTupleType(member)) continue
        if (member.getCallSignatures().length > 0) continue
        most = Math.max(most, member.getProperties().length)
      }
      return most
    }

    function isReexport(member) {
      let value = member.value
      if (value.type === 'MemberExpression') {
        while (value.type === 'MemberExpression') value = value.object
      }
      return value.type === 'Identifier' && isBoundToPart(findVariable(context, value, value.name))
    }

    function checkGroups(node, fnName) {
      for (const prop of node.properties) {
        if (prop.type !== 'Property' || prop.value.type !== 'ObjectExpression') continue
        const group = prop.key.name ?? prop.key.value
        const members = prop.value.properties.filter((one) => one.type === 'Property')
        if (members.length === 1) {
          context.report({ node: prop, messageId: 'singleKeyGroup', data: { name: fnName, group } })
          continue
        }
        const count = members.filter(isReexport).length
        if (count >= 3) {
          context.report({ node: prop, messageId: 'reexposedGroup', data: { name: fnName, group, count } })
        }
      }
    }

    function checkReturned(argument, fnName) {
      if (!fnName || !isPartName(fnName) || !argument) return
      const node = getReturnedObject(argument)
      let count = 0
      if (node) {
        for (const prop of node.properties) {
          if (prop.type === 'Property') {
            count++
          } else if (prop.type === 'SpreadElement') {
            // A composable names every property it returns; a factory that spreads is counted by its type.
            if (fnName.startsWith('use')) {
              context.report({ node: prop, messageId: 'spreadInReturn', data: { name: fnName } })
            } else {
              count = Math.max(count, countTypeProperties(node))
            }
          }
        }
        checkGroups(node, fnName)
      } else {
        count = countTypeProperties(argument)
      }
      if (count > max) {
        context.report({
          node: node ?? argument,
          messageId: 'tooManyReturnedProperties',
          data: { name: fnName, count, max },
        })
      }
    }

    function checkBody(body, fnName) {
      if (body.type !== 'BlockStatement') {
        checkReturned(body, fnName)
        return
      }
      for (const stmt of body.body) {
        if (stmt.type === 'ReturnStatement') checkReturned(stmt.argument, fnName)
      }
    }

    return {
      'FunctionDeclaration:exit'(node) {
        if (node.id) checkBody(node.body, node.id.name)
      },
      'VariableDeclarator:exit'(node) {
        const init = node.init
        if (node.id?.type !== 'Identifier' || !init) return
        if (init.type === 'ArrowFunctionExpression' || init.type === 'FunctionExpression') {
          checkBody(init.body, node.id.name)
        }
      },
    }
  },
}

/**
 * Custom rules for Numen project conventions:
 * 1. max-inline-props: Enforce extracting large props interfaces (> 6 fields) into *.types.ts.
 * 2. vue-script-sections: Enforce the canonical order of the top-level statements of a script setup (AGENTS.md rule 4).
 * 3. max-script-lines: Enforce max lines of executable code in <script> (ignoring comments/banners).
 */
export default {
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
          description:
            'Enforce the canonical order of the top-level statements of a script setup: props and events, state, hooks, handlers, helpers',
        },
        messages: {
          outOfOrder:
            '{{kind}} "{{name}}" comes after {{previous}}. Order of a script: props and events, state, hooks, handlers (on<Action>), helpers.',
        },
        schema: [],
      },
      create(context) {
        const HOOKS = new Set([
          'watch',
          'watchEffect',
          'watchSyncEffect',
          'watchPostEffect',
          'onMounted',
          'onUnmounted',
          'onBeforeMount',
          'onBeforeUnmount',
          'onUpdated',
          'onBeforeUpdate',
          'onActivated',
          'onDeactivated',
          'onErrorCaptured',
          'onScopeDispose',
        ])
        const MACROS = new Set(['defineProps', 'defineEmits', 'defineModel', 'defineSlots', 'defineOptions', 'defineExpose', 'withDefaults'])
        const KINDS = { 1: 'props or events', 3: 'state', 4: 'hook', 5: 'handler', 6: 'helper' }

        const isFunction = (node) =>
          node?.type === 'ArrowFunctionExpression' || node?.type === 'FunctionExpression'

        function getCalleeName(node) {
          const call = node?.type === 'AwaitExpression' ? node.argument : node
          return call?.type === 'CallExpression' && call.callee.type === 'Identifier' ? call.callee.name : null
        }

        // The rank and name of a top-level statement; null for a statement that has no place in the order.
        function classify(statement) {
          if (statement.type === 'FunctionDeclaration') {
            return { rank: /^on[A-Z]/.test(statement.id.name) ? 5 : 6, name: statement.id.name }
          }
          if (statement.type === 'ExpressionStatement') {
            const callee = getCalleeName(statement.expression)
            if (MACROS.has(callee)) return { rank: 1, name: callee }
            return { rank: HOOKS.has(callee) ? 4 : 3, name: callee ?? 'statement' }
          }
          if (statement.type !== 'VariableDeclaration') return null
          const declarator = statement.declarations[0]
          const name = declarator.id.type === 'Identifier' ? declarator.id.name : 'destructuring'
          const callee = getCalleeName(declarator.init)
          if (MACROS.has(callee)) return { rank: 1, name }
          if (HOOKS.has(callee)) return { rank: 4, name }
          // A function held in a constant cannot move before the statements that call it at once (it is not hoisted),
          // so only a handler named on<Action> takes a place in the order.
          if (isFunction(declarator.init)) return /^on[A-Z]/.test(name) ? { rank: 5, name } : null
          return { rank: 3, name }
        }

        return {
          Program(program) {
            let previous = null
            for (const statement of program.body) {
              const placed = classify(statement)
              if (!placed) continue
              if (previous && placed.rank < previous.rank) {
                context.report({
                  node: statement,
                  messageId: 'outOfOrder',
                  data: { kind: KINDS[placed.rank] ?? 'statement', name: placed.name, previous: `${KINDS[previous.rank]} "${previous.name}"` },
                })
              }
              if (!previous || placed.rank > previous.rank) previous = placed
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

            const codeLines = countScriptCodeLines(context.sourceCode, script)

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

            if (isBoolean && name !== 'modelValue' && !/^(is|has|can)[A-Z]/.test(name)) {
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
            'Enforce on<Action> for a function a template binds to an event, and forbid a handle... prefix on any function of the script',
        },
        messages: {
          invalidHandlerName:
            'Event handler "{{name}}" must be named on{{capitalized}}. A name that starts with "handle" is forbidden.',
          forbiddenHandlePrefix: 'Function "{{name}}" starts with "handle". Name what it does, or on{{capitalized}} when it answers an event.',
        },
      },
      create(context) {
        const services = context.sourceCode.parserServices
        if (!services?.defineTemplateBodyVisitor) return {}
        const declared = new Map()
        const bound = new Set()

        const declare = (name, node) => {
          declared.set(name, node)
          if (/^handle[A-Z]/.test(name)) {
            context.report({
              node,
              messageId: 'forbiddenHandlePrefix',
              data: { name, capitalized: name.slice('handle'.length) },
            })
          }
        }

        // The function a handler expression of the template names: `foo`, `foo($event)`.
        const getBoundName = (expression) => {
          if (expression?.type === 'VOnExpression') return getBoundName(expression.body[0]?.expression)
          if (expression?.type === 'Identifier') return expression.name
          if (expression?.type === 'CallExpression' && expression.callee.type === 'Identifier') return expression.callee.name
          return null
        }

        return services.defineTemplateBodyVisitor(
          {
            "VAttribute[directive=true][key.name.name='on']"(node) {
              const name = getBoundName(node.value?.expression)
              if (name) bound.add(name)
            },
            "VElement[name='template']:exit"(node) {
              if (node.parent.type !== 'VDocumentFragment') return
              for (const name of bound) {
                const at = declared.get(name)
                if (!at || /^on[A-Z]/.test(name) || /^handle[A-Z]/.test(name)) continue
                context.report({
                  node: at,
                  messageId: 'invalidHandlerName',
                  data: { name, capitalized: name.charAt(0).toUpperCase() + name.slice(1) },
                })
              }
            },
          },
          {
            'Program > FunctionDeclaration'(node) {
              if (node.id) declare(node.id.name, node.id)
            },
            'Program > VariableDeclaration > VariableDeclarator'(node) {
              const isFunction =
                node.init?.type === 'ArrowFunctionExpression' || node.init?.type === 'FunctionExpression'
              if (node.id.type === 'Identifier' && isFunction) declare(node.id.name, node.id)
            },
          },
        )
      },
    },

    'no-mock-of-subject-slice': {
      meta: {
        type: 'problem',
        docs: {
          description:
            'Forbid vi.mock of a module in the same slice as the test: a test that replaces its own slice verifies the replacement',
        },
        messages: {
          mocksOwnSlice:
            'vi.mock("{{specifier}}") replaces a module of the slice "{{slice}}" that this test verifies. Use the real module and fake only what lies outside the slice (a transport, a port).',
        },
        schema: [],
      },
      create(context) {
        const SLICED_LAYERS = new Set(['entities', 'features', 'pages', 'widgets', 'app', 'shared'])
        const filename = (context.filename ?? '').replaceAll('\\', '/')
        const root = /^(.*\/(?:src|frontend))\//.exec(filename)
        if (!root) return {}
        const rootDir = root[1]

        // The slice a path belongs to: layer and folder of an FSD layer, otherwise the first folder
        // under the source root; a file directly in the source root belongs to the root.
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

        return {
          'CallExpression[callee.object.name="vi"][callee.property.name=/^(mock|doMock)$/]'(node) {
            const specifier = node.arguments[0]?.value
            if (typeof specifier !== 'string') return
            const target = resolve(specifier)
            if (!target || !target.startsWith(`${rootDir}/`)) return
            const slice = getSlice(filename)
            if (getSlice(`${target}/x`) === slice || getSlice(target) === slice) {
              context.report({ node, messageId: 'mocksOwnSlice', data: { specifier, slice } })
            }
          },
        }
      },
    },

    'max-composable-return-properties': maxComposableReturnProperties,

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

    'no-style-declaration-read': {
      meta: {
        type: 'problem',
        docs: {
          description:
            'Forbid reading a CSSStyleDeclaration (getComputedStyle(...), element.style) in a test or story: they check behavior, not appearance',
        },
        messages: {
          forbiddenRead:
            'a test or story checks what the user can do (open, close, flip, move), never a style value: assert state, an attribute, an event or what is on screen',
        },
        schema: [],
      },
      create(context) {
        const services = context.sourceCode.parserServices
        const checker = services?.program?.getTypeChecker() ?? null
        const isDeclaration = (type, seen = new Set()) => {
          if (seen.has(type)) return false
          seen.add(type)
          if (type.getSymbol()?.getName() === 'CSSStyleDeclaration') return true
          if (type.isUnionOrIntersection()) return type.types.some((one) => isDeclaration(one, seen))
          return type.isClassOrInterface() && checker.getBaseTypes(type).some((one) => isDeclaration(one, seen))
        }
        const hasDeclarationType = (node) =>
          checker !== null && isDeclaration(checker.getTypeAtLocation(services.esTreeNodeToTSNodeMap.get(node)))
        const isBase = (node) => node.parent.type === 'MemberExpression' && node.parent.object === node
        const isWrite = (node) =>
          (node.parent.type === 'AssignmentExpression' && node.parent.left === node) ||
          (node.property.type === 'Identifier' && ['setProperty', 'removeProperty'].includes(node.property.name))
        // A style asserted without a declaration: `toHaveStyle`, `getAttribute('style')`, `expect(x.style)`.
        const isStyleAssertion = (node) => {
          if (node.type !== 'CallExpression') return false
          const callee = node.callee
          const name = callee.type === 'MemberExpression' ? callee.property.name : callee.name
          if (name === 'toHaveStyle') return true
          if (name === 'getAttribute') return node.arguments[0]?.value === 'style'
          if (name !== 'expect') return false
          let argument = node.arguments[0]
          if (argument?.type === 'ChainExpression') argument = argument.expression
          return argument?.type === 'MemberExpression' && ['style', 'cssText'].includes(argument.property.name)
        }
        return {
          'CallExpression, MemberExpression'(node) {
            if (isStyleAssertion(node)) {
              context.report({ node, messageId: 'forbiddenRead' })
              return
            }
            // A declaration that is kept or passed on (`const style = getComputedStyle(x)`, `read(x.style)`).
            if (!isBase(node) && hasDeclarationType(node)) {
              context.report({ node, messageId: 'forbiddenRead' })
              return
            }
            // A property or method of a declaration that is read.
            if (node.type === 'MemberExpression' && hasDeclarationType(node.object) && !isWrite(node)) {
              context.report({ node, messageId: 'forbiddenRead' })
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
