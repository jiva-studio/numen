// The component rules of tools/lint-layers/base-eslint.mjs, copied for the plugins: a plugin configuration
// imports nothing outside its directory, and every plugin carries a byte-identical copy of this file.

/** The canonical order of the section banners of a component script. */
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

/** Returns the name of a `--- Name ---` banner comment, or null when the comment is not a banner. */
function getBannerName(comment) {
  const match = /^-{3,}([^-]+)-{3,}$/.exec(comment.trim())
  const name = match?.[1].trim()
  return name && /^[A-Za-z &]+$/.test(name) ? name : null
}

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

/**
 * max-inline-props: a component declares at most `max` props inline.
 * vue-script-sections: a long script is grouped by section banners in the canonical order.
 * max-script-lines: a script holds at most `max` lines of code, comments and blank lines aside.
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

            const codeLines = countScriptCodeLines(context.sourceCode, script)

            const banners = []
            for (const comment of comments) {
              const name = getBannerName(comment.value)
              if (name !== null) {
                banners.push({
                  node: comment,
                  name,
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
  },
}
