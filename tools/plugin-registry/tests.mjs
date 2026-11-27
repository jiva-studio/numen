/**
 * The tests a plugin folder holds, counted from its source without running them: Go test and fuzz
 * functions in `*_test.go`, TypeScript test cases in `*.test.ts` and stories in `*.stories.ts`,
 * which run as browser tests; and the Vue components no story shows. TypeScript is read from its
 * syntax tree; Go, which has no parser here, with its comments and literals blanked first.
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { dirname, join, relative, resolve, sep } from 'node:path'
import ts from 'typescript'

const SKIPPED_FOLDERS = new Set(['node_modules', 'dist', 'testdata', '.git'])

const GO_TEST = /^func (?:Test|Fuzz)\w*\(\s*\w+\s+\*testing\.[TF]\s*\)/gm

const TS_TEST_NAMES = new Set(['it', 'test'])

const TS_TEST_MODIFIERS = new Set(['each', 'only', 'skip', 'todo', 'concurrent', 'fails', 'sequential'])

const findTokenEnd = (text, token, from) => {
  const found = text.indexOf(token, from)
  return found === -1 ? text.length : found + token.length
}

const findQuoteEnd = (text, quote, from) => {
  let end = from
  while (end < text.length && text[end] !== quote && text[end] !== '\n') {
    end += text[end] === '\\' ? 2 : 1
  }
  return Math.min(end + 1, text.length)
}

const findGoTokenEnd = (text, at) => {
  const pair = text.slice(at, at + 2)
  if (pair === '//') {
    const newline = text.indexOf('\n', at)
    return newline === -1 ? text.length : newline
  }
  if (pair === '/*') return findTokenEnd(text, '*/', at + 2)
  if (text[at] === '`') return findTokenEnd(text, '`', at + 1)
  if (text[at] === '"' || text[at] === "'") return findQuoteEnd(text, text[at], at + 1)
  return null
}

const blankGoCommentsAndLiterals = (text) => {
  let out = ''
  let at = 0
  while (at < text.length) {
    const end = findGoTokenEnd(text, at)
    if (end === null) {
      out += text[at]
      at += 1
      continue
    }
    out += text.slice(at, end).replace(/[^\n]/g, ' ')
    at = end
  }
  return out
}

const countGoTests = (text) => blankGoCommentsAndLiterals(text).match(GO_TEST)?.length ?? 0

const parseTypeScript = (text) =>
  ts.createSourceFile('file.ts', text, ts.ScriptTarget.Latest, false, ts.ScriptKind.TS)

const isTestCallee = (callee) => {
  let node = callee
  while (ts.isPropertyAccessExpression(node)) {
    if (!TS_TEST_MODIFIERS.has(node.name.text)) return false
    node = node.expression
  }
  return ts.isIdentifier(node) && TS_TEST_NAMES.has(node.text)
}

const countTypeScriptTests = (text) => {
  let count = 0
  const visit = (node) => {
    if (ts.isCallExpression(node) && isTestCallee(node.expression)) count += 1
    ts.forEachChild(node, visit)
  }
  visit(parseTypeScript(text))
  return count
}

const isExported = (statement) =>
  statement.modifiers?.some((modifier) => modifier.kind === ts.SyntaxKind.ExportKeyword) ?? false

const countStories = (text) =>
  parseTypeScript(text)
    .statements.filter((statement) => ts.isVariableStatement(statement) && isExported(statement))
    .flatMap((statement) => statement.declarationList.declarations)
    .filter((declaration) => ts.isIdentifier(declaration.name) && /^[A-Z]/.test(declaration.name.text))
    .length

const COUNTERS = [
  { suffix: '_test.go', kind: 'go', count: countGoTests },
  { suffix: '.test.ts', kind: 'typescript', count: countTypeScriptTests },
  { suffix: '.stories.ts', kind: 'stories', count: countStories },
]

const listFiles = (dir) =>
  readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    if (entry.isDirectory()) {
      return SKIPPED_FOLDERS.has(entry.name) ? [] : listFiles(join(dir, entry.name))
    }
    return entry.isFile() ? [join(dir, entry.name)] : []
  })

/**
 * The tests of the folder `dir`, by kind: Go functions, TypeScript cases (an `it.each` table
 * counts once) and stories.
 *
 * @param {string} dir
 * @returns {{ go: number, typescript: number, stories: number }}
 */
export const countTests = (dir) => {
  const counts = { go: 0, typescript: 0, stories: 0 }
  for (const file of listFiles(dir)) {
    const counter = COUNTERS.find((one) => file.endsWith(one.suffix))
    if (!counter) continue
    counts[counter.kind] += counter.count(readFileSync(file, 'utf8'))
  }
  return counts
}

const VUE_IMPORT = /(?:from\s+|import\s*\(\s*)["'](\.{1,2}\/[^"']+\.vue)["']/g

/**
 * The Vue components of the folder `frontend` of `dir` and the ones no story shows: a component
 * has a story when a `*.stories.ts` of the plugin imports its file by a relative path. Null when
 * the plugin has no `frontend` folder.
 *
 * @param {string} dir - the plugin folder
 * @returns {{ components: string[], unstoried: string[] } | null} paths relative to `dir`
 */
export const listUnstoriedComponents = (dir) => {
  const frontend = join(dir, 'frontend')
  if (!existsSync(frontend)) return null
  const files = listFiles(dir)
  const storied = new Set(
    files
      .filter((file) => file.endsWith('.stories.ts'))
      .flatMap((file) =>
        [...readFileSync(file, 'utf8').matchAll(VUE_IMPORT)].map((match) =>
          resolve(dirname(file), match[1]),
        ),
      ),
  )
  const components = files.filter((file) => file.startsWith(frontend + sep) && file.endsWith('.vue')).sort()
  const toName = (file) => relative(dir, file).split(sep).join('/')
  return {
    components: components.map(toName),
    unstoried: components.filter((file) => !storied.has(file)).map(toName),
  }
}
