import { describe, expect, it } from 'vitest'
import { formatQuery } from './formatQuery'

const LONG_CHAIN =
  'words("seed tray") && tags("garden") && path.startsWith("books/") || meaning("a cold frame for seedlings")'

describe('formatQuery', () => {
  it.each([
    [
      'a short query stays on one line',
      'words( "seed" )&&tags("garden")',
      'words("seed") && tags("garden")',
    ],
    [
      'a long chain breaks before each operator',
      LONG_CHAIN,
      [
        'words("seed tray")',
        '&& tags("garden")',
        '&& path.startsWith("books/")',
        '|| meaning("a cold frame for seedlings")',
      ].join('\n'),
    ],
    [
      'a group too long for its line opens over its own lines',
      '!(words("seed tray") || tags("garden") || meaning("a cold frame for the seedlings")) && tags("shed")',
      [
        '!(',
        '  words("seed tray")',
        '  || tags("garden")',
        '  || meaning("a cold frame for the seedlings")',
        ')',
        '&& tags("shed")',
      ].join('\n'),
    ],
    [
      'each stage of a broken query has its own line',
      'meaning("a cold frame for the seedlings in the greenhouse").rerank().top(5).sortBy("title")',
      [
        'meaning("a cold frame for the seedlings in the greenhouse")',
        '  .rerank()',
        '  .top(5)',
        '  .sortBy("title")',
      ].join('\n'),
    ],
    [
      'a comment stays on its line',
      '// beds\nwords("seed")   &&   tags("garden") // only the garden\n',
      '// beds\nwords("seed") && tags("garden") // only the garden',
    ],
    [
      'a comment inside a query breaks it after the comment',
      'words("seed") // seeds\n && tags("garden")',
      'words("seed") // seeds\n&& tags("garden")',
    ],
    [
      'a string with operators inside is kept as it is',
      'words("a  &&  b ,(x)")||tags(\'c || d\')',
      'words("a  &&  b ,(x)") || tags(\'c || d\')',
    ],
    [
      'an unclosed string leaves the query as it is',
      'words("seed && tags("garden"',
      'words("seed && tags("garden"',
    ],
    ['an unmatched bracket leaves the query as it is', 'words("seed"', 'words("seed"'],
  ])('%s', (_name, query, formatted) => {
    expect(formatQuery(query)).toBe(formatted)
  })

  it.each([
    LONG_CHAIN,
    '!(words("seed tray") || tags("garden") || meaning("a cold frame for the seedlings")) && tags("shed")',
    'meaning("a cold frame for the seedlings in the greenhouse").rerank().top(5).sortBy("title")',
    '// beds\nwords("seed")   &&   tags("garden") // only the garden\n',
  ])('gives the same text when it formats twice: %s', (query) => {
    const once = formatQuery(query)

    expect(formatQuery(once)).toBe(once)
  })
})
