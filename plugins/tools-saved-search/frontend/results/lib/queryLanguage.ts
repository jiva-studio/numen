/**
 * The search language in the query editor: its highlighting, the completion of the names of its
 * catalog, and the problems of a query as diagnostics at their line and column.
 */
import type { Completion, CompletionContext, CompletionResult } from '@codemirror/autocomplete'
import { HighlightStyle, StreamLanguage, type StringStream } from '@codemirror/language'
import type { Diagnostic } from '@codemirror/lint'
import type { Text } from '@codemirror/state'
import { tags as t } from '@lezer/highlight'

/** A function, a variable or a stage of the catalog; kind is "function", "variable" or "stage". */
export interface QueryFunction {
  readonly name: string
  readonly kind: string
  readonly signatures: readonly string[]
  readonly description: string
  readonly example: string
}

/** A problem of the query, at its line and column counted from 1. */
export interface QueryProblem {
  readonly line: number
  readonly column: number
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

const readToken = (stream: StringStream, scan: QueryScan): string | null => {
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

/** The highlighting of strings, names, numbers, the operators and the stages of the search language. */
export const queryLanguage = StreamLanguage.define<QueryScan>({
  name: 'numen-query',
  startState: () => ({ isAfterDot: false }),
  token: readToken,
  tokenTable: { function: t.function(t.variableName) },
})

/** The colours of the tokens, from the syntax tokens of the kit. */
export const queryHighlighting = HighlightStyle.define([
  { tag: t.string, color: 'var(--numen-syntax-string)' },
  { tag: t.number, color: 'var(--numen-syntax-number)' },
  { tag: t.function(t.variableName), color: 'var(--numen-syntax-name)' },
  { tag: t.variableName, color: 'var(--numen-syntax-type)' },
  { tag: t.propertyName, color: 'var(--numen-syntax-keyword)' },
  { tag: t.logicOperator, color: 'var(--numen-syntax-keyword)', fontWeight: '600' },
  { tag: [t.punctuation, t.derefOperator], color: 'var(--numen-syntax-punctuation)' },
])

const getOption = (fn: QueryFunction): Completion => ({
  label: fn.name,
  type: fn.kind === 'stage' ? 'method' : fn.kind,
  detail: fn.signatures.join('  '),
  info: [fn.description, fn.example].filter((part) => part !== '').join('\n\n'),
})

/**
 * The completion of the name before the cursor: the stages after a dot, the functions and variables
 * elsewhere, with the signature, the description and the example of each.
 */
export function createQueryCompletion(
  getFunctions: () => readonly QueryFunction[],
): (context: CompletionContext) => CompletionResult | null {
  return (context) => {
    const word = context.matchBefore(/[A-Za-z_]\w*/)
    const isAfterDot =
      context.state.sliceDoc((word?.from ?? context.pos) - 1, word?.from ?? context.pos) === '.'
    if (!word && !context.explicit && !isAfterDot) return null
    const from = word?.from ?? context.pos
    const options = getFunctions()
      .filter((fn) => (fn.kind === 'stage') === isAfterDot)
      .map(getOption)
    return { from, options, validFor: /^\w*$/ }
  }
}

const getProblemEnd = (doc: Text, from: number): number => {
  const line = doc.lineAt(from)
  const rest = line.text.slice(from - line.from)
  const word = NAME.exec(rest) ?? /^"[^"]*"?/.exec(rest)
  if (word) return from + word[0].length
  return Math.min(from + 1, line.to)
}

/** The problems as diagnostics: each from its line and column to the end of the name or string there. */
export function getQueryDiagnostics(doc: Text, problems: readonly QueryProblem[]): Diagnostic[] {
  return problems.map((problem) => {
    const lineNumber = Math.min(Math.max(problem.line, 1), doc.lines)
    const line = doc.line(lineNumber)
    const from = Math.min(line.from + Math.max(problem.column - 1, 0), line.to)
    return { from, to: getProblemEnd(doc, from), severity: 'error', message: problem.message }
  })
}
