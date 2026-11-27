/**
 * What the syntax trees of TypeScript files and of the script blocks of Vue files show: their
 * string literals, network client calls, test cases, stories and imported modules.
 */
import ts from 'typescript'

const TEST_NAMES = new Set(['it', 'test'])

const TEST_MODIFIERS = new Set(['each', 'only', 'skip', 'todo', 'concurrent', 'fails', 'sequential'])

const CALLED_CLIENTS = new Set(['fetch'])

const CONSTRUCTED_CLIENTS = new Set(['WebSocket', 'EventSource'])

const UNTARGETED_CLIENTS = new Set(['XMLHttpRequest'])

/** Helpers of `@numen/plugin-sdk` that build a URL of the host's own server for the plugin. */
const HOST_URL_HELPERS = new Set(['getAssetUrl'])

const SCRIPT_OPEN = '<script'

const SCRIPT_CLOSE = '</script>'

/** @param {string} text */
export const parseTypeScript = (text) =>
  ts.createSourceFile('file.ts', text, ts.ScriptTarget.Latest, false, ts.ScriptKind.TS)

const isTagNameEnd = (char) => char === '>' || char === '/' || (char !== undefined && char.trim() === '')

/** The texts of the `<script>` blocks of a Vue single-file component. */
export const readVueScripts = (text) => {
  const scripts = []
  let at = text.indexOf(SCRIPT_OPEN)
  while (at !== -1) {
    const tagEnd = text.indexOf('>', at)
    if (tagEnd === -1) break
    if (!isTagNameEnd(text[at + SCRIPT_OPEN.length])) {
      at = text.indexOf(SCRIPT_OPEN, at + SCRIPT_OPEN.length)
      continue
    }
    const close = text.indexOf(SCRIPT_CLOSE, tagEnd)
    scripts.push(text.slice(tagEnd + 1, close === -1 ? text.length : close))
    if (close === -1) break
    at = text.indexOf(SCRIPT_OPEN, close + SCRIPT_CLOSE.length)
  }
  return scripts
}

/** The syntax trees of a TypeScript file, or of the script blocks of a Vue file. */
export const parseSourceFile = (file, text) =>
  (file.endsWith('.vue') ? readVueScripts(text) : [text]).map(parseTypeScript)

const visitNodes = (sourceFile, visit) => {
  const walk = (node) => {
    visit(node)
    ts.forEachChild(node, walk)
  }
  walk(sourceFile)
}

/**
 * The texts of the string literals and template literals of a tree; of a template with
 * substitutions, the text before the first one.
 */
export const listStringTexts = (sourceFile) => {
  const texts = []
  visitNodes(sourceFile, (node) => {
    if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node) || ts.isTemplateHead(node)) {
      texts.push(node.text)
    }
  })
  return texts
}

/**
 * What a client call is given as its target: `helper` for a URL a host helper builds, `literal`
 * with the text a string or template literal starts with, `unknown` for anything else.
 *
 * @returns {{ kind: 'helper' } | { kind: 'literal', text: string } | { kind: 'unknown' }}
 */
const readTarget = (argument) => {
  if (argument === undefined) return { kind: 'unknown' }
  if (
    ts.isCallExpression(argument) &&
    ts.isIdentifier(argument.expression) &&
    HOST_URL_HELPERS.has(argument.expression.text)
  ) {
    return { kind: 'helper' }
  }
  if (ts.isStringLiteral(argument) || ts.isNoSubstitutionTemplateLiteral(argument)) {
    return { kind: 'literal', text: argument.text }
  }
  if (ts.isTemplateExpression(argument) && argument.head.text !== '') {
    return { kind: 'literal', text: argument.head.text }
  }
  return { kind: 'unknown' }
}

/**
 * The network client calls of a tree: `fetch(…)` of the global function, and `new WebSocket(…)`,
 * `new EventSource(…)` and `new XMLHttpRequest()`, each with the target its first argument gives.
 *
 * @returns {{ name: string, target: ReturnType<typeof readTarget> }[]}
 */
export const listNetworkCalls = (sourceFile) => {
  const calls = []
  visitNodes(sourceFile, (node) => {
    const isCall = ts.isCallExpression(node)
    if (!isCall && !ts.isNewExpression(node)) return
    if (!ts.isIdentifier(node.expression)) return
    const name = node.expression.text
    if (isCall ? CALLED_CLIENTS.has(name) : CONSTRUCTED_CLIENTS.has(name)) {
      calls.push({ name, target: readTarget(node.arguments?.[0]) })
    } else if (!isCall && UNTARGETED_CLIENTS.has(name)) {
      calls.push({ name, target: { kind: 'unknown' } })
    }
  })
  return calls
}

/** The module specifiers a tree imports or re-exports from, dynamic imports of a literal included. */
export const listImportedModules = (sourceFile) => {
  const modules = []
  visitNodes(sourceFile, (node) => {
    if ((ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) && node.moduleSpecifier) {
      if (ts.isStringLiteral(node.moduleSpecifier)) modules.push(node.moduleSpecifier.text)
    } else if (
      ts.isCallExpression(node) &&
      node.expression.kind === ts.SyntaxKind.ImportKeyword &&
      node.arguments[0] !== undefined &&
      ts.isStringLiteralLike(node.arguments[0])
    ) {
      modules.push(node.arguments[0].text)
    }
  })
  return modules
}

const isTestCallee = (callee) => {
  let node = callee
  while (ts.isPropertyAccessExpression(node)) {
    if (!TEST_MODIFIERS.has(node.name.text)) return false
    node = node.expression
  }
  return ts.isIdentifier(node) && TEST_NAMES.has(node.text)
}

/** The test cases of a tree: calls of `it` and `test` and their modifiers; an `it.each` table counts once. */
export const countTestCases = (sourceFile) => {
  let count = 0
  visitNodes(sourceFile, (node) => {
    if (ts.isCallExpression(node) && isTestCallee(node.expression)) count += 1
  })
  return count
}

const isExported = (statement) =>
  statement.modifiers?.some((modifier) => modifier.kind === ts.SyntaxKind.ExportKeyword) ?? false

const isCapitalized = (name) => name[0] >= 'A' && name[0] <= 'Z'

/** The stories of a tree: its exported variables whose names start with a capital letter. */
export const countStories = (sourceFile) =>
  sourceFile.statements
    .filter((statement) => ts.isVariableStatement(statement) && isExported(statement))
    .flatMap((statement) => statement.declarationList.declarations)
    .filter((declaration) => ts.isIdentifier(declaration.name) && isCapitalized(declaration.name.text))
    .length
