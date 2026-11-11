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

// The primitive types that hold data, by the name TypeScript prints for them; the literal types are found by the type itself.
const DATA_TYPE_NAMES = new Set(['string', 'number', 'boolean', 'bigint', 'undefined', 'null', 'void', 'never', 'true', 'false'])

const maxComposableReturnProperties = {
  meta: {
    type: 'problem',
    docs: {
      description:
        'Enforce a maximum number of properties a composable (use*) or a factory (create*) returns, and a factory takes in its input object, whether it returns an object literal, a call result or a typed value, and forbid groups that only bundle',
    },
    messages: {
      tooManyReturnedProperties:
        '"{{name}}" returns {{count}} properties (max: {{max}}). Decompose into smaller focused composables or factories, or group the result into named parts.',
      spreadInReturn:
        '"{{name}}" spreads an object into what it returns, which hides how many properties it returns. Name each property.',
      spreadOfInput:
        '"{{name}}" spreads a value it did not build into what it returns, which hides what the result holds. Name each property.',
      tooManyInputProperties:
        '"{{name}}" takes {{count}} properties in its input (max: {{max}}). Narrow the input to what the function uses, or split the function.',
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

    // True when a value holds data only: primitives, and objects and arrays declared in the project whose
    // members are data in turn. A words constant and a plain record are data; a service, a callback and a
    // type from a declaration file (a Ref, a Map, a Date) are not.
    function isPlainData(type, node, seen = new Set()) {
      if (type.isLiteral() || DATA_TYPE_NAMES.has(checker.typeToString(type))) return true
      if (type.isUnionOrIntersection()) return type.types.every((one) => isPlainData(one, node, seen))
      if (seen.has(type)) return true
      seen.add(type)
      if (checker.isArrayType(type) || checker.isTupleType(type)) {
        return checker.getTypeArguments(type).every((one) => isPlainData(one, node, seen))
      }
      if (type.getCallSignatures().length > 0 || type.getConstructSignatures().length > 0) return false
      const declarations = type.getSymbol()?.getDeclarations() ?? []
      if (declarations.length === 0 || declarations.some((one) => one.getSourceFile().isDeclarationFile)) return false
      return type.getProperties().every((one) => isPlainData(checker.getTypeOfSymbolAtLocation(one, node), node, seen))
    }

    // The properties of the project's own object type that are no plain data, or null when there is no such type.
    // A record of data only counts none: it is not a bag of dependencies. A type declared in a declaration file
    // (a Ref, a Promise, an array) is not a bag of the function's own either.
    function countDependencyProperties(argument) {
      if (!checker) return null
      const tsNode = services.esTreeNodeToTSNodeMap.get(argument)
      if (!tsNode) return null
      const type = checker.getNonNullableType(checker.getTypeAtLocation(tsNode))
      const members = type.isUnion() ? type.types : [type]
      let most = null
      for (const member of members) {
        const declarations = member.getSymbol()?.getDeclarations() ?? []
        const isOwn = declarations.length > 0 && declarations.every((one) => !one.getSourceFile().isDeclarationFile)
        if (!isOwn || checker.isArrayType(member) || checker.isTupleType(member)) continue
        if (member.getCallSignatures().length > 0) continue
        const count = member.getProperties().filter((one) => !isPlainData(checker.getTypeOfSymbolAtLocation(one, tsNode), tsNode)).length
        most = Math.max(most ?? 0, count)
      }
      return most
    }

    // True when the value of a property is plain data, which no count of dependencies includes.
    function isDataValue(value) {
      if (!checker) return false
      const tsNode = services.esTreeNodeToTSNodeMap.get(value)
      return tsNode ? isPlainData(checker.getTypeAtLocation(tsNode), tsNode) : false
    }

    function countTypeProperties(argument) {
      return countDependencyProperties(argument) ?? 0
    }

    // The members of a parameter that is an object pattern, an inline type literal or a type declared in the
    // file, or else the properties of its type when type information is available.
    function countInputProperties(param) {
      const target = param.type === 'AssignmentPattern' ? param.left : param
      const counted = countDependencyProperties(target)
      if (counted !== null) return counted
      if (target.type === 'ObjectPattern') return target.properties.length
      const annotation = target.typeAnnotation?.typeAnnotation
      if (annotation?.type === 'TSTypeLiteral') return annotation.members.length
      if (annotation?.type !== 'TSTypeReference' || annotation.typeName.type !== 'Identifier') return 0
      const declaration = findVariable(context, annotation, annotation.typeName.name)?.defs[0]?.node
      if (declaration?.type === 'TSInterfaceDeclaration') return declaration.body.body.length
      if (declaration?.type === 'TSTypeAliasDeclaration' && declaration.typeAnnotation.type === 'TSTypeLiteral') {
        return declaration.typeAnnotation.members.length
      }
      return 0
    }

    function checkInput(params, fnName) {
      if (!fnName.startsWith('create')) return
      for (const param of params) {
        const count = countInputProperties(param)
        if (count > max) {
          context.report({ node: param, messageId: 'tooManyInputProperties', data: { name: fnName, count, max } })
        }
      }
    }

    // True when a spread takes a value from outside the function: a parameter, a captured variable, a member or a
    // call result. A value the function builds as an object literal is a part it names itself.
    function isSpreadOfInput(argument, fn) {
      if (argument.type !== 'Identifier') return true
      const declarator = findVariable(context, argument, argument.name)?.defs[0]?.node
      const isOwn =
        declarator?.type === 'VariableDeclarator' &&
        declarator.init?.type === 'ObjectExpression' &&
        declarator.range[0] >= fn.range[0] &&
        declarator.range[1] <= fn.range[1]
      return !isOwn
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

    function checkReturned(argument, fnName, fn) {
      if (!fnName || !isPartName(fnName) || !argument) return
      const node = getReturnedObject(argument)
      let count = 0
      if (node) {
        for (const prop of node.properties) {
          if (prop.type === 'Property') {
            if (!isDataValue(prop.value)) count++
          } else if (prop.type === 'SpreadElement') {
            // A composable names every property it returns; a factory that spreads a part it built is counted by its type, and one that spreads a value it did not build is reported.
            if (fnName.startsWith('use')) {
              context.report({ node: prop, messageId: 'spreadInReturn', data: { name: fnName } })
            } else {
              if (isSpreadOfInput(prop.argument, fn)) {
                context.report({ node: prop, messageId: 'spreadOfInput', data: { name: fnName } })
              }
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
        checkReturned(body, fnName, body.parent)
        return
      }
      for (const stmt of body.body) {
        if (stmt.type === 'ReturnStatement') checkReturned(stmt.argument, fnName, body.parent)
      }
    }

    return {
      'FunctionDeclaration:exit'(node) {
        if (node.id) {
          checkInput(node.params, node.id.name)
          checkBody(node.body, node.id.name)
        }
      },
      'VariableDeclarator:exit'(node) {
        const init = node.init
        if (node.id?.type !== 'Identifier' || !init) return
        if (init.type === 'ArrowFunctionExpression' || init.type === 'FunctionExpression') {
          checkInput(init.params, node.id.name)
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
/** The globals a pure segment may not reach: the window, the document, storage, timers, observers and the network. */
const BROWSER_GLOBALS = new Set([
  'window', 'document', 'navigator', 'localStorage', 'sessionStorage', 'location', 'history', 'screen',
  'performance', 'getComputedStyle', 'matchMedia', 'innerWidth', 'innerHeight', 'screenX', 'screenY',
  'setTimeout', 'clearTimeout', 'setInterval', 'clearInterval', 'requestAnimationFrame',
  'cancelAnimationFrame', 'requestIdleCallback', 'cancelIdleCallback', 'queueMicrotask',
  'ResizeObserver', 'IntersectionObserver', 'MutationObserver', 'fetch', 'XMLHttpRequest',
  'WebSocket', 'EventSource', 'Worker', 'CSS', 'alert', 'confirm', 'prompt',
])

/** The DOM types a pure segment may not name when they are the platform's own and no file declares them. */
const DOM_TYPES = new Set([
  'Element', 'Node', 'ParentNode', 'ChildNode', 'NodeList', 'Document', 'DocumentFragment', 'ShadowRoot',
  'Window', 'Navigator', 'Storage', 'Location', 'History', 'EventTarget', 'Event', 'KeyboardEvent',
  'MouseEvent', 'PointerEvent', 'WheelEvent', 'DragEvent', 'ClipboardEvent', 'FocusEvent', 'InputEvent',
  'TouchEvent', 'CustomEvent', 'UIEvent', 'CompositionEvent', 'MessageEvent', 'StorageEvent', 'DOMRect',
  'DOMRectReadOnly', 'DOMTokenList', 'DOMException', 'Range', 'Selection', 'CSSStyleDeclaration',
  'ResizeObserverSize', 'ResizeObserverEntry', 'DataTransfer',
])

const isPlatformElementType = (name) => /^(HTML|SVG|MathML)\w*Element$/.test(name)

/** The members that build, query or listen to the DOM, wherever the object came from. */
const DOM_MEMBERS = new Set([
  'addEventListener', 'removeEventListener', 'dispatchEvent', 'createElement', 'createElementNS',
  'createTextNode', 'createRange', 'createDocumentFragment', 'querySelector', 'querySelectorAll',
  'getElementById', 'getBoundingClientRect', 'appendChild', 'replaceChildren', 'setAttribute',
  'removeAttribute', 'classList', 'dataset', 'innerHTML', 'execCommand', 'getSelection',
])

/** The platform ports of `@numen/ui`: a pure segment takes what they provide as a parameter. */
const PLATFORM_PORTS = new Set([
  'doc', 'win', 'nav', 'hasWindow', 'hasDocument', 'measureWindow', 'measureRect', 'measureElementRect',
  'getNow', 'getCurrentDate', 'getMonotonicNow', 'browserClock', 'browserViewport', 'onNextFrame',
  'loadWailsRuntime', 'openExternalUrl', 'interceptExternalLinks',
])

const isFrameworkImport = (source) =>
  source === 'vue' || source === 'pinia' || source === 'vue-router' || source.startsWith('@vue/') || source.endsWith('.vue')

const isUndeclared = (reference) => !reference.resolved || reference.resolved.defs.length === 0

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

    'no-unknown': {
      meta: {
        type: 'problem',
        docs: {
          description:
            'Forbid `unknown`: name the real type, JsonValue for JSON, Caught for a caught error, and Unvalidated only for the parameter of a parse function',
        },
        messages: {
          unknownType:
            '`unknown` is not a type to write. Name the type, use JsonValue for JSON, Caught for a caught value, or Unvalidated for the parameter of a parse function.',
          unvalidatedOutsideParse:
            '`Unvalidated` is the parameter of a function that validates it: name the function parseX, or return a type guard.',
          unvalidatedNotNarrowed:
            'The `Unvalidated` parameter "{{ name }}" is never narrowed: check it with typeof, instanceof, in, a guard or a schema call in the body.',
          caughtOutsideCatch:
            '`Caught` is the type of a caught error: the parameter of a catch clause or of an error callback, named error, err, e, reason or cause.',
        },
      },
      create(context) {
        const HELPERS = new Set(['Caught', 'Unvalidated'])
        const FUNCTIONS = new Set([
          'FunctionDeclaration',
          'FunctionExpression',
          'ArrowFunctionExpression',
          'TSDeclareFunction',
        ])
        const getFunctionName = (fn) => {
          if (fn.id) return fn.id.name
          const holder = fn.parent
          if (holder?.type === 'VariableDeclarator' && holder.id.type === 'Identifier') return holder.id.name
          if ((holder?.type === 'MethodDefinition' || holder?.type === 'Property') && holder.key.type === 'Identifier')
            return holder.key.name
          return ''
        }
        const isParseBoundary = (node) => {
          const annotation = node.parent
          const param = annotation?.parent
          if (annotation?.type !== 'TSTypeAnnotation' || param?.type !== 'Identifier') return false
          const fn = param.parent
          if (!fn || !FUNCTIONS.has(fn.type) || !fn.params.includes(param)) return false
          return fn.returnType?.typeAnnotation?.type === 'TSTypePredicate' || /^parse[A-Z]/.test(getFunctionName(fn))
        }
        const getParameter = (node) => {
          const annotation = node.parent
          const param = annotation?.parent
          return annotation?.type === 'TSTypeAnnotation' && param?.type === 'Identifier' ? param : undefined
        }
        const ERROR_PARAMETER = /^(error|err|e|reason|cause|caught|ex|exception|thrown)$/i
        const CALLABLES = new Set([
          ...FUNCTIONS,
          'TSFunctionType',
          'TSMethodSignature',
          'TSCallSignatureDeclaration',
          'TSEmptyBodyFunctionExpression',
          'TSConstructorType',
        ])
        const isCaughtPosition = (node) => {
          const param = getParameter(node)
          if (!param || !ERROR_PARAMETER.test(param.name)) return false
          const owner = param.parent
          return owner?.type === 'CatchClause' || (CALLABLES.has(owner?.type) && owner.params.includes(param))
        }
        /** The guard and schema calls that narrow a value: `isX(v)`, `assertX(v)`, `parseX(v)`, `schema.parse(v)`, `Array.isArray(v)`. */
        const NARROWING_CALL = /^(is|assert|parse|safeParse|validate|check|ensure|decode|has)([A-Z]|$)/
        const calleeName = (callee) =>
          callee.type === 'Identifier' ? callee.name : callee.type === 'MemberExpression' && !callee.computed ? callee.property.name : ''
        const isNamed = (node, name) => node?.type === 'Identifier' && node.name === name
        const isNarrowing = (node, name) => {
          if (node.type === 'UnaryExpression' && node.operator === 'typeof') return isNamed(node.argument, name)
          if (node.type === 'BinaryExpression' && node.operator === 'instanceof') return isNamed(node.left, name)
          if (node.type === 'BinaryExpression' && node.operator === 'in') return isNamed(node.right, name)
          return node.type === 'CallExpression' && NARROWING_CALL.test(calleeName(node.callee)) && node.arguments.some((one) => isNamed(one, name))
        }
        const narrows = (body, name) => {
          const keys = context.sourceCode.visitorKeys
          const visit = (node) => {
            if (isNarrowing(node, name)) return true
            return (keys[node.type] ?? []).some((key) => {
              const child = node[key]
              return Array.isArray(child) ? child.some((one) => one && visit(one)) : Boolean(child && typeof child.type === 'string' && visit(child))
            })
          }
          return visit(body)
        }
        return {
          TSUnknownKeyword(node) {
            const alias = node.parent
            const isHelper =
              alias?.type === 'TSTypeAliasDeclaration' && alias.typeAnnotation === node && HELPERS.has(alias.id.name)
            if (!isHelper) context.report({ node, messageId: 'unknownType' })
          },
          TSTypeReference(node) {
            if (node.typeName.type !== 'Identifier') return
            if (node.typeName.name === 'Caught' && !isCaughtPosition(node)) {
              context.report({ node, messageId: 'caughtOutsideCatch' })
            }
            if (node.typeName.name !== 'Unvalidated') return
            if (!isParseBoundary(node)) {
              context.report({ node, messageId: 'unvalidatedOutsideParse' })
              return
            }
            const param = getParameter(node)
            const fn = param.parent
            if (fn.body && !narrows(fn.body, param.name)) {
              context.report({ node, messageId: 'unvalidatedNotNarrowed', data: { name: param.name } })
            }
          },
        }
      },
    },

    'pure-segment': {
      meta: {
        type: 'problem',
        docs: {
          description:
            'Keep a lib segment pure TypeScript: no Vue, DOM, browser global, timer, clock or platform port',
        },
        messages: {
          framework: 'A pure segment imports no Vue, Pinia or component: "{{ source }}".',
          global: 'A pure segment does not reach the browser global "{{ name }}"; take it as a parameter.',
          domType: 'A pure segment does not name the DOM type "{{ name }}"; describe the fields it reads.',
          domMember: 'A pure segment does not build, query or listen to the DOM: "{{ name }}".',
          port: 'A pure segment does not import the platform port "{{ name }}"; take it as a parameter.',
          clock: 'A pure segment does not read the clock; take the time as a parameter.',
          random: 'A pure segment does not draw random numbers or identities; take them as a parameter.',
          computed: 'A pure segment does not reach "{{ name }}" through a computed key; name the member.',
        },
      },
      create(context) {
        const sourceCode = context.sourceCode
        const reportReference = (reference) => {
          const name = reference.identifier.name
          if (!isUndeclared(reference)) return
          if (BROWSER_GLOBALS.has(name)) {
            context.report({ node: reference.identifier, messageId: 'global', data: { name } })
          } else if (DOM_TYPES.has(name) || isPlatformElementType(name)) {
            context.report({ node: reference.identifier, messageId: 'domType', data: { name } })
          }
        }
        const findVariable = (node) => {
          for (let one = sourceCode.getScope(node); one; one = one.upper) {
            const found = one.set.get(node.name)
            if (found) return found
          }
          return undefined
        }
        /** The name of a member written `.name` or `['name']`, undefined for a computed key that is not a string. */
        const getMemberName = (node) => {
          if (!node.computed) return node.property.name
          if (node.property.type === 'Literal' && typeof node.property.value === 'string') return node.property.value
          if (node.property.type === 'TemplateLiteral' && node.property.expressions.length === 0) return node.property.quasis[0].value.cooked
          return undefined
        }
        /**
         * The global an expression stands for: the name `Date`, `globalThis.Date`, `globalThis['Date']`, a constant
         * initialised with one of them (`const D = Date`) and a name destructured from `globalThis` (`const { Date: D } = globalThis`).
         */
        const resolveGlobal = (node, seen = new Set()) => {
          if (node.type === 'MemberExpression') {
            const name = getMemberName(node)
            return name && resolveGlobal(node.object, seen) === 'globalThis' ? name : undefined
          }
          if (node.type !== 'Identifier') return undefined
          const variable = findVariable(node)
          if (!variable || variable.defs.length === 0) return node.name
          if (variable.defs.length !== 1 || seen.has(variable)) return undefined
          seen.add(variable)
          const def = variable.defs[0].node
          if (def?.type !== 'VariableDeclarator' || !def.init) return undefined
          if (def.id.type === 'Identifier') return resolveGlobal(def.init, seen)
          if (def.id.type !== 'ObjectPattern' || resolveGlobal(def.init, seen) !== 'globalThis') return undefined
          const property = def.id.properties.find((one) => one.type === 'Property' && one.value.type === 'Identifier' && one.value.name === node.name)
          return property && !property.computed && property.key.type === 'Identifier' ? property.key.name : undefined
        }
        const isGlobalName = (node, name) => resolveGlobal(node) === name
        /** The message for reaching member `member` of the global `owner`, or undefined when the member is pure. */
        const getImpureMessage = (owner, member) => {
          if (owner === 'Date' && member === 'now') return 'clock'
          if (owner === 'Math' && member === 'random') return 'random'
          if (owner === 'crypto' && (member === 'randomUUID' || member === 'getRandomValues')) return 'random'
          return undefined
        }
        const IMPURE_OWNERS = new Map([
          ['Date', 'clock'],
          ['Math', 'random'],
          ['crypto', 'random'],
        ])
        return {
          Program() {
            for (const scope of sourceCode.scopeManager.scopes) {
              for (const reference of scope.references) reportReference(reference)
            }
          },
          ImportDeclaration(node) {
            const source = String(node.source.value)
            if (isFrameworkImport(source)) context.report({ node, messageId: 'framework', data: { source } })
            if (source !== '@numen/ui') return
            for (const specifier of node.specifiers) {
              const name = specifier.type === 'ImportSpecifier' ? specifier.imported.name : ''
              if (PLATFORM_PORTS.has(name)) context.report({ node: specifier, messageId: 'port', data: { name } })
            }
          },
          MemberExpression(node) {
            const owner = resolveGlobal(node.object)
            const property = getMemberName(node)
            if (owner === 'globalThis' && property !== undefined && (BROWSER_GLOBALS.has(property) || DOM_TYPES.has(property))) {
              context.report({ node, messageId: 'global', data: { name: property } })
            }
            if (property && DOM_MEMBERS.has(property)) {
              context.report({ node: node.property, messageId: 'domMember', data: { name: property } })
            }
            const impure = owner && property !== undefined ? getImpureMessage(owner, property) : undefined
            if (impure) context.report({ node, messageId: impure })
            if (owner && property === undefined && node.computed && IMPURE_OWNERS.has(owner)) {
              context.report({ node, messageId: 'computed', data: { name: owner } })
            }
            if (owner === 'globalThis' && property === undefined && node.computed) {
              context.report({ node, messageId: 'computed', data: { name: owner } })
            }
          },
          VariableDeclarator(node) {
            if (node.id.type !== 'ObjectPattern' || !node.init) return
            const owner = resolveGlobal(node.init)
            if (!owner) return
            for (const property of node.id.properties) {
              if (property.type !== 'Property') continue
              const name = property.computed ? (property.key.type === 'Literal' ? String(property.key.value) : undefined) : property.key.name
              const impure = name === undefined ? undefined : getImpureMessage(owner, name)
              if (impure) context.report({ node: property, messageId: impure })
              if (name === undefined && IMPURE_OWNERS.has(owner)) context.report({ node: property, messageId: 'computed', data: { name: owner } })
              if (owner === 'globalThis' && name !== undefined && IMPURE_OWNERS.has(name) === false && (BROWSER_GLOBALS.has(name) || DOM_TYPES.has(name))) {
                context.report({ node: property, messageId: 'global', data: { name } })
              }
            }
          },
          CallExpression(node) {
            if (isGlobalName(node.callee, 'Date')) context.report({ node, messageId: 'clock' })
          },
          NewExpression(node) {
            if (!isGlobalName(node.callee, 'Date')) return
            if (node.arguments.length === 0 || node.arguments.some((one) => one.type === 'SpreadElement')) {
              context.report({ node, messageId: 'clock' })
            }
          },
        }
      },
    },

    'no-comment-only-catch': {
      meta: {
        type: 'problem',
        docs: { description: 'Forbid a catch block that holds no statement, a comment included' },
        messages: {
          emptyCatch:
            'A catch block with no statement hides the failure. Handle it: log it, return it as a result, or leave out the try.',
        },
      },
      create(context) {
        return {
          CatchClause(node) {
            if (node.body.body.length === 0) context.report({ node, messageId: 'emptyCatch' })
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
