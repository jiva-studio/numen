/** The messages of the parser of the core in plain words, and any other message as it came. */
import { describe, expect, it } from 'vitest'
import { getProblemText } from './problemText'

describe('getProblemText', () => {
  it.each([
    ["extraneous input 'e' expecting <EOF>", 'Unexpected “e”: the query should end here'],
    [
      "Syntax error: mismatched input ')' expecting <EOF>",
      'Unexpected “)”: the query should end here',
    ],
    ["mismatched input '&&' expecting {'(', IDENTIFIER}", 'Unexpected “&&” here'],
    ["missing ')' at '<EOF>'", 'Missing “)” at the end of the query'],
    ["missing ')' at '&&'", 'Missing “)” before “&&”'],
    ['no viable alternative at input \'words("gale"\'', 'Cannot read the query at “words("gale"”'],
    ["token recognition error at: '#'", '“#” is not allowed in a query'],
    ['Syntax error: unexpected end of input', 'The query ends too early'],
  ])('says %s in plain words', (message, text) => {
    expect(getProblemText(message)).toBe(text)
  })

  it('keeps a message that is not one of the parser', () => {
    expect(getProblemText('unknown function meanin (did you mean meaning?)')).toBe(
      'unknown function meanin (did you mean meaning?)',
    )
  })
})
