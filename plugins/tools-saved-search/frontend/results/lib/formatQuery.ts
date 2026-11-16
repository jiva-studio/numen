/**
 * The layout of a query: one line when it fits in 80 columns; otherwise a line for each operand of the
 * outer `&&` and `||` with the operator first, a line for each stage, and a group too long for its line
 * opened over its own lines indented by 2. Strings and comments are kept as they are.
 */

type TokenKind =
  'string' | 'number' | 'name' | 'dot' | 'comma' | 'open' | 'close' | 'binary' | 'not'

interface Token {
  readonly kind: TokenKind
  readonly text: string
  readonly leading: string[]
  trailing: string | null
}

interface Lexed {
  readonly tokens: Token[]
  readonly tail: string[]
}

const WIDTH = 80
const INDENT = 2
const NAME = /^[A-Za-z_]\w*/
const NUMBER = /^(?:0[xX][\da-fA-F]+|\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/
const STRING_PREFIX = /^(?:[rR][bB]?|[bB][rR]?)?(?=["'])/
const BINARY = ['&&', '||', '==', '!=', '<=', '>=', '<', '>']
const PAIRS: Record<string, string> = { '(': ')', '[': ']' }

const readStringEnd = (query: string, start: number): number => {
  const quote = query[start]
  let isEscaped = false
  for (let index = start + 1; index < query.length; index++) {
    const char = query[index]
    if (char === '\n') return -1
    if (char === quote && !isEscaped) return index + 1
    isEscaped = char === '\\' && !isEscaped
  }
  return -1
}

const readSymbol = (rest: string): { kind: TokenKind; text: string } | null => {
  const operator = BINARY.find((op) => rest.startsWith(op))
  if (operator) return { kind: 'binary', text: operator }
  const char = rest[0] ?? ''
  if (char === '!') return { kind: 'not', text: char }
  if (char === '.') return { kind: 'dot', text: char }
  if (char === ',') return { kind: 'comma', text: char }
  if (char === '(' || char === '[') return { kind: 'open', text: char }
  if (char === ')' || char === ']') return { kind: 'close', text: char }
  return null
}

const readWord = (query: string, index: number): { kind: TokenKind; text: string } | null => {
  const rest = query.slice(index)
  const prefix = STRING_PREFIX.exec(rest)?.[0]
  if (prefix !== undefined) {
    if (/^("""|''')/.test(rest.slice(prefix.length))) return null
    const end = readStringEnd(query, index + prefix.length)
    return end < 0 ? null : { kind: 'string', text: query.slice(index, end) }
  }
  const number = NUMBER.exec(rest)?.[0]
  if (number) return { kind: 'number', text: number }
  const name = NAME.exec(rest)?.[0]
  if (name) return { kind: 'name', text: name }
  return readSymbol(rest)
}

const attachComment = (
  comment: string,
  last: Token | undefined,
  isLineStart: boolean,
  pending: string[],
): void => {
  if (last && !isLineStart) last.trailing = comment
  else pending.push(comment)
}

const lexQuery = (query: string): Lexed | null => {
  const tokens: Token[] = []
  let pending: string[] = []
  let isLineStart = true
  let index = 0
  while (index < query.length) {
    const rest = query.slice(index)
    const space = /^\s+/.exec(rest)?.[0]
    if (space) {
      if (space.includes('\n')) isLineStart = true
      index += space.length
      continue
    }
    if (rest.startsWith('//')) {
      const comment = (/^[^\n]*/.exec(rest)?.[0] ?? '').trimEnd()
      attachComment(comment, tokens.at(-1), isLineStart, pending)
      index += comment.length
      isLineStart = false
      continue
    }
    const word = readWord(query, index)
    if (!word) return null
    tokens.push({ ...word, leading: pending, trailing: null })
    pending = []
    index += word.text.length
    isLineStart = false
  }
  return { tokens, tail: pending }
}

const isBalanced = (tokens: readonly Token[]): boolean => {
  const open: string[] = []
  for (const token of tokens) {
    if (token.kind === 'open') open.push(PAIRS[token.text]!)
    if (token.kind === 'close' && open.pop() !== token.text) return false
  }
  return open.length === 0
}

const NO_SPACE_BEFORE: ReadonlySet<TokenKind> = new Set(['comma', 'close', 'dot'])
const NO_SPACE_AFTER: ReadonlySet<TokenKind> = new Set(['open', 'dot', 'not'])
const CALLEES: ReadonlySet<TokenKind> = new Set(['name', 'close'])

const getSpace = (before: Token, after: Token): string => {
  if (NO_SPACE_BEFORE.has(after.kind) || NO_SPACE_AFTER.has(before.kind)) return ''
  return after.kind === 'open' && CALLEES.has(before.kind) ? '' : ' '
}

const getFlat = (tokens: readonly Token[], indent: string): string => {
  let text = ''
  tokens.forEach((token, index) => {
    const before = tokens[index - 1]
    if (before) {
      if (before.trailing !== null) text += '\n' + indent
      else if (token.leading.length === 0) text += getSpace(before, token)
      for (const comment of token.leading) text += '\n' + indent + comment + '\n' + indent
    }
    text += token.text
    if (token.trailing !== null) text += ' ' + token.trailing
  })
  return text
}

const hasInnerComments = (tokens: readonly Token[]): boolean =>
  tokens.some(
    (token, index) =>
      (index > 0 && token.leading.length > 0) ||
      (index < tokens.length - 1 && token.trailing !== null),
  )

const isFitting = (tokens: readonly Token[], indent: number): boolean =>
  !hasInnerComments(tokens) && indent + getFlat(tokens, '').length <= WIDTH

const getFlatLines = (tokens: readonly Token[], indent: number): string[] => {
  const pad = ' '.repeat(indent)
  const comments = tokens[0]?.leading.map((comment) => pad + comment) ?? []
  return [...comments, pad + getFlat(tokens, pad)]
}

const getDepths = (tokens: readonly Token[]): number[] => {
  let depth = 0
  return tokens.map((token) => {
    if (token.kind === 'close') depth--
    const current = depth
    if (token.kind === 'open') depth++
    return current
  })
}

const splitTop = (
  tokens: readonly Token[],
  isSeparator: (token: Token) => boolean,
  side: 'after' | 'before',
): Token[][] => {
  const depths = getDepths(tokens)
  const parts: Token[][] = [[]]
  tokens.forEach((token, index) => {
    const isSplit = depths[index] === 0 && isSeparator(token)
    if (isSplit && side === 'before' && parts.at(-1)!.length > 0) parts.push([])
    parts.at(-1)!.push(token)
    if (isSplit && side === 'after') parts.push([])
  })
  return parts.filter((part) => part.length > 0)
}

const isStageStart = (tokens: readonly Token[], index: number): boolean =>
  tokens[index]?.kind === 'dot' &&
  tokens[index - 1]?.kind === 'close' &&
  tokens[index + 1]?.kind === 'name' &&
  tokens[index + 2]?.kind === 'open'

const splitStages = (tokens: readonly Token[]): Token[][] => {
  const depths = getDepths(tokens)
  const parts: Token[][] = [[]]
  tokens.forEach((token, index) => {
    if (depths[index] === 0 && isStageStart(tokens, index)) parts.push([])
    parts.at(-1)!.push(token)
  })
  return parts
}

const findClose = (tokens: readonly Token[], open: number): number => {
  let depth = 0
  for (let index = open; index < tokens.length; index++) {
    if (tokens[index]!.kind === 'open') depth++
    if (tokens[index]!.kind === 'close' && --depth === 0) return index
  }
  return -1
}

const getGroupLines = (tokens: readonly Token[], indent: number): string[] => {
  if (isFitting(tokens, indent)) return getFlatLines(tokens, indent)
  const open = tokens.findIndex((token) => token.kind === 'open')
  const close = open < 0 ? -1 : findClose(tokens, open)
  if (close < 0 || close === open + 1) return getFlatLines(tokens, indent)
  return [
    ...getFlatLines(tokens.slice(0, open + 1), indent),
    ...getPieceLines(tokens.slice(open + 1, close), indent + INDENT),
    ...getFlatLines(tokens.slice(close), indent),
  ]
}

const getChainLines = (tokens: readonly Token[], indent: number): string[] => {
  const [head, ...stages] = splitStages(tokens)
  if (stages.length === 0) return getGroupLines(tokens, indent)
  return [
    ...getGroupLines(head!, indent),
    ...stages.flatMap((stage) => getGroupLines(stage, indent + INDENT)),
  ]
}

const getBrokenLines = (tokens: readonly Token[], indent: number): string[] => {
  const items = splitTop(tokens, (token) => token.kind === 'comma', 'after')
  if (items.length > 1) return items.flatMap((item) => getPieceLines(item, indent))
  const operands = splitTop(tokens, (token) => token.text === '&&' || token.text === '||', 'before')
  return operands.flatMap((operand) => getChainLines(operand, indent))
}

const getPieceLines = (tokens: readonly Token[], indent: number): string[] =>
  isFitting(tokens, indent) ? getFlatLines(tokens, indent) : getBrokenLines(tokens, indent)

/** The query laid out, or the query as it is when it cannot be read: an unclosed string, an unknown sign or an unmatched bracket. */
export function formatQuery(query: string): string {
  const lexed = lexQuery(query)
  if (!lexed || lexed.tokens.length === 0 || !isBalanced(lexed.tokens)) return query
  const lines = getPieceLines(lexed.tokens, 0)
  return [...lines, ...lexed.tail].join('\n')
}
