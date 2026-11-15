/**
 * The search language in the query editor: its highlighting, the completion of the names of its
 * catalog, and the problems of a query as diagnostics at their line and column.
 */
import type { Completion, CompletionContext, CompletionResult } from '@codemirror/autocomplete'
import { HighlightStyle, StreamLanguage, type StringStream } from '@codemirror/language'
import type { Diagnostic } from '@codemirror/lint'
import type { Text } from '@codemirror/state'
import { tags as t } from '@lezer/highlight'
import { WORDS as words } from '../../words'

/**
 * A function, a variable or a stage of the catalog, or a method of a text; kind is "function",
 * "variable", "stage" or "method".
 */
export interface QueryFunction {
  readonly name: string
  readonly kind: string
  readonly summary: string
  readonly description: string
  readonly example: string
}

/**
 * A problem of the query, at its line and column counted from 1, and at the line and column of its last
 * character when it covers more than one.
 */
export interface QueryProblem {
  readonly line: number
  readonly column: number
  readonly endLine?: number
  readonly endColumn?: number
  readonly message: string
}

interface QueryScan {
  isAfterDot: boolean
}

const NAME = /^[A-Za-z_][\w]*/
const NUMBER = /^\d+(\.\d+)?/

const readString = (stream: StringStream, quote: string): string => {
  let isEscaped = false
  for (let next = stream.next(); next !== undefined; next = stream.next()) {
    if (next === quote && !isEscaped) break
    isEscaped = next === '\\' && !isEscaped
  }
  return 'string'
}

const readName = (stream: StringStream, scan: QueryScan): string => {
  const isStage = scan.isAfterDot
  scan.isAfterDot = false
  if (isStage) return 'propertyName'
  return stream.match(/^\s*\(/, false) ? 'function' : 'variableName'
}

const readComment = (stream: StringStream): string => {
  stream.skipToEnd()
  return 'comment'
}

const readCode = (stream: StringStream, scan: QueryScan): string | null => {
  if (stream.eatSpace()) return null
  const quote = stream.peek()
  if (quote === '"' || quote === "'") {
    stream.next()
    scan.isAfterDot = false
    return readString(stream, quote)
  }
  if (stream.match(NAME)) return readName(stream, scan)
  scan.isAfterDot = false
  if (stream.match(NUMBER)) return 'number'
  if (stream.match('&&') || stream.match('||') || stream.match('!')) return 'logicOperator'
  if (stream.eat('.')) {
    scan.isAfterDot = true
    return 'derefOperator'
  }
  stream.next()
  return 'punctuation'
}

const readToken = (stream: StringStream, scan: QueryScan): string | null =>
  stream.match('//') ? readComment(stream) : readCode(stream, scan)

/** The highlighting of strings, names, numbers, the operators and the stages of the search language. */
export const queryLanguage = StreamLanguage.define<QueryScan>({
  name: 'numen-query',
  startState: () => ({ isAfterDot: false }),
  token: readToken,
  tokenTable: { function: t.function(t.variableName) },
})

/** The colours of the tokens, as the notes editor paints code: the syntax tokens of the kit. */
export const queryHighlighting = HighlightStyle.define([
  {
    tag: [t.keyword, t.modifier, t.controlKeyword, t.operatorKeyword],
    color: 'var(--numen-syntax-keyword)',
  },
  {
    tag: [t.name, t.variableName, t.propertyName, t.attributeName],
    color: 'var(--numen-syntax-name)',
  },
  { tag: [t.typeName, t.className, t.namespace, t.tagName], color: 'var(--numen-syntax-type)' },
  { tag: [t.string, t.special(t.string), t.regexp], color: 'var(--numen-syntax-string)' },
  { tag: [t.number, t.bool, t.atom, t.literal], color: 'var(--numen-syntax-number)' },
  {
    tag: [t.comment, t.lineComment, t.blockComment, t.docComment],
    color: 'var(--numen-syntax-comment)',
    fontStyle: 'italic',
  },
  {
    tag: [t.operator, t.punctuation, t.bracket, t.separator, t.derefOperator],
    color: 'var(--numen-syntax-punctuation)',
  },
  { tag: t.meta, color: 'var(--numen-syntax-comment)' },
  { tag: t.processingInstruction, color: 'var(--numen-syntax-mark)' },
  { tag: t.escape, color: 'var(--numen-syntax-number)' },
  { tag: t.link, color: 'var(--numen-ring)' },
  { tag: t.invalid, color: 'var(--numen-alarm)' },
])

/** An entry of the completion list, with the function, variable, stage or method it names. */
export interface QueryCompletion extends Completion {
  readonly entry: QueryFunction
}

const TEXT_METHODS: readonly QueryFunction[] = words.textMethods.map((method) => ({
  ...method,
  kind: 'method',
}))

const getDotEntries = (
  functions: readonly QueryFunction[],
  before: string,
): readonly QueryFunction[] => {
  const name = /([A-Za-z_]\w*)\s*$/.exec(before)?.[1]
  const isAfterVariable = functions.some((fn) => fn.kind === 'variable' && fn.name === name)
  if (isAfterVariable || /["']\s*$/.test(before)) return TEXT_METHODS
  return functions.filter((fn) => fn.kind === 'stage')
}

const getOption = (entry: QueryFunction): QueryCompletion => ({
  label: entry.name,
  detail: entry.summary,
  entry,
})

/**
 * The completion of the name before the cursor: after a dot, the stages when a search comes before it
 * and the methods of a text when a variable or a string does; elsewhere the functions and variables. Each
 * entry says what it does in a few words and carries its description and example for its row.
 */
export function createQueryCompletion(
  getFunctions: () => readonly QueryFunction[],
): (context: CompletionContext) => CompletionResult | null {
  return (context) => {
    const word = context.matchBefore(/[A-Za-z_]\w*/)
    const from = word?.from ?? context.pos
    const before = context.state.sliceDoc(0, from)
    const isAfterDot = /\.\s*$/.test(before) && !/\d\.\s*$/.test(before)
    if (!word && !context.explicit && !isAfterDot) return null
    const functions = getFunctions()
    const entries = isAfterDot
      ? getDotEntries(functions, before.replace(/\.\s*$/, ''))
      : functions.filter((fn) => fn.kind !== 'stage')
    return { from, options: entries.map(getOption), validFor: /^\w*$/ }
  }
}

const getProblemRange = (doc: Text, from: number): { from: number; to: number } => {
  const line = doc.lineAt(from)
  const rest = line.text.slice(from - line.from)
  const word = NAME.exec(rest) ?? /^"[^"]*"?/.exec(rest)
  if (word) return { from, to: from + word[0].length }
  if (from < line.to) return { from, to: line.to }
  if (from > line.from) return { from: from - 1, to: from }
  return { from, to: Math.min(from + 1, doc.length) }
}

const getOffset = (doc: Text, lineNumber: number, column: number): number => {
  const line = doc.line(Math.min(Math.max(lineNumber, 1), doc.lines))
  return Math.min(line.from + Math.max(column - 1, 0), line.to)
}

const getEndOffset = (doc: Text, problem: QueryProblem): number => {
  const { endLine = 0, endColumn = 0 } = problem
  if (endLine < 1 || endColumn < 1) return 0
  return Math.min(getOffset(doc, endLine, endColumn) + 1, doc.length)
}

/**
 * The problems as diagnostics with the message of the core: a problem with an end from its line and column
 * through its last character; any other from its line and column to the end of the name or string there,
 * to the end of the line from any other place, and over the last character from the end of a line, so every
 * problem underlines at least one character of a query that has one.
 */
export function getQueryDiagnostics(doc: Text, problems: readonly QueryProblem[]): Diagnostic[] {
  return problems.map((problem) => {
    const start = getOffset(doc, problem.line, problem.column)
    const end = getEndOffset(doc, problem)
    const { from, to } = end > start ? { from: start, to: end } : getProblemRange(doc, start)
    return { from, to, severity: 'error', message: problem.message }
  })
}
