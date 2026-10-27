import { describe, expect, it } from 'vitest'
import { getUrls } from './uriList'

describe('the addresses of a uri list', () => {
  it('are its lines, past the comments and the blank lines, across CRLF and LF', () => {
    const list =
      '# comment\r\nhttps://one.test/\r\n\r\n# another\nhttps://two.test/path\n  https://three.test  \n'
    expect(getUrls(list)).toEqual([
      'https://one.test/',
      'https://two.test/path',
      'https://three.test',
    ])
  })

  it('are none where the list holds comments alone', () => {
    expect(getUrls('# only\n# comments\n\n')).toEqual([])
  })
})
