/**
 * A problem of the query in plain words: the messages of the parser of the core that name its grammar
 * (extraneous input, mismatched input, missing, no viable alternative, token recognition error) said
 * as a person reads the query; any other message as the core wrote it.
 */
import { WORDS as words } from '../../words'

const END = '<EOF>'
const SYNTAX_PREFIX = /^Syntax error:\s*/

const unquote = (token: string): string => token.replace(/^'(.*)'$/, '$1')

const getUnexpected = (token: string, expected: string): string =>
  expected === END ? words.problem.unexpectedAtEnd(token) : words.problem.unexpected(token)

const getMissing = (token: string, at: string): string =>
  at === END ? words.problem.missingAtEnd(unquote(token)) : words.problem.missing(unquote(token), at)

const RULES: readonly [RegExp, (...parts: string[]) => string][] = [
  [/^extraneous input '(.+)' expecting (.+)$/, getUnexpected],
  [/^mismatched input '(.+)' expecting (.+)$/, getUnexpected],
  [/^missing (.+) at '(.+)'$/, getMissing],
  [/^no viable alternative at input '(.+)'$/, (token) => words.problem.unreadable(token)],
  [/^token recognition error at: '(.+)'$/, (token) => words.problem.notAllowed(token)],
  [/^unexpected end of input$/, () => words.problem.endsEarly],
]

/** The message of a problem in plain words, or the message itself when it is not one of the parser's. */
export function getProblemText(message: string): string {
  const bare = message.replace(SYNTAX_PREFIX, '')
  for (const [pattern, say] of RULES) {
    const match = pattern.exec(bare)
    if (match) return say(...match.slice(1))
  }
  return message
}
