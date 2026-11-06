/**
 * The reading of a when clause into a tree: the keys of WHEN_KEYS combined with !, && and || and
 * parentheses; ! binds tightest, then &&, then ||. The Go reader of the manifest follows the same
 * rules, and both are tested against libs/protocol/when.json.
 */
import { asFailure, asValue, WHEN_KEYS, type Result, type WhenKey } from '@numen/plugin-sdk'

/** A condition read once and evaluated in many contexts. */
export type WhenExpression =
  | { readonly operator: 'key'; readonly key: WhenKey }
  | { readonly operator: '!'; readonly operand: WhenExpression }
  | {
      readonly operator: '&&' | '||'
      readonly left: WhenExpression
      readonly right: WhenExpression
    }

const TOKEN = /\s*(&&|\|\||!|\(|\)|[A-Za-z][A-Za-z0-9]*)/y
const SYNTAX = 'not an expression of keys, !, &&, || and parentheses'

function split(text: string): Result<string[], string> {
  const tokens: string[] = []
  TOKEN.lastIndex = 0
  while (text.slice(TOKEN.lastIndex).trim() !== '') {
    const start = TOKEN.lastIndex
    const match = TOKEN.exec(text)
    if (!match?.[1]) return asFailure(`${SYNTAX}: "${text.slice(start).trim()}"`)
    tokens.push(match[1])
  }
  return asValue(tokens)
}

const isKey = (token: string): token is WhenKey => WHEN_KEYS.some((key) => key === token)

/** Reads the tokens; throws the text of the error, which parseWhen returns as a failure. */
function read(tokens: readonly string[]): WhenExpression {
  let next = 0
  const readBinary = (operator: '&&' | '||', readOperand: () => WhenExpression): WhenExpression => {
    let expression = readOperand()
    while (tokens[next] === operator) {
      next += 1
      expression = { operator, left: expression, right: readOperand() }
    }
    return expression
  }
  const readUnary = (): WhenExpression => {
    const token = tokens[next]
    next += 1
    if (token === '!') return { operator: '!', operand: readUnary() }
    if (token === '(') {
      const inner = readOr()
      if (tokens[next] !== ')') throw new Error(`${SYNTAX}: a parenthesis is not closed`)
      next += 1
      return inner
    }
    if (token === undefined) throw new Error(`${SYNTAX}: it ends where a key is expected`)
    if (isKey(token)) return { operator: 'key', key: token }
    if (/^[A-Za-z]/.test(token)) {
      throw new Error(`"${token}" is none of the keys ${WHEN_KEYS.join(', ')}`)
    }
    throw new Error(`${SYNTAX}: "${token}" where a key is expected`)
  }
  const readAnd = () => readBinary('&&', readUnary)
  const readOr = (): WhenExpression => readBinary('||', readAnd)
  const expression = readOr()
  if (next < tokens.length) throw new Error(`${SYNTAX}: "${tokens[next]}" after the end`)
  return expression
}

/** Reads a condition into a tree; a failure for a malformed condition or an unknown key. */
export function parseWhen(text: string): Result<WhenExpression, string> {
  const tokens = split(text)
  if (!tokens.ok) return tokens
  try {
    return asValue(read(tokens.value))
  } catch (error) {
    return asFailure((error as Error).message)
  }
}

/** The keys the condition names, each once, in the order they first appear. */
export function listWhenKeys(expression: WhenExpression): WhenKey[] {
  const collect = (one: WhenExpression): WhenKey[] => {
    if (one.operator === 'key') return [one.key]
    if (one.operator === '!') return collect(one.operand)
    return [...collect(one.left), ...collect(one.right)]
  }
  return [...new Set(collect(expression))]
}
